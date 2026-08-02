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
	"github.com/thalesraymond/go-http-server/internal/auth"
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
			body: `{"body":"hello world"}`,
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
			body:       `{"body":""}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Chirp body cannot be empty",
		},
		{
			name:       "too long body",
			body:       `{"body":"` + strings.Repeat("a", 141) + `"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "140",
		},
		{
			name: "profanity cleaned",
			body: `{"body":"what a kerfuffle"}`,
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
			body: `{"body":"hello"}`,
			mockFn: func(_ context.Context, _ database.CreateChirpParams) (database.Chirp, error) {
				return database.Chirp{}, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Failed to create chirp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockChirpStore{}
			if tt.mockFn != nil {
				store.createChirpFn = tt.mockFn
			}

			cfg, h := newTestChirpHandler(store)

			token, err := auth.MakeJWT(userID, cfg.Secret, time.Hour)
			if err != nil {
				t.Fatalf("make jwt: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/chirps", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()

			h.CreateChirp(rr, req)

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
			store := &mockChirpStore{}
			store.getAllChirpsFn = tt.mockFn

			_, h := newTestChirpHandler(store)

			req := httptest.NewRequest(http.MethodGet, "/api/chirps", nil)
			rr := httptest.NewRecorder()

			h.GetChirps(rr, req)

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
			store := &mockChirpStore{}
			if tt.mockFn != nil {
				store.getChirpByIDFn = tt.mockFn
			}

			_, h := newTestChirpHandler(store)

			req := httptest.NewRequest(http.MethodGet, "/api/chirps/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			rr := httptest.NewRecorder()

			h.GetChirpByID(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}

func TestHandlerUpdateChirpByID(t *testing.T) {
	t.Run("stub returns ok", func(t *testing.T) {
		_, h := newTestChirpHandler(nil)

		req := httptest.NewRequest(http.MethodPut, "/api/chirps", nil)
		rr := httptest.NewRecorder()

		h.UpdateChirpByID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}

func TestHandlerDeleteChirpByID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	userID := uuid.New()
	otherUserID := uuid.New()
	chirpID := uuid.New()

	tests := []struct {
		name         string
		pathID       string
		ctxUserID    uuid.UUID
		mockGetFn    func(ctx context.Context, id uuid.UUID) (database.Chirp, error)
		mockDeleteFn func(ctx context.Context, id uuid.UUID) error
		wantStatus   int
		wantBody     string
	}{
		{
			name:       "missing id",
			wantStatus: http.StatusBadRequest,
			wantBody:   "Missing chirp ID",
		},
		{
			name:       "invalid uuid",
			pathID:     "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid chirp ID",
		},
		{
			name:      "not found",
			pathID:    chirpID.String(),
			ctxUserID: userID,
			mockGetFn: func(_ context.Context, _ uuid.UUID) (database.Chirp, error) {
				return database.Chirp{}, sql.ErrNoRows
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Chirp not found",
		},
		{
			name:      "forbidden",
			pathID:    chirpID.String(),
			ctxUserID: userID,
			mockGetFn: func(_ context.Context, _ uuid.UUID) (database.Chirp, error) {
				return database.Chirp{ID: chirpID, Body: "hello", UserID: otherUserID, CreatedAt: now, UpdatedAt: now}, nil
			},
			wantStatus: http.StatusForbidden,
			wantBody:   "not authorized",
		},
		{
			name:      "delete error",
			pathID:    chirpID.String(),
			ctxUserID: userID,
			mockGetFn: func(_ context.Context, _ uuid.UUID) (database.Chirp, error) {
				return database.Chirp{ID: chirpID, Body: "hello", UserID: userID, CreatedAt: now, UpdatedAt: now}, nil
			},
			mockDeleteFn: func(_ context.Context, _ uuid.UUID) error {
				return errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Failed to delete chirp by ID",
		},
		{
			name:      "success",
			pathID:    chirpID.String(),
			ctxUserID: userID,
			mockGetFn: func(_ context.Context, _ uuid.UUID) (database.Chirp, error) {
				return database.Chirp{ID: chirpID, Body: "hello", UserID: userID, CreatedAt: now, UpdatedAt: now}, nil
			},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockChirpStore{}
			if tt.mockGetFn != nil {
				store.getChirpByIDFn = tt.mockGetFn
			}
			if tt.mockDeleteFn != nil {
				store.deleteChirpByIDFn = tt.mockDeleteFn
			}

			_, h := newTestChirpHandler(store)

			req := httptest.NewRequest(http.MethodDelete, "/api/chirps/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			if tt.ctxUserID != uuid.Nil {
				ctx := context.WithValue(req.Context(), userIDKey, tt.ctxUserID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			h.DeleteChirpByID(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if tt.wantBody != "" && !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}
