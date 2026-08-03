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
	userID := uuid.New()
	validToken, err := auth.MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	cfg := newTestApiConfig()
	cfg.Secret = secret

	target := func(w http.ResponseWriter, r *http.Request, gotUserID uuid.UUID) {
		if gotUserID != userID {
			t.Errorf("userID = %q, want %q", gotUserID, userID)
		}
		w.WriteHeader(http.StatusOK)
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "valid token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid prefix",
			authHeader: "Basic " + validToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong secret",
			authHeader: func() string {
				tok, _ := auth.MakeJWT(userID, "wrong-secret", time.Hour)
				return "Bearer " + tok
			}(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "expired token",
			authHeader: func() string {
				tok, _ := auth.MakeJWT(userID, secret, -time.Hour)
				return "Bearer " + tok
			}(),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			cfg.RequireAuth(target).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}

func TestRequirePolkaAuth(t *testing.T) {
	cfg := newTestApiConfig()

	target := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "valid key",
			authHeader: "ApiKey " + testPolkaKey,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong prefix",
			authHeader: "Bearer " + testPolkaKey,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong key",
			authHeader: "ApiKey wrong-key",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/polka/webhooks", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			cfg.RequirePolkaAuth(http.HandlerFunc(target)).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}
