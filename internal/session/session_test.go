package session

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
)

type fakeStore struct {
	createFn func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	getFn    func(ctx context.Context, token string) (database.RefreshToken, error)
	revokeFn func(ctx context.Context, token string) error

	created  database.CreateRefreshTokenParams
	gotToken string
	revoked  string
}

func (f *fakeStore) CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
	f.created = arg
	if f.createFn == nil {
		return database.RefreshToken{}, errUnstubbed
	}
	return f.createFn(ctx, arg)
}

func (f *fakeStore) GetRefreshTokenByID(ctx context.Context, token string) (database.RefreshToken, error) {
	f.gotToken = token
	if f.getFn == nil {
		return database.RefreshToken{}, errUnstubbed
	}
	return f.getFn(ctx, token)
}

func (f *fakeStore) RevokeRefreshToken(ctx context.Context, token string) error {
	f.revoked = token
	if f.revokeFn == nil {
		return errUnstubbed
	}
	return f.revokeFn(ctx, token)
}

type fakeAuthenticator struct {
	makeJWTErr        bool
	accessToken       string
	refreshToken      string
	makeJWTUserID     uuid.UUID
	makeJWTExpiresIn  time.Duration
	validateJWTArg    string
	hashPasswordArg   string
	checkPasswordArgs struct{ password, hash string }
}

func (f *fakeAuthenticator) HashPassword(password string) (string, error) {
	f.hashPasswordArg = password
	return "hashed:" + password, nil
}

func (f *fakeAuthenticator) CheckPasswordHash(password, hash string) (bool, error) {
	f.checkPasswordArgs.password = password
	f.checkPasswordArgs.hash = hash
	return true, nil
}

func (f *fakeAuthenticator) MakeJWT(userID uuid.UUID, expiresIn time.Duration) (string, error) {
	f.makeJWTUserID = userID
	f.makeJWTExpiresIn = expiresIn
	if f.makeJWTErr {
		return "", errors.New("jwt failure")
	}
	return f.accessToken, nil
}

func (f *fakeAuthenticator) ValidateJWT(tokenString string) (uuid.UUID, error) {
	f.validateJWTArg = tokenString
	return uuid.Nil, nil
}

func (f *fakeAuthenticator) MakeRefreshToken() string {
	return f.refreshToken
}

var errUnstubbed = errors.New("fake method not stubbed")

func newTestSession(f *fakeStore, a *fakeAuthenticator) *Session {
	return New(f, a)
}

func testRequest(method, target string, authHeader string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return req
}

func TestStart(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	t.Run("issues access and refresh tokens", func(t *testing.T) {
		store := &fakeStore{
			createFn: func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
				return database.RefreshToken{Token: arg.Token}, nil
			},
		}
		auth := &fakeAuthenticator{accessToken: "access-1", refreshToken: "refresh-1"}
		s := newTestSession(store, auth)

		access, refresh, err := s.Start(context.Background(), userID)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		if access != "access-1" {
			t.Errorf("access token = %q, want %q", access, "access-1")
		}
		if refresh != "refresh-1" {
			t.Errorf("refresh token = %q, want %q", refresh, "refresh-1")
		}
		if store.created.UserID != userID {
			t.Errorf("store received userID = %v, want %v", store.created.UserID, userID)
		}
		if store.created.Token != "refresh-1" {
			t.Errorf("store received token = %q, want %q", store.created.Token, "refresh-1")
		}
		wantExpiry := time.Now().Add(RefreshTokenTTL)
		if diff := store.created.ExpiresAt.Sub(wantExpiry); diff > time.Second || diff < -time.Second {
			t.Errorf("store received expires_at %v, want ~%v", store.created.ExpiresAt, wantExpiry)
		}
		if auth.makeJWTExpiresIn != AccessTokenTTL {
			t.Errorf("MakeJWT expires_in = %v, want %v", auth.makeJWTExpiresIn, AccessTokenTTL)
		}
		if auth.makeJWTUserID != userID {
			t.Errorf("MakeJWT userID = %v, want %v", auth.makeJWTUserID, userID)
		}
	})

	t.Run("returns error when JWT creation fails", func(t *testing.T) {
		store := &fakeStore{}
		auth := &fakeAuthenticator{accessToken: "access-1", refreshToken: "refresh-1", makeJWTErr: true}
		s := newTestSession(store, auth)

		if _, _, err := s.Start(context.Background(), userID); err == nil {
			t.Fatal("Start: expected error, got nil")
		}
	})

	t.Run("returns error when store fails", func(t *testing.T) {
		store := &fakeStore{
			createFn: func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
				return database.RefreshToken{}, errors.New("database down")
			},
		}
		auth := &fakeAuthenticator{accessToken: "access-1", refreshToken: "refresh-1"}
		s := newTestSession(store, auth)

		if _, _, err := s.Start(context.Background(), userID); err == nil {
			t.Fatal("Start: expected error, got nil")
		}
	})
}

