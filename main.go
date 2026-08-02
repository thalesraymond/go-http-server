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

	godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}

	defer db.Close()

	apiConfig := handler.NewApiConfig(database.New(db))

	serverMux.Handle("/app/", apiConfig.MiddlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./")))))

	serverMux.HandleFunc("GET /api/healthz", handler.HealthzHandler)

	serverMux.HandleFunc("GET /admin/metrics", apiConfig.HandlerMetrics)

	serverMux.HandleFunc("POST /admin/reset", apiConfig.HandlerReset)

	serverMux.HandleFunc("POST /api/validate_chirp", handler.ValidateChirpHandler)

	userHandler := handler.NewUserHandler(apiConfig)
	userHandler.RegisterRoutes(serverMux)

	err = server.ListenAndServe()
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
