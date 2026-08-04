package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
	"github.com/thalesraymond/go-http-server/internal/handler"
	"github.com/thalesraymond/go-http-server/internal/session"

	_ "github.com/lib/pq"
)

func main() {

	serverMux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	err := godotenv.Load()

	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}

	defer func() {
		err := db.Close()
		if err != nil {
			fmt.Println("Failed to close database:", err)
		}
	}()

	secret := os.Getenv("SECRET_KEY")
	polkaKey := os.Getenv("POLKA_KEY")
	authenticator := auth.NewRealAuthenticator(secret)
	dbQuerier := database.New(db)
	logger := handler.NewLogger()
	handshake := handler.NewAuthHandshake(logger, authenticator, polkaKey)
	admin := handler.NewAdmin(dbQuerier, logger)

	serverMux.Handle("/app/", admin.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./")))))

	serverMux.HandleFunc("GET /api/healthz", handler.HealthzHandler)

	serverMux.HandleFunc("GET /admin/metrics", admin.HandlerMetrics)

	serverMux.HandleFunc("POST /admin/reset", admin.HandlerReset)

	userHandler := handler.NewUserHandler(dbQuerier, logger, authenticator, handshake)
	userHandler.RegisterRoutes(serverMux)

	sessionHandler := handler.NewSessionHandler(dbQuerier, logger, authenticator, session.New(dbQuerier, authenticator))
	sessionHandler.RegisterRoutes(serverMux)

	chiprHandler := handler.NewChirpHandler(dbQuerier, logger, handshake)
	chiprHandler.RegisterRoutes(serverMux)

	polkaHandler := handler.NewPolkaHandler(dbQuerier, logger, handshake)
	polkaHandler.RegisterRoutes(serverMux)

	logger.Info("Starting server on :8080")

	err = server.ListenAndServe()
	if err != nil {
		logger.Error("Failed to start server", err)
	}
}
