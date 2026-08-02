package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func TestMakeJWT_Success(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}
	if tokenString == "" {
		t.Fatal("MakeJWT returned empty token string")
	}
}

func TestMakeJWT_Claims(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	before := time.Now().UTC()

	tokenString, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	after := time.Now().UTC()

	claims := jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("ParseWithClaims returned error: %v", err)
	}
	if !parsedToken.Valid {
		t.Fatal("parsed token is not valid")
	}

	if claims.Issuer != "chirpy-access" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "chirpy-access")
	}
	if claims.Subject != userID.String() {
		t.Errorf("Subject = %q, want %q", claims.Subject, userID.String())
	}

	if claims.IssuedAt == nil {
		t.Fatal("IssuedAt is nil")
	}
	if claims.IssuedAt.Before(before.Add(-time.Second)) || claims.IssuedAt.After(after.Add(time.Second)) {
		t.Errorf("IssuedAt = %v, want between %v and %v", claims.IssuedAt.Time, before, after)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
	expectedExpiresAt := claims.IssuedAt.Add(expiresIn)
	if claims.ExpiresAt.Before(expectedExpiresAt.Add(-time.Second)) || claims.ExpiresAt.After(expectedExpiresAt.Add(time.Second)) {
		t.Errorf("ExpiresAt = %v, want approximately %v", claims.ExpiresAt.Time, expectedExpiresAt)
	}
}

func TestMakeJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	claims := jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})
	if err == nil {
		t.Fatal("expected error when parsing with wrong secret, got nil")
	}
}

func TestMakeJWT_Expired(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	tokenString, err := MakeJWT(userID, secret, time.Millisecond)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	claims := jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestMakeJWT_DifferentUserIDs(t *testing.T) {
	secret := "test-secret"
	expiresIn := time.Hour

	token1, err := MakeJWT(uuid.New(), secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	token2, err := MakeJWT(uuid.New(), secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected different tokens for different user IDs")
	}
}
