package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

// mockChirpStore implements chirpStore with per-method stub functions.
// Every method also records its arguments so tests can assert on them.
type mockChirpStore struct {
	createChirpFn      func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error)
	getAllChirpsAscFn  func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error)
	getAllChirpsDescFn func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error)
	getChirpByIDFn     func(ctx context.Context, id uuid.UUID) (database.Chirp, error)
	deleteChirpByIDFn  func(ctx context.Context, id uuid.UUID) error

	createChirpParams        database.CreateChirpParams
	getAllChirpsAscAuthorID  uuid.NullUUID
	getAllChirpsDescAuthorID uuid.NullUUID
	getChirpByIDID           uuid.UUID
	deleteChirpByIDID        uuid.UUID
}

func (m *mockChirpStore) CreateChirp(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
	m.createChirpParams = arg
	if m.createChirpFn == nil {
		return database.Chirp{}, errUnstubbed
	}
	return m.createChirpFn(ctx, arg)
}

func (m *mockChirpStore) GetAllChirpsAsc(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
	m.getAllChirpsAscAuthorID = authorID
	if m.getAllChirpsAscFn == nil {
		return nil, errUnstubbed
	}
	return m.getAllChirpsAscFn(ctx, authorID)
}

func (m *mockChirpStore) GetAllChirpsDesc(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
	m.getAllChirpsDescAuthorID = authorID
	if m.getAllChirpsDescFn == nil {
		return nil, errUnstubbed
	}
	return m.getAllChirpsDescFn(ctx, authorID)
}

func (m *mockChirpStore) GetChirpByID(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
	m.getChirpByIDID = id
	if m.getChirpByIDFn == nil {
		return database.Chirp{}, errUnstubbed
	}
	return m.getChirpByIDFn(ctx, id)
}

func (m *mockChirpStore) DeleteChirpByID(ctx context.Context, id uuid.UUID) error {
	m.deleteChirpByIDID = id
	if m.deleteChirpByIDFn == nil {
		return errUnstubbed
	}
	return m.deleteChirpByIDFn(ctx, id)
}

// newTestChirpHandler wires a ChirpHandler around a mock store.
func newTestChirpHandler(mock chirpStore) (*ApiConfig, *ChirpHandler) {
	cfg := newTestApiConfig()
	return cfg, &ChirpHandler{apiConfig: cfg, store: mock}
}

