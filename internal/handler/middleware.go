package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
)

type AuthedHandler func(w http.ResponseWriter, r *http.Request, userID uuid.UUID)

func (cfg *ApiConfig) RequireAuth(next AuthedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			cfg.Logger.Error("Missing or invalid Authorization header", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		userID, err := cfg.Authenticator.ValidateJWT(token)
		if err != nil {
			cfg.Logger.Error("Invalid or expired JWT", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		next(w, r, userID)
	})
}

func (cfg *ApiConfig) RequirePolkaAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := auth.GetAPIKey(r.Header)
		if err != nil || apiKey != cfg.PolkaKey {
			cfg.Logger.Error("Missing or invalid Polka API key", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
