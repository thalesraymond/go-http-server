package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
)

type AuthedHandler func(w http.ResponseWriter, r *http.Request, userID uuid.UUID)

type AuthHandshake struct {
	logger        Logger
	authenticator auth.Authenticator
	polkaKey      string
}

func NewAuthHandshake(logger Logger, authenticator auth.Authenticator, polkaKey string) *AuthHandshake {
	return &AuthHandshake{
		logger:        logger,
		authenticator: authenticator,
		polkaKey:      polkaKey,
	}
}

func (hs *AuthHandshake) RequireAuth(next AuthedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			hs.logger.Error("Missing or invalid Authorization header", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		userID, err := hs.authenticator.ValidateJWT(token)
		if err != nil {
			hs.logger.Error("Invalid or expired JWT", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		next(w, r, userID)
	})
}

func (hs *AuthHandshake) RequirePolkaAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := auth.GetAPIKey(r.Header)
		if err != nil || apiKey != hs.polkaKey {
			hs.logger.Error("Missing or invalid Polka API key", err)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
