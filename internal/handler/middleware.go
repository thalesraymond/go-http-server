package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
)

type contextKey string

const userIDKey contextKey = "userID"

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)

	if !ok {
		return uuid.Nil, false
	}

	return userID, true
}

func (cfg *ApiConfig) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			cfg.Logger.Error("Missing or invalid Authorization header", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		userID, err := auth.ValidateJWT(token, cfg.Secret)
		if err != nil {
			cfg.Logger.Error("Invalid or expired JWT", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
