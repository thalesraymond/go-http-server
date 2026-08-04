package auth

import "testing"

func TestHashPassword(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	password := "testpassword123"

	hash, err := authenticator.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}
	if hash == password {
		t.Fatal("HashPassword returned the plain password as the hash")
	}
}

func TestHashPassword_SaltsEachHash(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	password := "testpassword123"

	hash1, err := authenticator.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	hash2, err := authenticator.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash1 == hash2 {
		t.Fatal("expected different hashes for the same password (argon2id salts)")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	authenticator := NewRealAuthenticator("test-secret")
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	hash, err := authenticator.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	ok, err := authenticator.CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash returned error for correct password: %v", err)
	}
	if !ok {
		t.Fatal("CheckPasswordHash returned false for correct password")
	}

	ok, err = authenticator.CheckPasswordHash(wrongPassword, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash returned error for wrong password: %v", err)
	}
	if ok {
		t.Fatal("CheckPasswordHash returned true for wrong password")
	}
}
