package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
	"github.com/thalesraymond/go-http-server/internal/handler"
	"github.com/thalesraymond/go-http-server/internal/session"

	_ "github.com/lib/pq"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := handler.NewLogger()

	// .env is optional — production environments inject variables directly.
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, relying on environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return errors.New("DB_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error("Error closing database connection", err)
		}
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	secret := os.Getenv("SECRET_KEY")
	polkaKey := os.Getenv("POLKA_KEY")
	authenticator := auth.NewRealAuthenticator(secret)
	dbQuerier := database.New(db)
	handshake := handler.NewAuthHandshake(logger, authenticator, polkaKey)
	admin := handler.NewAdmin(dbQuerier, logger)

	serverMux := http.NewServeMux()

	serverMux.Handle("/app/", admin.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./")))))

	serverMux.HandleFunc("GET /api/healthz", handler.HealthzHandler)

	serverMux.HandleFunc("GET /admin/metrics", admin.HandlerMetrics)

	serverMux.HandleFunc("POST /admin/reset", admin.HandlerReset)

	userHandler := handler.NewUserHandler(dbQuerier, logger, authenticator, handshake)
	userHandler.RegisterRoutes(serverMux)

	sessionHandler := handler.NewSessionHandler(dbQuerier, logger, authenticator, session.New(dbQuerier, authenticator))
	sessionHandler.RegisterRoutes(serverMux)

	chirpHandler := handler.NewChirpHandler(dbQuerier, logger, handshake)
	chirpHandler.RegisterRoutes(serverMux)

	polkaHandler := handler.NewPolkaHandler(dbQuerier, logger, handshake)
	polkaHandler.RegisterRoutes(serverMux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	logger.Info("Shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("Server stopped")
	return nil
}
