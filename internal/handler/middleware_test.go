package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
)

func TestRequireAuth(t *testing.T) {
	secret := "test-secret"
	authenticator := auth.NewRealAuthenticator(secret)
	cfg := &ApiConfig{
		Authenticator: authenticator,
		Logger:        testLogger{},
		PolkaKey:      "test-polka-key",
	}
	userID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	validToken, err := authenticator.MakeJWT(userID, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT: %v", err)
	}
	wrongSecretToken, err := auth.NewRealAuthenticator("wrong-secret").MakeJWT(userID, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT: %v", err)
	}
	expiredToken, err := authenticator.MakeJWT(userID, -time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT: %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantError  string
	}{
		{
			name:       "allows valid token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "rejects missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects malformed token",
			authHeader: "Bearer not-a-jwt",
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects token signed with wrong secret",
			authHeader: "Bearer " + wrongSecretToken,
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects expired token",
			authHeader: "Bearer " + expiredToken,
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID uuid.UUID
			next := AuthedHandler(func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
				gotUserID = id
				writeNoContent(w)
			})

			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodGet, "/api/chirps", "")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			cfg.RequireAuth(next).ServeHTTP(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if gotUserID != userID {
				t.Errorf("handler received userID = %v, want %v", gotUserID, userID)
			}
		})
	}
}

func TestRequirePolkaAuth(t *testing.T) {
	cfg := newTestApiConfig()

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantError  string
	}{
		{
			name:       "allows matching api key",
			authHeader: "ApiKey test-polka-key",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "rejects missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects wrong api key",
			authHeader: "ApiKey wrong-key",
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects bearer token",
			authHeader: "Bearer whatever",
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeNoContent(w)
			})

			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPost, "/api/polka/webhooks", "")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			cfg.RequirePolkaAuth(next).ServeHTTP(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}
		})
	}
}
