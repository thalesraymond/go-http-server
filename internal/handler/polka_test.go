package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

func TestPolkaWebhook(t *testing.T) {
	validUserID := uuid.New()

	tests := []struct {
		name            string
		body            string
		apiKey          string
		updateFn        func(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error)
		wantStatus      int
		wantStoreCalled bool
		wantUserID      uuid.UUID
		wantChirpyRed   bool
	}{
		{
			name:            "valid upgrade event sets chirpy red",
			body:            `{"event":"user.upgraded","data":{"user_id":"` + validUserID.String() + `"}}`,
			wantStatus:      http.StatusNoContent,
			wantStoreCalled: true,
			wantUserID:      validUserID,
			wantChirpyRed:   true,
		},
		{
			name:            "wrong api key is rejected",
			body:            `{"event":"user.upgraded","data":{"user_id":"` + validUserID.String() + `"}}`,
			apiKey:          "wrong-key",
			wantStatus:      http.StatusUnauthorized,
			wantStoreCalled: false,
		},
		{
			name:            "non-upgrade event is ignored",
			body:            `{"event":"user.downgraded","data":{"user_id":"` + validUserID.String() + `"}}`,
			wantStatus:      http.StatusNoContent,
			wantStoreCalled: false,
		},
		{
			name:       "invalid json",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user id",
			body:       `{"event":"user.upgraded","data":{"user_id":"not-a-uuid"}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			body: `{"event":"user.upgraded","data":{"user_id":"` + validUserID.String() + `"}}`,
			updateFn: func(_ context.Context, _ database.UpdateChirpyRedFlagParams) (database.User, error) {
				return database.User{}, sql.ErrNoRows
			},
			wantStatus:      http.StatusNotFound,
			wantStoreCalled: true,
			wantUserID:      validUserID,
			wantChirpyRed:   true,
		},
		{
			name: "database error",
			body: `{"event":"user.upgraded","data":{"user_id":"` + validUserID.String() + `"}}`,
			updateFn: func(_ context.Context, _ database.UpdateChirpyRedFlagParams) (database.User, error) {
				return database.User{}, errors.New("db error")
			},
			wantStatus:      http.StatusInternalServerError,
			wantStoreCalled: true,
			wantUserID:      validUserID,
			wantChirpyRed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockPolkaStore{}
			var gotUserID uuid.UUID
			var gotChirpyRed bool

			store.updateChirpyRedFlagFn = func(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error) {
				gotUserID = arg.ID
				gotChirpyRed = arg.IsChirpyRed
				if tt.updateFn != nil {
					return tt.updateFn(ctx, arg)
				}
				return database.User{}, nil
			}

			apiKey := tt.apiKey
			if apiKey == "" {
				apiKey = testPolkaKey
			}

			req := httptest.NewRequest(http.MethodPost, "/api/polka/webhooks", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "ApiKey "+apiKey)
			rr := httptest.NewRecorder()

			newTestPolkaRoute(store).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantStoreCalled {
				if gotUserID != tt.wantUserID {
					t.Errorf("store called with user ID %s, want %s", gotUserID, tt.wantUserID)
				}
				if gotChirpyRed != tt.wantChirpyRed {
					t.Errorf("store called with is_chirpy_red = %t, want %t", gotChirpyRed, tt.wantChirpyRed)
				}
			}
		})
	}
}

func TestPolkaWebhookNoContent(t *testing.T) {
	// The webhook always answers 204 No Content on success, with an empty body.
	validUserID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/api/polka/webhooks", bytes.NewBufferString(
		`{"event":"user.upgraded","data":{"user_id":"`+validUserID.String()+`"}}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+testPolkaKey)
	rr := httptest.NewRecorder()

	newTestPolkaRoute(nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rr.Body.String())
	}
}

func TestPolkaWebhookErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		updateFn   func(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error)
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid json",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "invalid user id",
			body:       `{"event":"user.upgraded","data":{"user_id":"not-a-uuid"}}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid user ID",
		},
		{
			name: "user not found",
			body: `{"event":"user.upgraded","data":{"user_id":"` + uuid.NewString() + `"}}`,
			updateFn: func(_ context.Context, _ database.UpdateChirpyRedFlagParams) (database.User, error) {
				return database.User{}, sql.ErrNoRows
			},
			wantStatus: http.StatusNotFound,
			wantError:  "User not found",
		},
		{
			name: "database error",
			body: `{"event":"user.upgraded","data":{"user_id":"` + uuid.NewString() + `"}}`,
			updateFn: func(_ context.Context, _ database.UpdateChirpyRedFlagParams) (database.User, error) {
				return database.User{}, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to update user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockPolkaStore{}
			if tt.updateFn != nil {
				store.updateChirpyRedFlagFn = tt.updateFn
			}

			req := httptest.NewRequest(http.MethodPost, "/api/polka/webhooks", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "ApiKey "+testPolkaKey)
			rr := httptest.NewRecorder()

			newTestPolkaRoute(store).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			var resp map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if resp["error"] != tt.wantError {
				t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
			}
		})
	}
}
