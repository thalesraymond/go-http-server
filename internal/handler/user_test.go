package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

// --- helpers ---

func newTestApiConfig() *ApiConfig {
	return &ApiConfig{Database: &noopQuerier{}, Logger: &MockLogger{}, Secret: "test-secret"}
}

// --- User Tests ---

func TestHandlerCreateUser(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	wantUser := database.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	originalHashPassword := auth.HashPassword
	defer func() { auth.HashPassword = originalHashPassword }()

	tests := []struct {
		name           string
		body           string
		mockFn         func(ctx context.Context, arg database.CreateUserParams) (database.User, error)
		hashPasswordFn func(password string) (string, error)
		wantStatus     int
		wantEmail      string
	}{
		{
			name: "valid user",
			body: `{"email":"test@example.com","password":"secret123"}`,
			mockFn: func(_ context.Context, arg database.CreateUserParams) (database.User, error) {
				if arg.Email != "test@example.com" || arg.HashedPassword == "" {
					return database.User{}, errors.New("unexpected params")
				}
				return wantUser, nil
			},
			wantStatus: http.StatusCreated,
			wantEmail:  "test@example.com",
		},
		{
			name:       "invalid json",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty email still succeeds",
			body: `{"email":"","password":"secret123"}`,
			mockFn: func(_ context.Context, arg database.CreateUserParams) (database.User, error) {
				if arg.HashedPassword == "" {
					return database.User{}, errors.New("password not hashed")
				}
				return wantUser, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "database error",
			body: `{"email":"fail@example.com","password":"secret123"}`,
			mockFn: func(_ context.Context, _ database.CreateUserParams) (database.User, error) {
				return database.User{}, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "password hashing error",
			body: `{"email":"test@example.com","password":"secret123"}`,
			hashPasswordFn: func(_ string) (string, error) {
				return "", errors.New("hash error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockUserStore{}
			if tt.mockFn != nil {
				store.createUserFn = tt.mockFn
			}

			if tt.hashPasswordFn != nil {
				auth.HashPassword = tt.hashPasswordFn
			} else {
				auth.HashPassword = originalHashPassword
			}

			h := newTestUserHandler(store)

			req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.CreateUser(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantStatus == http.StatusCreated && tt.wantEmail != "" {
				var resp UserResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.Email != tt.wantEmail {
					t.Errorf("email = %q, want %q", resp.Email, tt.wantEmail)
				}
			}
		})
	}
}

func TestHandlerGetUsers(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		h := newTestUserHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rr := httptest.NewRecorder()

		h.GetUsers(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}

func TestHandlerGetUserByID(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		h := newTestUserHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/users/"+uuid.New().String(), nil)
		rr := httptest.NewRecorder()

		h.GetUserByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}

func TestHandlerUpdateUserByID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()
	updatedUser := database.User{
		ID:        userID,
		Email:     "updated@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	originalHashPassword := auth.HashPassword
	defer func() { auth.HashPassword = originalHashPassword }()

	tests := []struct {
		name           string
		body           string
		userID         uuid.UUID
		mockFn         func(ctx context.Context, arg database.UpdateUserParams) (database.User, error)
		hashPasswordFn func(password string) (string, error)
		wantStatus     int
		wantEmail      string
	}{
		{
			name:   "valid update",
			body:   `{"email":"updated@example.com","password":"newsecret123"}`,
			userID: userID,
			mockFn: func(_ context.Context, arg database.UpdateUserParams) (database.User, error) {
				if arg.ID != userID || arg.Email != "updated@example.com" || arg.HashedPassword == "" {
					return database.User{}, errors.New("unexpected params")
				}
				return updatedUser, nil
			},
			wantStatus: http.StatusOK,
			wantEmail:  "updated@example.com",
		},
		{
			name:       "missing user id in context",
			body:       `{"email":"updated@example.com","password":"newsecret123"}`,
			userID:     uuid.Nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json",
			body:       `{bad json`,
			userID:     userID,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing email",
			body:       `{"email":"","password":"newsecret123"}`,
			userID:     userID,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"email":"updated@example.com","password":""}`,
			userID:     userID,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "password hashing error",
			body:   `{"email":"updated@example.com","password":"newsecret123"}`,
			userID: userID,
			hashPasswordFn: func(_ string) (string, error) {
				return "", errors.New("hash error")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "database error",
			body:   `{"email":"fail@example.com","password":"newsecret123"}`,
			userID: userID,
			mockFn: func(_ context.Context, _ database.UpdateUserParams) (database.User, error) {
				return database.User{}, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockUserStore{}
			if tt.mockFn != nil {
				store.updateUserFn = tt.mockFn
			}

			if tt.hashPasswordFn != nil {
				auth.HashPassword = tt.hashPasswordFn
			} else {
				auth.HashPassword = originalHashPassword
			}

			h := newTestUserHandler(store)

			req := httptest.NewRequest(http.MethodPut, "/api/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != uuid.Nil {
				ctx := context.WithValue(req.Context(), userIDKey, tt.userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			h.UpdateUserByID(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantStatus == http.StatusOK && tt.wantEmail != "" {
				var resp UserResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.Email != tt.wantEmail {
					t.Errorf("email = %q, want %q", resp.Email, tt.wantEmail)
				}
			}
		})
	}
}

func TestHandlerDeleteUserByID(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		h := newTestUserHandler(nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/users", nil)
		rr := httptest.NewRecorder()

		h.DeleteUserByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}
