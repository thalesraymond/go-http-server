package session

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
	"github.com/thalesraymond/go-http-server/internal/database"
)

const (
	AccessTokenTTL  = time.Hour
	RefreshTokenTTL = 24 * time.Hour
)

var (
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid")
	ErrRefreshTokenRevoked = errors.New("refresh token is revoked")
	ErrRefreshTokenExpired = errors.New("refresh token is expired")
)

type Store interface {
	CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	GetRefreshTokenByID(ctx context.Context, token string) (database.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}

type Session struct {
	store Store
	auth  auth.Authenticator
}

func New(store Store, auth auth.Authenticator) *Session {
	return &Session{store: store, auth: auth}
}

// Start issues a session for a verified User: an access token plus a
// long-lived refresh token.
func (s *Session) Start(ctx context.Context, userID uuid.UUID) (string, string, error) {
	accessToken, err := s.auth.MakeJWT(userID, AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken := s.auth.MakeRefreshToken()
	_, err = s.store.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    userID,
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
	})
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// Renew issues a fresh access token for the refresh token presented in the
// request's Authorization header.
func (s *Session) Renew(ctx context.Context, r *http.Request) (string, error) {
	refreshToken, err := bearerToken(r)
	if err != nil {
		return "", ErrRefreshTokenInvalid
	}

	stored, err := s.store.GetRefreshTokenByID(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrRefreshTokenInvalid
		}
		return "", err
	}

	if stored.RevokedAt.Valid {
		return "", ErrRefreshTokenRevoked
	}

	if stored.ExpiresAt.Before(time.Now()) {
		return "", ErrRefreshTokenExpired
	}

	return s.auth.MakeJWT(stored.UserID, AccessTokenTTL)
}

// End revokes the refresh token presented in the request's Authorization
// header.
func (s *Session) End(ctx context.Context, r *http.Request) error {
	refreshToken, err := bearerToken(r)
	if err != nil {
		return ErrRefreshTokenInvalid
	}

	return s.store.RevokeRefreshToken(ctx, refreshToken)
}

func bearerToken(r *http.Request) (string, error) {
	return auth.GetBearerToken(r.Header)
}
