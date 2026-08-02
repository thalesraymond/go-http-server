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

	tests := []struct {
		name       string
		body       string
		mockFn     func(ctx context.Context, arg database.CreateUserParams) (database.User, error)
		wantStatus int
		wantEmail  string
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
			name:       "password hashing error",
			body:       `{"email":"test@example.com","password":""}`,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			if tt.mockFn != nil {
				mockDB.CreateUserFn = tt.mockFn
			}

			h := NewUserHandler(newTestApiConfig(mockDB))

			req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.HandlerCreateUser(rr, req)

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
		mockDB := &MockDatabase{}
		h := NewUserHandler(newTestApiConfig(mockDB))

		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rr := httptest.NewRecorder()

		h.HandlerGetUsers(rr, req)

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

		h.HandlerGetUserByID(rr, req)

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

		h.HandlerUpdateUserByID(rr, req)

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

		h.HandlerDeleteUserByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}
