package auth

import "github.com/alexedwards/argon2id"

// HashPassword is a variable so handlers can override it in tests.
var HashPassword = func(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

// CheckPasswordHash is a variable so handlers can override it in tests.
var CheckPasswordHash = func(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
