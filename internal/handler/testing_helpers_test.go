package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/auth"
)

// testUserID is the fixed identity used to exercise authenticated handlers.
var testUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// errUnstubbed is returned by mock stores when a method is called without a
// stub function, so the test fails loudly instead of panicking.
var errUnstubbed = errors.New("mock method not stubbed")

// testLogger discards all log output so handler tests run quietly.
type testLogger struct{}

func (testLogger) Error(msg string, err error) {}
func (testLogger) Info(msg string)             {}
func (testLogger) Debug(msg string)            {}
func (testLogger) Warn(msg string)             {}

// mockAuthenticator implements auth.Authenticator with per-method stub
// functions. Each method records its arguments so tests can assert on them.
type mockAuthenticator struct {
	hashPasswordFn     func(password string) (string, error)
	checkPasswordFn    func(password, hash string) (bool, error)
	makeJWTFn          func(userID uuid.UUID, expiresIn time.Duration) (string, error)
	validateJWTFn      func(tokenString string) (uuid.UUID, error)
	makeRefreshTokenFn func() string

	hashPasswordArg   string
	checkPasswordArgs struct{ password, hash string }
	makeJWTArgs       struct {
		userID    uuid.UUID
		expiresIn time.Duration
	}
	validateJWTArg        string
	makeRefreshTokenCalls int
}

// newDefaultMockAuthenticator returns a mockAuthenticator with sensible
// defaults: hashing prefixes "hashed:", password check always succeeds,
// JWT creation succeeds with a fixed token, refresh token returns a
// fixed string. ValidateJWT has no stub, so it falls back to returning
// the userID from the most recent MakeJWT call, mirroring the real
// authenticator's round-trip behavior.
func newDefaultMockAuthenticator() *mockAuthenticator {
	return &mockAuthenticator{
		hashPasswordFn: func(password string) (string, error) {
			return "hashed:" + password, nil
		},
		checkPasswordFn: func(password, hash string) (bool, error) {
			return true, nil
		},
		makeJWTFn: func(userID uuid.UUID, expiresIn time.Duration) (string, error) {
			return "test-access-token", nil
		},
		makeRefreshTokenFn: func() string {
			return "test-refresh-token"
		},
	}
}

func (m *mockAuthenticator) HashPassword(password string) (string, error) {
	m.hashPasswordArg = password
	if m.hashPasswordFn == nil {
		return "", errUnstubbed
	}
	return m.hashPasswordFn(password)
}

func (m *mockAuthenticator) CheckPasswordHash(password, hash string) (bool, error) {
	m.checkPasswordArgs.password = password
	m.checkPasswordArgs.hash = hash
	if m.checkPasswordFn == nil {
		return false, errUnstubbed
	}
	return m.checkPasswordFn(password, hash)
}

func (m *mockAuthenticator) MakeJWT(userID uuid.UUID, expiresIn time.Duration) (string, error) {
	m.makeJWTArgs.userID = userID
	m.makeJWTArgs.expiresIn = expiresIn
	if m.makeJWTFn == nil {
		return "", errUnstubbed
	}
	return m.makeJWTFn(userID, expiresIn)
}

func (m *mockAuthenticator) ValidateJWT(tokenString string) (uuid.UUID, error) {
	m.validateJWTArg = tokenString
	if m.validateJWTFn == nil {
		// Default: mimic the real authenticator and return the userID
		// embedded by the most recent MakeJWT call.
		return m.makeJWTArgs.userID, nil
	}
	return m.validateJWTFn(tokenString)
}

func (m *mockAuthenticator) MakeRefreshToken() string {
	m.makeRefreshTokenCalls++
	if m.makeRefreshTokenFn == nil {
		return errUnstubbed.Error()
	}
	return m.makeRefreshTokenFn()
}

// newTestHandshake returns an AuthHandshake wired for tests: a silent
// logger, a default mockAuthenticator, and a fixed Polka key.
func newTestHandshake() *AuthHandshake {
	return NewAuthHandshake(testLogger{}, newDefaultMockAuthenticator(), "test-polka-key")
}

// newTestRequest builds a request with a JSON content type. Pass an empty
// body for requests without one.
func newTestRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rdr)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// withAuth attaches a valid Bearer JWT for userID, signed with the given
// authenticator, so requests pass through RequireAuth.
func withAuth(r *http.Request, authenticator auth.Authenticator, userID uuid.UUID) *http.Request {
	token, err := authenticator.MakeJWT(userID, time.Hour)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

// wantStatus fails the test unless the recorder holds the expected status.
func wantStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("status = %d, want %d (body: %q)", rr.Code, want, rr.Body.String())
	}
}

// decodeBody unmarshals the recorded JSON response into dst.
func decodeBody(t *testing.T, rr *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response body %q: %v", rr.Body.String(), err)
	}
}

// wantErrorBody fails the test unless the response carries the standard
// {"error": msg} payload with the expected message.
func wantErrorBody(t *testing.T, rr *httptest.ResponseRecorder, want string) {
	t.Helper()
	var got struct {
		Error string `json:"error"`
	}
	decodeBody(t, rr, &got)
	if got.Error != want {
		t.Errorf("error message = %q, want %q", got.Error, want)
	}
}
