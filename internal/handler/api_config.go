package handler

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/thalesraymond/go-http-server/internal/database"
)

type ApiConfig struct {
	Database       *database.Queries
	fileserverHits atomic.Int32
}

func NewApiConfig(db *database.Queries) *ApiConfig {
	return &ApiConfig{
		Database: db,
	}
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) HandlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())

	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (cfg *ApiConfig) HandlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)

	cfg.Database.DeleteAllUsers(r.Context())

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte("OK"))

	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