func TestRenew(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	valid := database.RefreshToken{
		Token:     "rt-valid",
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	t.Run("issues a new access token", func(t *testing.T) {
		store := &fakeStore{
			getFn: func(ctx context.Context, token string) (database.RefreshToken, error) {
				return valid, nil
			},
		}
		auth := &fakeAuthenticator{accessToken: "access-1", refreshToken: "refresh-1"}
		s := newTestSession(store, auth)

		access, err := s.Renew(context.Background(), testRequest(http.MethodPost, "/api/refresh", "Bearer rt-valid"))
		if err != nil {
			t.Fatalf("Renew: %v", err)
		}
		if access != "access-1" {
			t.Errorf("access token = %q, want %q", access, "access-1")
		}
		if store.gotToken != "rt-valid" {
			t.Errorf("store received token = %q, want %q", store.gotToken, "rt-valid")
		}
		if auth.makeJWTUserID != userID {
			t.Errorf("MakeJWT userID = %v, want %v", auth.makeJWTUserID, userID)
		}
		if auth.makeJWTExpiresIn != AccessTokenTTL {
			t.Errorf("MakeJWT expires_in = %v, want %v", auth.makeJWTExpiresIn, AccessTokenTTL)
		}
	})

	t.Run("rejects missing authorization header", func(t *testing.T) {
		store := &fakeStore{}
		s := newTestSession(store, &fakeAuthenticator{})

		_, err := s.Renew(context.Background(), testRequest(http.MethodPost, "/api/refresh", ""))
		if !errors.Is(err, ErrRefreshTokenInvalid) {
			t.Errorf("error = %v, want %v", err, ErrRefreshTokenInvalid)
		}
	})

	t.Run("rejects unknown refresh token", func(t *testing.T) {
		store := &fakeStore{
			getFn: func(ctx context.Context, token string) (database.RefreshToken, error) {
				return database.RefreshToken{}, sql.ErrNoRows
			},
		}
		s := newTestSession(store, &fakeAuthenticator{})

		_, err := s.Renew(context.Background(), testRequest(http.MethodPost, "/api/refresh", "Bearer rt-unknown"))
		if !errors.Is(err, ErrRefreshTokenInvalid) {
			t.Errorf("error = %v, want %v", err, ErrRefreshTokenInvalid)
		}
	})

	t.Run("rejects revoked refresh token", func(t *testing.T) {
		store := &fakeStore{
			getFn: func(ctx context.Context, token string) (database.RefreshToken, error) {
				return database.RefreshToken{
					Token:     "rt-revoked",
					ExpiresAt: time.Now().Add(time.Hour),
					RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
				}, nil
			},
		}
		s := newTestSession(store, &fakeAuthenticator{})

		_, err := s.Renew(context.Background(), testRequest(http.MethodPost, "/api/refresh", "Bearer rt-revoked"))
		if !errors.Is(err, ErrRefreshTokenRevoked) {
			t.Errorf("error = %v, want %v", err, ErrRefreshTokenRevoked)
		}
	})

	t.Run("rejects expired refresh token", func(t *testing.T) {
		store := &fakeStore{
			getFn: func(ctx context.Context, token string) (database.RefreshToken, error) {
				return database.RefreshToken{
					Token:     "rt-expired",
					ExpiresAt: time.Now().Add(-time.Hour),
				}, nil
			},
		}
		s := newTestSession(store, &fakeAuthenticator{})

		_, err := s.Renew(context.Background(), testRequest(http.MethodPost, "/api/refresh", "Bearer rt-expired"))
		if !errors.Is(err, ErrRefreshTokenExpired) {
			t.Errorf("error = %v, want %v", err, ErrRefreshTokenExpired)
		}
	})

	t.Run("propagates store error", func(t *testing.T) {
		store := &fakeStore{
			getFn: func(ctx context.Context, token string) (database.RefreshToken, error) {
				return database.RefreshToken{}, errors.New("database down")
			},
		}
		s := newTestSession(store, &fakeAuthenticator{})

		_, err := s.Renew(context.Background(), testRequest(http.MethodPost, "/api/refresh", "Bearer rt-valid"))
		if err == nil {
			t.Fatal("Renew: expected error, got nil")
		}
	})
}

func TestEnd(t *testing.T) {
	t.Run("revokes refresh token", func(t *testing.T) {
		store := &fakeStore{revokeFn: func(ctx context.Context, token string) error { return nil }}
		s := newTestSession(store, &fakeAuthenticator{})

		err := s.End(context.Background(), testRequest(http.MethodPost, "/api/revoke", "Bearer rt-revoke-me"))
		if err != nil {
			t.Fatalf("End: %v", err)
		}
		if store.revoked != "rt-revoke-me" {
			t.Errorf("store received token = %q, want %q", store.revoked, "rt-revoke-me")
		}
	})

	t.Run("rejects missing authorization header", func(t *testing.T) {
		store := &fakeStore{}
		s := newTestSession(store, &fakeAuthenticator{})

		err := s.End(context.Background(), testRequest(http.MethodPost, "/api/revoke", ""))
		if !errors.Is(err, ErrRefreshTokenInvalid) {
			t.Errorf("error = %v, want %v", err, ErrRefreshTokenInvalid)
		}
	})

	t.Run("propagates store error", func(t *testing.T) {
		store := &fakeStore{revokeFn: func(ctx context.Context, token string) error { return errors.New("database down") }}
		s := newTestSession(store, &fakeAuthenticator{})

		err := s.End(context.Background(), testRequest(http.MethodPost, "/api/revoke", "Bearer rt-revoke-me"))
		if err == nil {
			t.Fatal("End: expected error, got nil")
		}
	})
}
