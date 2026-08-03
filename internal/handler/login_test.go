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
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

// mockLoginStore implements loginStore with per-method stub functions and
// records its arguments so tests can assert on them.
type mockLoginStore struct {
	getUserByEmailFn      func(ctx context.Context, email string) (database.User, error)
	createRefreshTokenFn  func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	getRefreshTokenByIDFn func(ctx context.Context, token string) (database.RefreshToken, error)
	revokeRefreshTokenFn  func(ctx context.Context, token string) error

	getUserByEmailEmail      string
	createRefreshTokenParams database.CreateRefreshTokenParams
	getRefreshTokenByIDToken string
	revokeRefreshTokenToken  string
}

func (m *mockLoginStore) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	m.getUserByEmailEmail = email
	if m.getUserByEmailFn == nil {
		return database.User{}, errUnstubbed
	}
	return m.getUserByEmailFn(ctx, email)
}

func (m *mockLoginStore) CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
	m.createRefreshTokenParams = arg
	if m.createRefreshTokenFn == nil {
		return database.RefreshToken{}, errUnstubbed
	}
	return m.createRefreshTokenFn(ctx, arg)
}

func (m *mockLoginStore) GetRefreshTokenByID(ctx context.Context, token string) (database.RefreshToken, error) {
	m.getRefreshTokenByIDToken = token
	if m.getRefreshTokenByIDFn == nil {
		return database.RefreshToken{}, errUnstubbed
	}
	return m.getRefreshTokenByIDFn(ctx, token)
}

func (m *mockLoginStore) RevokeRefreshToken(ctx context.Context, token string) error {
	m.revokeRefreshTokenToken = token
	if m.revokeRefreshTokenFn == nil {
		return errUnstubbed
	}
	return m.revokeRefreshTokenFn(ctx, token)
}

// newTestLoginHandler wires a LoginHandler around a mock store.
func newTestLoginHandler(mock loginStore) *LoginHandler {
	return &LoginHandler{apiConfig: newTestApiConfig(), store: mock}
}

func TestLogin(t *testing.T) {
	originalCheck := auth.CheckPasswordHash
	originalJWT := auth.MakeJWT
	originalRefresh := auth.MakeRefreshToken
	t.Cleanup(func() {
		auth.CheckPasswordHash = originalCheck
		auth.MakeJWT = originalJWT
		auth.MakeRefreshToken = originalRefresh
	})

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
		setupMock  func(m *mockLoginStore)
		wantStatus int
		wantError  string
	}{
		{
			name: "logs in and returns tokens",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockLoginStore) {
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
			setupMock: func(m *mockLoginStore) {
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
			setupMock: func(m *mockLoginStore) {
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
			setupMock: func(m *mockLoginStore) {
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
			setupMock: func(m *mockLoginStore) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to check password hash",
		},
		{
			name: "handles JWT creation error",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockLoginStore) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
				m.createRefreshTokenFn = func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
					return database.RefreshToken{Token: arg.Token}, nil
				}
			},
			makeJWTErr: true,
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to generate token",
		},
		{
			name: "handles refresh token creation error",
			body: `{"email": "ada@example.com", "password": "hunter2"}`,
			setupMock: func(m *mockLoginStore) {
				m.getUserByEmailFn = func(ctx context.Context, email string) (database.User, error) {
					return loginUser, nil
				}
				m.createRefreshTokenFn = func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
					return database.RefreshToken{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to create refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The auth helpers are package-level vars; subtests run
			// sequentially, so each case swaps in its own stubs and the
			// test-level cleanup restores the originals.
			checkHash := func(password, hash string) (bool, error) {
				return true, nil
			}
			if tt.checkHash != nil {
				checkHash = tt.checkHash
			}
			auth.CheckPasswordHash = checkHash

			makeJWT := func(userID uuid.UUID, secret string, expiresIn time.Duration) (string, error) {
				return "test-access-token", nil
			}
			if tt.makeJWTErr {
				makeJWT = func(userID uuid.UUID, secret string, expiresIn time.Duration) (string, error) {
					return "", errors.New("jwt failure")
				}
			}
			auth.MakeJWT = makeJWT

			auth.MakeRefreshToken = func() string { return "test-refresh-token" }

			mock := &mockLoginStore{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestLoginHandler(mock)
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
	originalJWT := auth.MakeJWT
	t.Cleanup(func() { auth.MakeJWT = originalJWT })

	valid := database.RefreshToken{
		Token:     "rt-valid",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tests := []struct {
		name       string
		authHeader string
		makeJWTErr bool
		setupMock  func(m *mockLoginStore)
		wantStatus int
		wantError  string
	}{
		{
			name:       "issues new access token",
			authHeader: "Bearer rt-valid",
			setupMock: func(m *mockLoginStore) {
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
			wantError:  "Invalid Authorization header",
		},
		{
			name:       "rejects unknown refresh token",
			authHeader: "Bearer rt-unknown",
			setupMock: func(m *mockLoginStore) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return database.RefreshToken{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Invalid refresh token",
		},
		{
			name:       "rejects revoked refresh token",
			authHeader: "Bearer rt-revoked",
			setupMock: func(m *mockLoginStore) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return database.RefreshToken{
						Token:     "rt-revoked",
						ExpiresAt: time.Now().Add(time.Hour),
						RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
					}, nil
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Refresh token revoked",
		},
		{
			name:       "rejects expired refresh token",
			authHeader: "Bearer rt-expired",
			setupMock: func(m *mockLoginStore) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return database.RefreshToken{
						Token:     "rt-expired",
						ExpiresAt: time.Now().Add(-time.Hour),
					}, nil
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "Refresh token expired",
		},
		{
			name:       "handles JWT creation error",
			authHeader: "Bearer rt-valid",
			makeJWTErr: true,
			setupMock: func(m *mockLoginStore) {
				m.getRefreshTokenByIDFn = func(ctx context.Context, token string) (database.RefreshToken, error) {
					return valid, nil
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to generate token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			makeJWT := func(userID uuid.UUID, secret string, expiresIn time.Duration) (string, error) {
				return "test-access-token", nil
			}
			if tt.makeJWTErr {
				makeJWT = func(userID uuid.UUID, secret string, expiresIn time.Duration) (string, error) {
					return "", errors.New("jwt failure")
				}
			}
			auth.MakeJWT = makeJWT

			mock := &mockLoginStore{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestLoginHandler(mock)
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
		setupMock  func(m *mockLoginStore)
		wantStatus int
		wantError  string
	}{
		{
			name:       "revokes refresh token",
			authHeader: "Bearer rt-revoke-me",
			setupMock: func(m *mockLoginStore) {
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
			wantError:  "Invalid Authorization header",
		},
		{
			name:       "handles store error",
			authHeader: "Bearer rt-revoke-me",
			setupMock: func(m *mockLoginStore) {
				m.revokeRefreshTokenFn = func(ctx context.Context, token string) error {
					return errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to remove refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLoginStore{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestLoginHandler(mock)
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
