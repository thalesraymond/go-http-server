package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type Authenticator interface {
	HashPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) (bool, error)
	MakeJWT(userID uuid.UUID, expiresIn time.Duration) (string, error)
	ValidateJWT(tokenString string) (uuid.UUID, error)
	MakeRefreshToken() string
}


type RealAuthenticator struct {
	secret string
}


func NewRealAuthenticator(secret string) *RealAuthenticator {
	return &RealAuthenticator{secret: secret}
}

func (a *RealAuthenticator) HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func (a *RealAuthenticator) CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func (a *RealAuthenticator) MakeJWT(userID uuid.UUID, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secret))
}

func (a *RealAuthenticator) ValidateJWT(tokenString string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.secret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	if !token.Valid {
		return uuid.Nil, jwt.ErrSignatureInvalid
	}

	if claims.Subject == "" {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func (a *RealAuthenticator) MakeRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
