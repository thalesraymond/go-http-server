package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
	"github.com/thalesraymond/go-http-server/internal/session"
)

// newTestSessionHandler wires a SessionHandler around the mock store, mock
// authenticator, and a real session.Session built from them.
func newTestSessionHandler(mock *mockQuerier, authMock *mockAuthenticator) *SessionHandler {
	return NewSessionHandler(mock, testLogger{}, authMock, session.New(mock, authMock))
}

func TestLogin(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	loginUser := database.User{
		ID:             userID,
		Email:          "ada@example.com",
		HashedPassword: "stored-hash",
		CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		IsChirpyRed:    false,
	}

	tests := []struct {
		name       string
		body       string
		checkHash  func(password, hash string) (bool, error)
		makeJWTErr bool
		setupMock  func(m *mockQuerier)
		wantStatus int
		wantError  string
	}{
		{
			name: "logs in and returns tokens",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockQuerier) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
				m.createRefreshTokenFn = func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
					return database.RefreshToken{Token: arg.Token}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "rejects invalid JSON",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "rejects missing email or password",
			body:       `{"email": "", "password": "hunter2"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "email and password are required",
		},
		{
			name: "rejects unknown email",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockQuerier) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return database.User{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Incorrect email or password",
		},
		{
			name: "handles user lookup error",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockQuerier) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return database.User{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to get user by email",
		},
		{
			name: "rejects wrong password",
			body: `{"email": "ada@example.com", "password": "wrong"}`,
			checkHash: func(password, hash string) (bool, error) {
				return false, nil
			},
			setupMock: func(m *mockQuerier) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Incorrect email or password",
		},
		{
			name: "handles password check error",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			checkHash: func(password, hash string) (bool, error) {
				return false, errors.New("compare failure")
			},
			setupMock: func(m *mockQuerier) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to check password hash",
		},
		{
			name: "handles session start error",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockQuerier) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
				m.createRefreshTokenFn = func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
					return database.RefreshToken{Token: arg.Token}, nil
				}
			},
			makeJWTErr: true,
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
		},
		{
			name: "handles refresh token creation error",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockQuerier) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
				m.createRefreshTokenFn = func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
					return database.RefreshToken{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := newDefaultMockAuthenticator()
			if tt.checkHash != nil {
				authMock.checkPasswordFn = tt.checkHash
			}
			if tt.makeJWTErr {
				authMock.makeJWTFn = func(userID uuid.UUID, expiresIn time.Duration) (string, error) {
					return "", errors.New("jwt failure")
				}
			}

			mock := &mockQuerier{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestSessionHandler(mock, authMock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPost, "/api/login", tt.body)

			h.Login(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.getUserByEmailEmail != "ada@example.com" {
				t.Errorf("store received email = %q, want %q", mock.getUserByEmailEmail, "ada@example.com")
			}
			if mock.createRefreshTokenParams.UserID != userID {
				t.Errorf("store received refresh token userID = %v, want %v", mock.createRefreshTokenParams.UserID, userID)
			}
			if mock.createRefreshTokenParams.Token != "test-refresh-token" {
				t.Errorf("store received refresh token = %q, want %q", mock.createRefreshTokenParams.Token, "test-refresh-token")
			}
			if mock.createRefreshTokenParams.ExpiresAt.IsZero() {
				t.Error("store received zero expires_at for refresh token")
			}

			var got LoginResponse
			decodeBody(t, rr, &got)
			if got.Token != "test-access-token" {
				t.Errorf("response token = %q, want %q", got.Token, "test-access-token")
			}
			if got.RefreshToken != "test-refresh-token" {
				t.Errorf("response refresh_token = %q, want %q", got.RefreshToken, "test-refresh-token")
			}
			if got.Email != loginUser.Email {
				t.Errorf("response email = %q, want %q", got.Email, loginUser.Email)
			}
			if got.Id != userID {
				t.Errorf("response id = %v, want %v", got.Id, userID)
			}
			if got.IsChirpyRed {
				t.Error("response is_chirpy_red = true, want false")
			}
		})
	}
}

func TestGetRefreshToken(t *testing.T) {
	valid := database.RefreshToken{
		Token:     "rt-valid",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tests := []struct {
		name       string
		authHeader string
		makeJWTErr bool
		setupMock  func(m *mockQuerier)
		wantStatus int
		wantError  string
	}{
		{
			name:       "issues new access token",
			authHeader: "Bearer rt-valid",
			setupMock: func(m *mockQuerier) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return valid, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "rejects missing authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects unknown refresh token",
			authHeader: "Bearer rt-unknown",
			setupMock: func(m *mockQuerier) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return database.RefreshToken{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects revoked refresh token",
			authHeader: "Bearer rt-revoked",
			setupMock: func(m *mockQuerier) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return database.RefreshToken{
						Token:     "rt-revoked",
						ExpiresAt: time.Now().Add(time.Hour),
						RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
					}, nil
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects expired refresh token",
			authHeader: "Bearer rt-expired",
			setupMock: func(m *mockQuerier) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return database.RefreshToken{
						Token:     "rt-expired",
						ExpiresAt: time.Now().Add(-time.Hour),
					}, nil
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "handles store error",
			authHeader: "Bearer rt-valid",
			setupMock: func(m *mockQuerier) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return database.RefreshToken{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
		},
		{
			name:       "handles JWT creation error",
			authHeader: "Bearer rt-valid",
			makeJWTErr: true,
			setupMock: func(m *mockQuerier) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return valid, nil
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := newDefaultMockAuthenticator()
			if tt.makeJWTErr {
				authMock.makeJWTFn = func(userID uuid.UUID, expiresIn time.Duration) (string, error) {
					return "", errors.New("jwt failure")
				}
			}

			mock := &mockQuerier{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestSessionHandler(mock, authMock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPost, "/api/refresh", "")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			h.GetRefreshToken(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.getRefreshTokenByIDToken != "rt-valid" {
				t.Errorf("store received token = %q, want %q", mock.getRefreshTokenByIDToken, "rt-valid")
			}

			var got struct {
				Token string `json:"token"`
			}
			decodeBody(t, rr, &got)
			if got.Token != "test-access-token" {
				t.Errorf("response token = %q, want %q", got.Token, "test-access-token")
			}
		})
	}
}

func TestRevokeRefreshToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		setupMock  func(m *mockQuerier)
		wantStatus int
		wantError  string
	}{
		{
			name:       "revokes refresh token",
			authHeader: "Bearer rt-revoke-me",
			setupMock: func(m *mockQuerier) {
				m.revokeRefreshTokenFn = func(ctx context.Context, token string) error {
					return nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "rejects missing authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "handles store error",
			authHeader: "Bearer rt-revoke-me",
			setupMock: func(m *mockQuerier) {
				m.revokeRefreshTokenFn = func(ctx context.Context, token string) error {
					return errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockQuerier{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			authMock := newDefaultMockAuthenticator()
			h := newTestSessionHandler(mock, authMock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPost, "/api/revoke", "")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			h.RevokeRefreshToken(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.revokeRefreshTokenToken != "rt-revoke-me" {
				t.Errorf("store received token = %q, want %q", mock.revokeRefreshTokenToken, "rt-revoke-me")
			}
		})
	}
}
