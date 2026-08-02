package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()
	validToken, err := auth.MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	cfg := newTestApiConfig(&MockDatabase{})
	cfg.Secret = secret

	target := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Error("expected userID in context, got none")
		}
		if gotUserID != userID {
			t.Errorf("userID = %q, want %q", gotUserID, userID)
		}
		w.WriteHeader(http.StatusOK)
	})

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

			cfg.AuthMiddleware(target).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}

func TestGetUserIDFromContext_Missing(t *testing.T) {
	_, ok := GetUserIDFromContext(t.Context())
	if ok {
		t.Error("expected no userID in context")
	}
}
