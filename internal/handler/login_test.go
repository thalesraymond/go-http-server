package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

func TestHandlerLogin(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	storedUser := database.User{
		ID:             uuid.New(),
		Email:          "test@example.com",
		HashedPassword: "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$somehash",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	originalCheckPasswordHash := auth.CheckPasswordHash
	defer func() { auth.CheckPasswordHash = originalCheckPasswordHash }()

	tests := []struct {
		name               string
		body               string
		getUserFn          func(ctx context.Context, email string) (database.User, error)
		checkPassFn        func(password, hash string) (bool, error)
		createRefreshToken func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
		wantStatus         int
		wantEmail          string
	}{
		{
			name: "valid login",
			body: `{"email":"test@example.com","password":"secret123"}`,
			getUserFn: func(_ context.Context, email string) (database.User, error) {
				if email != storedUser.Email {
					return database.User{}, errors.New("unexpected email")
				}
				return storedUser, nil
			},
			checkPassFn: func(password, hash string) (bool, error) {
				if password != "secret123" || hash != storedUser.HashedPassword {
					return false, nil
				}
				return true, nil
			},
			createRefreshToken: func(_ context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
				return database.RefreshToken{Token: arg.Token, UserID: arg.UserID, ExpiresAt: arg.ExpiresAt}, nil
			},
			wantStatus: http.StatusOK,
			wantEmail:  "test@example.com",
		},
		{
			name:       "invalid json",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing email",
			body:       `{"email":"","password":"secret123"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"email":"test@example.com","password":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			body: `{"email":"missing@example.com","password":"secret123"}`,
			getUserFn: func(_ context.Context, _ string) (database.User, error) {
				return database.User{}, errors.New("not found")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "password hash check error",
			body: `{"email":"test@example.com","password":"secret123"}`,
			getUserFn: func(_ context.Context, _ string) (database.User, error) {
				return storedUser, nil
			},
			checkPassFn: func(_, _ string) (bool, error) {
				return false, errors.New("hash error")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "incorrect password",
			body: `{"email":"test@example.com","password":"wrongpassword"}`,
			getUserFn: func(_ context.Context, _ string) (database.User, error) {
				return storedUser, nil
			},
			checkPassFn: func(password, _ string) (bool, error) {
				return password == "secret123", nil
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "refresh token creation error",
			body: `{"email":"test@example.com","password":"secret123"}`,
			getUserFn: func(_ context.Context, email string) (database.User, error) {
				if email != storedUser.Email {
					return database.User{}, errors.New("unexpected email")
				}
				return storedUser, nil
			},
			checkPassFn: func(password, hash string) (bool, error) {
				if password != "secret123" || hash != storedUser.HashedPassword {
					return false, nil
				}
				return true, nil
			},
			createRefreshToken: func(_ context.Context, _ database.CreateRefreshTokenParams) (database.RefreshToken, error) {
				return database.RefreshToken{}, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockLoginStore{}
			if tt.getUserFn != nil {
				store.getUserByEmailFn = tt.getUserFn
			}

			if tt.checkPassFn != nil {
				auth.CheckPasswordHash = tt.checkPassFn
			} else {
				auth.CheckPasswordHash = originalCheckPasswordHash
			}

			if tt.createRefreshToken != nil {
				store.createRefreshTokenFn = tt.createRefreshToken
			}

			h := newTestLoginHandler(store)

			req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.Login(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantStatus == http.StatusOK && tt.wantEmail != "" {
				var resp LoginResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.Email != tt.wantEmail {
					t.Errorf("email = %q, want %q", resp.Email, tt.wantEmail)
				}
				if resp.Token == "" {
					t.Errorf("token is empty")
				}
				if resp.RefreshToken == "" {
					t.Errorf("refresh token is empty")
				}
			}
		})
	}
}

func TestHandlerGetRefreshTokenMissingAuth(t *testing.T) {
	cases := []struct {
		name       string
		authHeader string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Invalid Authorization header",
		},
		{
			name:       "malformed header",
			authHeader: "Bearer",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Invalid Authorization header",
		},
		{
			name:       "non-bearer scheme",
			authHeader: "Basic abc123",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Invalid Authorization header",
		},
	}

	h := newTestLoginHandler(nil)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rr := httptest.NewRecorder()

			h.GetRefreshToken(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandlerRevokeRefreshTokenSuccess(t *testing.T) {
	store := &mockLoginStore{}
	revoked := false
	store.revokeRefreshTokenFn = func(_ context.Context, _ string) error {
		revoked = true
		return nil
	}

	h := newTestLoginHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/revoke", nil)
	req.Header.Set("Authorization", "Bearer some-refresh-token")
	rr := httptest.NewRecorder()

	h.RevokeRefreshToken(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body: %s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Errorf("body = %q, want empty (204 must have no body)", rr.Body.String())
	}
	if !revoked {
		t.Errorf("store.RevokeRefreshToken was not called")
	}
}

func TestHandlerRevokeRefreshTokenMissingAuth(t *testing.T) {
	cases := []struct {
		name       string
		authHeader string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Invalid Authorization header",
		},
		{
			name:       "malformed header",
			authHeader: "Bearer",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Invalid Authorization header",
		},
		{
			name:       "non-bearer scheme",
			authHeader: "Basic abc123",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Invalid Authorization header",
		},
	}

	h := newTestLoginHandler(nil)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/revoke", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rr := httptest.NewRecorder()

			h.RevokeRefreshToken(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}
