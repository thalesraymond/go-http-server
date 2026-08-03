package handler

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/thalesraymond/go-http-server/internal/database"
)

type Logger interface {
	Error(msg string, err error)
	Info(msg string)
	Debug(msg string)
	Warn(msg string)
}

type defaultLogger struct{ l *log.Logger }

func (l *defaultLogger) Error(msg string, err error) {
	l.l.Printf("ERROR %s: %v", msg, err)
}

func (l *defaultLogger) Info(msg string) {
	l.l.Printf("INFO %s", msg)
}

func (l *defaultLogger) Debug(msg string) {
	l.l.Printf("DEBUG %s", msg)
}

func (l *defaultLogger) Warn(msg string) {
	l.l.Printf("WARN %s", msg)
}

func NewLogger() Logger {
	return &defaultLogger{l: log.Default()}
}

type ApiConfig struct {
	Database       database.Querier
	fileserverHits atomic.Int32
	Logger         Logger
	Secret         string
	PolkaKey       string
}

func NewApiConfig(db database.Querier, secret string, polkaKey string) *ApiConfig {
	return &ApiConfig{
		Database: db,
		Logger:   NewLogger(),
		Secret:   secret,
		PolkaKey: polkaKey,
	}
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) HandlerMetrics(w http.ResponseWriter, r *http.Request) {
	body := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())

	writeHTML(w, http.StatusOK, body)
}

func (cfg *ApiConfig) HandlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)

	err := cfg.Database.DeleteAllUsers(r.Context())
	if err != nil {
		cfg.Logger.Error("Failed to delete all users", err)
		http.Error(w, "Failed to delete all users", http.StatusInternalServerError)
		return
	}

	writeText(w, http.StatusOK, "OK")
}
