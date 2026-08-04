package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thalesraymond/go-http-server/internal/database"
)

// newTestUserHandler wires a UserHandler around a mock store and mock authenticator.
func newTestUserHandler(mock userStore, authMock *mockAuthenticator) *UserHandler {
	return &UserHandler{apiConfig: newTestApiConfigWithAuth(authMock), store: mock}
}

func TestCreateUser(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name       string
		body       string
		hashError  bool
		setupMock  func(m *mockQuerier)
		wantStatus int
		wantError  string
	}{
		{
			name: "creates user",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockQuerier) {
				m.createUserFn = func(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
					return database.User{
						ID:             arg.ID,
						Email:          arg.Email,
						HashedPassword: arg.HashedPassword,
						CreatedAt:      createdAt,
						UpdatedAt:      createdAt,
					}, nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "rejects invalid JSON",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "handles password hashing error",
			body:       `{"email": "ada@example.com", "password": "hunter2"}`,
			hashError:  true,
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to create user",
		},
		{
			name: "handles store error",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockQuerier) {
				m.createUserFn = func(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
					return database.User{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to create user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := newDefaultMockAuthenticator()
			if tt.hashError {
				authMock.hashPasswordFn = func(password string) (string, error) {
					return "", errors.New("hash failure")
				}
			}

			mock := &mockQuerier{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestUserHandler(mock, authMock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPost, "/api/users", tt.body)
			h.CreateUser(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.createUserParams.Email != "ada@example.com" {
				t.Errorf("store received email = %q, want %q", mock.createUserParams.Email, "ada@example.com")
			}
			if mock.createUserParams.HashedPassword != "hashed:hunter2" {
				t.Errorf("store received hashed password = %q, want %q", mock.createUserParams.HashedPassword, "hashed:hunter2")
			}

			var got UserResponse
			decodeBody(t, rr, &got)
			if got.Email != "ada@example.com" {
				t.Errorf("response email = %q, want %q", got.Email, "ada@example.com")
			}
			if got.ID != mock.createUserParams.ID {
				t.Errorf("response id = %v, want stored user id %v", got.ID, mock.createUserParams.ID)
			}
			if got.IsChirpyRed {
				t.Error("response is_chirpy_red = true, want false")
			}
		})
	}
}

func TestUpdateUserByID(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		hashError  bool
		setupMock  func(m *mockQuerier)
		wantStatus int
		wantError  string
	}{
		{
			name: "updates user",
			body: `{"email": "ada@example.com", "password": "newpass"}`,
			setupMock: func(m *mockQuerier) {
				m.updateUserFn = func(ctx context.Context, arg database.UpdateUserParams) (database.User, error) {
					return database.User{
						ID:             arg.ID,
						Email:          arg.Email,
						HashedPassword: arg.HashedPassword,
					}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "rejects missing email",
			body:       `{"email": "", "password": "newpass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "email and password are required",
		},
		{
			name:       "rejects missing password",
			body:       `{"email": "ada@example.com", "password": ""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "email and password are required",
		},
		{
			name:       "rejects invalid JSON",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "handles password hashing error",
			body:       `{"email": "ada@example.com", "password": "newpass"}`,
			hashError:  true,
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to update user",
		},
		{
			name: "handles store error",
			body: `{"email": "ada@example.com", "password": "newpass"}`,
			setupMock: func(m *mockQuerier) {
				m.updateUserFn = func(ctx context.Context, arg database.UpdateUserParams) (database.User, error) {
					return database.User{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to update user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := newDefaultMockAuthenticator()
			if tt.hashError {
				authMock.hashPasswordFn = func(password string) (string, error) {
					return "", errors.New("hash failure")
				}
			}

			mock := &mockQuerier{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestUserHandler(mock, authMock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPut, "/api/users", tt.body)
			h.UpdateUserByID(rr, req, testUserID)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.updateUserParams.ID != testUserID {
				t.Errorf("store received id = %v, want %v", mock.updateUserParams.ID, testUserID)
			}
			if mock.updateUserParams.Email != "ada@example.com" {
				t.Errorf("store received email = %q, want %q", mock.updateUserParams.Email, "ada@example.com")
			}
			if mock.updateUserParams.HashedPassword != "hashed:newpass" {
				t.Errorf("store received hashed password = %q, want %q", mock.updateUserParams.HashedPassword, "hashed:newpass")
			}

			var got UserResponse
			decodeBody(t, rr, &got)
			if got.Email != "ada@example.com" {
				t.Errorf("response email = %q, want %q", got.Email, "ada@example.com")
			}
			if got.ID != testUserID {
				t.Errorf("response id = %v, want %v", got.ID, testUserID)
			}
		})
	}
}
