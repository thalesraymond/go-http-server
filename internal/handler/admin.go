package handler

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
)

type resetStore interface {
	DeleteAllUsers(ctx context.Context) error
}

type Admin struct {
	db             resetStore
	logger         Logger
	fileserverHits atomic.Int32
}

func NewAdmin(db resetStore, logger Logger) *Admin {
	return &Admin{db: db, logger: logger}
}

func (a *Admin) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (a *Admin) HandlerMetrics(w http.ResponseWriter, r *http.Request) {
	body := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, a.fileserverHits.Load())

	writeHTML(w, http.StatusOK, body)
}

func (a *Admin) HandlerReset(w http.ResponseWriter, r *http.Request) {
	a.fileserverHits.Store(0)

	err := a.db.DeleteAllUsers(r.Context())
	if err != nil {
		a.logger.Error("Failed to delete all users", err)
		http.Error(w, "Failed to delete all users", http.StatusInternalServerError)
		return
	}

	writeText(w, http.StatusOK, "OK")
}
