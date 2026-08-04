package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

// newTestPolkaHandler wires a PolkaHandler around a mock store.
func newTestPolkaHandler(mock polkaStore) *PolkaHandler {
	return &PolkaHandler{apiConfig: newTestApiConfig(), store: mock}
}

func TestPolkaWebhook(t *testing.T) {
	userID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	tests := []struct {
		name            string
		body            string
		setupMock       func(m *mockQuerier)
		wantStatus      int
		wantError       string
		wantStoreCalled bool
	}{
		{
			name: "upgrades user to chirpy red",
			body: `{"event": "user.upgraded", "data": {"user_id": "` + userID.String() + `"}}`,
			setupMock: func(m *mockQuerier) {
				m.updateChirpyRedFlagFn = func(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error) {
					return database.User{ID: arg.ID, IsChirpyRed: arg.IsChirpyRed}, nil
				}
			},
			wantStatus:      http.StatusNoContent,
			wantStoreCalled: true,
		},
		{
			name:            "ignores non-upgrade events",
			body:            `{"event": "user.paid", "data": {"user_id": "` + userID.String() + `"}}`,
			wantStatus:      http.StatusNoContent,
			wantStoreCalled: false,
		},
		{
			name:       "rejects invalid JSON",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name:       "rejects invalid user id",
			body:       `{"event": "user.upgraded", "data": {"user_id": "not-a-uuid"}}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid user ID",
		},
		{
			name: "returns 404 for unknown user",
			body: `{"event": "user.upgraded", "data": {"user_id": "` + userID.String() + `"}}`,
			setupMock: func(m *mockQuerier) {
				m.updateChirpyRedFlagFn = func(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error) {
					return database.User{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
			wantError:  "User not found",
		},
		{
			name: "handles store error",
			body: `{"event": "user.upgraded", "data": {"user_id": "` + userID.String() + `"}}`,
			setupMock: func(m *mockQuerier) {
				m.updateChirpyRedFlagFn = func(ctx context.Context, arg database.UpdateChirpyRedFlagParams) (database.User, error) {
					return database.User{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to update user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockQuerier{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			h := newTestPolkaHandler(mock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPost, "/api/polka/webhooks", tt.body)

			h.PolkaWebhook(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if tt.wantStoreCalled {
				if mock.updateChirpyRedFlagCalls != 1 {
					t.Errorf("store called %d times, want 1", mock.updateChirpyRedFlagCalls)
				}
				if mock.updateChirpyRedFlagParams.ID != userID {
					t.Errorf("store received user id = %v, want %v", mock.updateChirpyRedFlagParams.ID, userID)
				}
				if !mock.updateChirpyRedFlagParams.IsChirpyRed {
					t.Error("store received is_chirpy_red = false, want true")
				}
			} else if mock.updateChirpyRedFlagCalls != 0 {
				t.Errorf("store called %d times, want 0", mock.updateChirpyRedFlagCalls)
			}
		})
	}
}
