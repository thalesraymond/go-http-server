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

func newTestApiConfig(db *MockDatabase) *ApiConfig {
	return &ApiConfig{Database: db, Logger: &MockLogger{}}
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
			mockDB := &MockDatabase{}
			if tt.mockFn != nil {
				mockDB.CreateUserFn = tt.mockFn
			}

			if tt.hashPasswordFn != nil {
				auth.HashPassword = tt.hashPasswordFn
			} else {
				auth.HashPassword = originalHashPassword
			}

			h := NewUserHandler(newTestApiConfig(mockDB))

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
		name        string
		body        string
		getUserFn   func(ctx context.Context, email string) (database.User, error)
		checkPassFn func(password, hash string) (bool, error)
		wantStatus  int
		wantEmail   string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			if tt.getUserFn != nil {
				mockDB.GetUserByEmailFn = tt.getUserFn
			}

			if tt.checkPassFn != nil {
				auth.CheckPasswordHash = tt.checkPassFn
			} else {
				auth.CheckPasswordHash = originalCheckPasswordHash
			}

			h := NewUserHandler(newTestApiConfig(mockDB))

			req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.Login(rr, req)

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

func TestHandlerGetUsers(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		mockDB := &MockDatabase{}
		h := NewUserHandler(newTestApiConfig(mockDB))

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
		mockDB := &MockDatabase{}
		h := NewUserHandler(newTestApiConfig(mockDB))

		req := httptest.NewRequest(http.MethodGet, "/api/users/"+uuid.New().String(), nil)
		rr := httptest.NewRecorder()

		h.GetUserByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}

func TestHandlerUpdateUserByID(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		mockDB := &MockDatabase{}
		h := NewUserHandler(newTestApiConfig(mockDB))

		req := httptest.NewRequest(http.MethodPut, "/api/users", nil)
		rr := httptest.NewRecorder()

		h.UpdateUserByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}

func TestHandlerDeleteUserByID(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		mockDB := &MockDatabase{}
		h := NewUserHandler(newTestApiConfig(mockDB))

		req := httptest.NewRequest(http.MethodDelete, "/api/users", nil)
		rr := httptest.NewRecorder()

		h.DeleteUserByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}
