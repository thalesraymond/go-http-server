package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// MakeRefreshToken is a variable so handlers can override it in tests.
var MakeRefreshToken = func() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
