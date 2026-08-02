package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

func TestHandlerCreateChirp(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()

	tests := []struct {
		name       string
		body       string
		mockFn     func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "valid chirp",
			body: `{"body":"hello world","user_id":"` + userID.String() + `"}`,
			mockFn: func(_ context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
				return database.Chirp{
					ID:        arg.ID,
					Body:      arg.Body,
					UserID:    arg.UserID,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody:   "hello world",
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid request payload",
		},
		{
			name:       "empty body",
			body:       `{"body":"","user_id":"` + userID.String() + `"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Chirp body cannot be empty",
		},
		{
			name:       "too long body",
			body:       `{"body":"` + strings.Repeat("a", 141) + `","user_id":"` + userID.String() + `"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "140",
		},
		{
			name: "profanity cleaned",
			body: `{"body":"what a kerfuffle","user_id":"` + userID.String() + `"}`,
			mockFn: func(_ context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
				return database.Chirp{
					ID:        arg.ID,
					Body:      arg.Body,
					UserID:    arg.UserID,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody:   "****",
		},
		{
			name: "database error",
			body: `{"body":"hello","user_id":"` + userID.String() + `"}`,
			mockFn: func(_ context.Context, _ database.CreateChirpParams) (database.Chirp, error) {
				return database.Chirp{}, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Failed to create chirp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			if tt.mockFn != nil {
				mockDB.CreateChirpFn = tt.mockFn
			}

			h := NewChirpHandler(newTestApiConfig(mockDB))

			req := httptest.NewRequest(http.MethodPost, "/api/chirps", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.HandlerCreateChirp(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantBody != "" && !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandlerGetChirps(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context) ([]database.Chirp, error)
		wantStatus int
		wantCount  int
	}{
		{
			name: "returns chirps",
			mockFn: func(_ context.Context) ([]database.Chirp, error) {
				return []database.Chirp{
					{ID: uuid.New(), Body: "first", UserID: userID, CreatedAt: now, UpdatedAt: now},
					{ID: uuid.New(), Body: "second", UserID: userID, CreatedAt: now, UpdatedAt: now},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "empty list",
			mockFn: func(_ context.Context) ([]database.Chirp, error) {
				return nil, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "database error",
			mockFn: func(_ context.Context) ([]database.Chirp, error) {
				return nil, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			mockDB.GetAllChirpsFn = tt.mockFn

			h := NewChirpHandler(newTestApiConfig(mockDB))

			req := httptest.NewRequest(http.MethodGet, "/api/chirps", nil)
			rr := httptest.NewRecorder()

			h.HandlerGetChirps(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp []ChirpResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(resp) != tt.wantCount {
					t.Errorf("count = %d, want %d", len(resp), tt.wantCount)
				}
			}
		})
	}
}

func TestHandlerGetChirpByID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()
	chirpID := uuid.New()

	tests := []struct {
		name       string
		pathID     string
		mockFn     func(ctx context.Context, id uuid.UUID) (database.Chirp, error)
		wantStatus int
	}{
		{
			name:   "found",
			pathID: chirpID.String(),
			mockFn: func(_ context.Context, id uuid.UUID) (database.Chirp, error) {
				return database.Chirp{ID: id, Body: "hello", UserID: userID, CreatedAt: now, UpdatedAt: now}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing id",
			pathID:     "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid uuid",
			pathID:     "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "not found",
			pathID: chirpID.String(),
			mockFn: func(_ context.Context, _ uuid.UUID) (database.Chirp, error) {
				return database.Chirp{}, sql.ErrNoRows
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "database error",
			pathID: chirpID.String(),
			mockFn: func(_ context.Context, _ uuid.UUID) (database.Chirp, error) {
				return database.Chirp{}, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			if tt.mockFn != nil {
				mockDB.GetChirpByIDFn = tt.mockFn
			}

			h := NewChirpHandler(newTestApiConfig(mockDB))

			req := httptest.NewRequest(http.MethodGet, "/api/chirps/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			rr := httptest.NewRecorder()

			h.HandlerGetChirpByID(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}

func TestHandlerUpdateChirpByID(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		mockDB := &MockDatabase{}
		h := NewChirpHandler(newTestApiConfig(mockDB))

		req := httptest.NewRequest(http.MethodPut, "/api/chirps", nil)
		rr := httptest.NewRecorder()

		h.HandlerUpdateChirpByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}

func TestHandlerDeleteChirpByID(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		mockDB := &MockDatabase{}
		h := NewChirpHandler(newTestApiConfig(mockDB))

		req := httptest.NewRequest(http.MethodDelete, "/api/chirps", nil)
		rr := httptest.NewRecorder()

		h.HandlerDeleteChirpByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}
