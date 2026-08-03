package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/thalesraymond/go-http-server/internal/database"
	"github.com/thalesraymond/go-http-server/internal/handler"

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
	apiConfig := handler.NewApiConfig(database.New(db), secret, polkaKey)

	serverMux.Handle("/app/", apiConfig.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./")))))

	serverMux.HandleFunc("GET /api/healthz", handler.HealthzHandler)

	serverMux.HandleFunc("GET /admin/metrics", apiConfig.HandlerMetrics)

	serverMux.HandleFunc("POST /admin/reset", apiConfig.HandlerReset)

	userHandler := handler.NewUserHandler(apiConfig)
	userHandler.RegisterRoutes(serverMux)

	loginHandler := handler.NewLoginHandler(apiConfig)
	loginHandler.RegisterRoutes(serverMux)

	chiprHandler := handler.NewChirpHandler(apiConfig)
	chiprHandler.RegisterRoutes(serverMux)

	polkaHandler := handler.NewPolkaHandler(apiConfig)
	polkaHandler.RegisterRoutes(serverMux)

	apiConfig.Logger.Info("Starting server on :8080")

	err = server.ListenAndServe()
	if err != nil {
		apiConfig.Logger.Error("Failed to start server", err)
	}
}