// newTestChirpRouter registers the handler routes so tests can exercise path
// parameters (r.PathValue) and the RequireAuth middleware.
func newTestChirpRouter(mock chirpStore) (*ApiConfig, http.Handler) {
	cfg, h := newTestChirpHandler(mock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return cfg, mux
}

func TestCreateChirp(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name       string
		body       string
		setupMock  func(m *mockChirpStore)
		wantStatus int
		wantError  string
		wantBody   string
	}{
		{
			name: "creates chirp",
			body: `{"body": "Hello world"}`,
			setupMock: func(m *mockChirpStore) {
				m.createChirpFn = func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
					return database.Chirp{
						ID:        arg.ID,
						Body:      arg.Body,
						UserID:    arg.UserID,
						CreatedAt: createdAt,
						UpdatedAt: createdAt,
					}, nil
				}
			},
			wantStatus: http.StatusCreated,
			wantBody:   "Hello world",
		},
		{
			name: "cleans profanity before storing",
			body: `{"body": "I heard kerfuffle is bad"}`,
			setupMock: func(m *mockChirpStore) {
				m.createChirpFn = func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
					return database.Chirp{
						ID:        arg.ID,
						Body:      arg.Body,
						UserID:    arg.UserID,
						CreatedAt: createdAt,
						UpdatedAt: createdAt,
					}, nil
				}
			},
			wantStatus: http.StatusCreated,
			wantBody:   "I heard **** is bad",
		},
		{
			name:       "rejects empty body",
			body:       `{"body": ""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Chirp body cannot be empty",
		},
		{
			name:       "rejects body over 140 characters",
			body:       `{"body": "` + strings.Repeat("a", 141) + `"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Chirp body exceeds maximum length of 140 characters",
		},
		{
			name:       "rejects invalid JSON",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request payload",
		},
		{
			name: "handles store error",
			body: `{"body": "Hello world"}`,
			setupMock: func(m *mockChirpStore) {
				m.createChirpFn = func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
					return database.Chirp{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to create chirp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockChirpStore{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			_, h := newTestChirpHandler(mock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodPost, "/api/chirps", tt.body)

			h.CreateChirp(rr, req, testUserID)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.createChirpParams.UserID != testUserID {
				t.Errorf("store received userID = %v, want %v", mock.createChirpParams.UserID, testUserID)
			}

			var got ChirpResponse
			decodeBody(t, rr, &got)
			if got.Body != tt.wantBody {
				t.Errorf("response body = %q, want %q", got.Body, tt.wantBody)
			}
			if got.ID != mock.createChirpParams.ID {
				t.Errorf("response id = %v, want stored chirp id %v", got.ID, mock.createChirpParams.ID)
			}
			if got.UserID != testUserID {
				t.Errorf("response user_id = %v, want %v", got.UserID, testUserID)
			}
		})
	}
}

func TestGetChirps(t *testing.T) {
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	chirps := []database.Chirp{
		{
			ID:        uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
			Body:      "first",
			UserID:    userA,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
			Body:      "second",
			UserID:    userB,
			CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name               string
		target             string
		setupMock          func(m *mockChirpStore)
		wantStatus         int
		wantError          string
		wantCount          int
		wantAuthorFiltered bool
		wantAuthor         uuid.UUID
	}{
		{
			name:   "returns chirps ascending by default",
			target: "/api/chirps",
			setupMock: func(m *mockChirpStore) {
				m.getAllChirpsAscFn = func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
					return chirps, nil
				}
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:   "returns chirps descending when requested",
			target: "/api/chirps?sort=desc",
			setupMock: func(m *mockChirpStore) {
				m.getAllChirpsDescFn = func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
					return chirps, nil
				}
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:   "filters by author",
			target: "/api/chirps?author_id=" + userA.String(),
			setupMock: func(m *mockChirpStore) {
				m.getAllChirpsAscFn = func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
					return nil, nil
				}
			},
			wantStatus:         http.StatusOK,
			wantCount:          0,
			wantAuthorFiltered: true,
			wantAuthor:         userA,
		},
		{
			name:       "rejects invalid sort parameter",
			target:     "/api/chirps?sort=sideways",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid sort parameter. Must be 'asc' or 'desc'",
		},
		{
			name:       "rejects invalid author id",
			target:     "/api/chirps?author_id=not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid author ID",
		},
		{
			name:   "returns empty list when no chirps exist",
			target: "/api/chirps",
			setupMock: func(m *mockChirpStore) {
				m.getAllChirpsAscFn = func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
					return []database.Chirp{}, nil
				}
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:   "handles store error",
			target: "/api/chirps",
			setupMock: func(m *mockChirpStore) {
				m.getAllChirpsAscFn = func(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error) {
					return nil, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to get chirps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockChirpStore{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			_, h := newTestChirpHandler(mock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodGet, tt.target, "")

			h.GetChirps(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if tt.wantAuthorFiltered {
				want := uuid.NullUUID{UUID: tt.wantAuthor, Valid: true}
				if mock.getAllChirpsAscAuthorID != want {
					t.Errorf("store received author filter = %v, want %v", mock.getAllChirpsAscAuthorID, want)
				}
			} else if mock.getAllChirpsAscAuthorID.Valid {
				t.Errorf("store received unexpected author filter %v", mock.getAllChirpsAscAuthorID)
			}
			if mock.getAllChirpsDescAuthorID.Valid {
				t.Errorf("desc store call received unexpected author filter %v", mock.getAllChirpsDescAuthorID)
			}

			var got []ChirpResponse
			decodeBody(t, rr, &got)
			if len(got) != tt.wantCount {
				t.Errorf("got %d chirps, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestGetChirpByID(t *testing.T) {
	chirpID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	chirp := database.Chirp{
		ID:        chirpID,
		Body:      "Hello world",
		UserID:    testUserID,
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	tests := []struct {
		name       string
		target     string
		setupMock  func(m *mockChirpStore)
		wantStatus int
		wantError  string
		wantChirp  bool
	}{
		{
			name:   "returns chirp",
			target: "/api/chirps/" + chirpID.String(),
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return chirp, nil
				}
			},
			wantStatus: http.StatusOK,
			wantChirp:  true,
		},
		{
			name:       "rejects invalid id",
			target:     "/api/chirps/not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid chirp ID",
		},
		{
			name:   "returns 404 when chirp not found",
			target: "/api/chirps/" + chirpID.String(),
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return database.Chirp{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
			wantError:  "Chirp not found",
		},
		{
			name:   "handles store error",
			target: "/api/chirps/" + chirpID.String(),
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return database.Chirp{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to get chirp by ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockChirpStore{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			_, router := newTestChirpRouter(mock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodGet, tt.target, "")

			router.ServeHTTP(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.getChirpByIDID != chirpID {
				t.Errorf("store received id = %v, want %v", mock.getChirpByIDID, chirpID)
			}

			var got ChirpResponse
			decodeBody(t, rr, &got)
			if got.ID != chirp.ID || got.Body != chirp.Body || got.UserID != chirp.UserID {
				t.Errorf("response = %+v, want chirp %+v", got, chirp)
			}
		})
	}
}

func TestDeleteChirpByID(t *testing.T) {
	chirpID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	chirpOwner := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	chirp := database.Chirp{
		ID:     chirpID,
		Body:   "mine",
		UserID: chirpOwner,
	}

	tests := []struct {
		name       string
		target     string
		withAuth   bool
		authAs     uuid.UUID
		setupMock  func(m *mockChirpStore)
		wantStatus int
		wantError  string
	}{
		{
			name:     "deletes own chirp",
			target:   "/api/chirps/" + chirpID.String(),
			withAuth: true,
			authAs:   chirpOwner,
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return chirp, nil
				}
				m.deleteChirpByIDFn = func(ctx context.Context, id uuid.UUID) error {
					return nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "requires authentication",
			target:     "/api/chirps/" + chirpID.String(),
			withAuth:   false,
			wantStatus: http.StatusUnauthorized,
			wantError:  "Unauthorized",
		},
		{
			name:       "rejects invalid id",
			target:     "/api/chirps/not-a-uuid",
			withAuth:   true,
			authAs:     testUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid chirp ID",
		},
		{
			name:     "returns 404 when chirp not found",
			target:   "/api/chirps/" + chirpID.String(),
			withAuth: true,
			authAs:   testUserID,
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return database.Chirp{}, sql.ErrNoRows
				}
			},
			wantStatus: http.StatusNotFound,
			wantError:  "Chirp not found",
		},
		{
			name:     "forbids deleting another user's chirp",
			target:   "/api/chirps/" + chirpID.String(),
			withAuth: true,
			authAs:   testUserID,
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return chirp, nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantError:  "You are not authorized to delete this chirp",
		},
		{
			name:     "handles get store error",
			target:   "/api/chirps/" + chirpID.String(),
			withAuth: true,
			authAs:   testUserID,
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return database.Chirp{}, errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to get chirp by ID",
		},
		{
			name:     "handles delete store error",
			target:   "/api/chirps/" + chirpID.String(),
			withAuth: true,
			authAs:   chirpOwner,
			setupMock: func(m *mockChirpStore) {
				m.getChirpByIDFn = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
					return chirp, nil
				}
				m.deleteChirpByIDFn = func(ctx context.Context, id uuid.UUID) error {
					return errors.New("database down")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to delete chirp by ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockChirpStore{}
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			cfg, router := newTestChirpRouter(mock)
			rr := httptest.NewRecorder()
			req := newTestRequest(t, http.MethodDelete, tt.target, "")
			if tt.withAuth {
				req = withAuth(req, cfg, tt.authAs)
			}

			router.ServeHTTP(rr, req)

			wantStatus(t, rr, tt.wantStatus)
			if tt.wantError != "" {
				wantErrorBody(t, rr, tt.wantError)
				return
			}

			if mock.deleteChirpByIDID != chirpID {
				t.Errorf("store received delete id = %v, want %v", mock.deleteChirpByIDID, chirpID)
			}
		})
	}
}
