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

// newTestApiConfig returns an ApiConfig wired for tests: a silent logger and
// fixed secrets. It has no database — mock stores are injected straight into
// each handler's unexported store field.
func newTestApiConfig() *ApiConfig {
	return &ApiConfig{
		Logger:   testLogger{},
		Secret:   "test-secret",
		PolkaKey: "test-polka-key",
	}
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

// withAuth attaches a valid Bearer JWT for userID, signed with the config's
// secret, so requests pass through RequireAuth.
func withAuth(r *http.Request, cfg *ApiConfig, userID uuid.UUID) *http.Request {
	token, err := auth.MakeJWT(userID, cfg.Secret, time.Hour)
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
