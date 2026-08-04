package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func TestMakeJWT_Success(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	userID := uuid.New()
	expiresIn := time.Hour

	tokenString, err := authenticator.MakeJWT(userID, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}
	if tokenString == "" {
		t.Fatal("MakeJWT returned empty token string")
	}
}

func TestMakeJWT_Claims(t *testing.T) {
	secret := "test-secret"
	authenticator := NewRealAuthenticator(secret)
	userID := uuid.New()
	expiresIn := time.Hour

	before := time.Now().UTC()

	tokenString, err := authenticator.MakeJWT(userID, expiresIn)
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
	secret := "test-secret"
	authenticator := NewRealAuthenticator(secret)
	userID := uuid.New()
	expiresIn := time.Hour

	tokenString, err := authenticator.MakeJWT(userID, expiresIn)
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
	authenticator := NewRealAuthenticator("test-secret")
	userID := uuid.New()

	tokenString, err := authenticator.MakeJWT(userID, time.Millisecond)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	claims := jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestMakeJWT_DifferentUserIDs(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	expiresIn := time.Hour

	token1, err := authenticator.MakeJWT(uuid.New(), expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	token2, err := authenticator.MakeJWT(uuid.New(), expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected different tokens for different user IDs")
	}
}

func TestValidateJWT_RoundTrip(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	userID := uuid.New()

	tokenString, err := authenticator.MakeJWT(userID, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	got, err := authenticator.ValidateJWT(tokenString)
	if err != nil {
		t.Fatalf("ValidateJWT returned error: %v", err)
	}
	if got != userID {
		t.Errorf("ValidateJWT returned userID = %v, want %v", got, userID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	other := NewRealAuthenticator("wrong-secret")
	userID := uuid.New()

	tokenString, err := authenticator.MakeJWT(userID, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	if _, err := other.ValidateJWT(tokenString); err == nil {
		t.Fatal("expected error for token signed with a different secret, got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	userID := uuid.New()

	tokenString, err := authenticator.MakeJWT(userID, time.Millisecond)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	if _, err := authenticator.ValidateJWT(tokenString); err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateJWT_Malformed(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")

	if _, err := authenticator.ValidateJWT("not-a-jwt"); err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestMakeRefreshToken(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")

	token1 := authenticator.MakeRefreshToken()
	token2 := authenticator.MakeRefreshToken()

	if len(token1) != 64 {
		t.Errorf("refresh token length = %d, want 64", len(token1))
	}
	if token1 == token2 {
		t.Fatal("expected different refresh tokens")
	}
}
