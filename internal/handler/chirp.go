package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

type ChirpHandler struct {
	store     chirpStore
	logger    Logger
	handshake *AuthHandshake
}

type chirpStore interface {
	CreateChirp(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error)
	GetAllChirpsAsc(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error)
	GetAllChirpsDesc(ctx context.Context, authorID uuid.NullUUID) ([]database.Chirp, error)
	GetChirpByID(ctx context.Context, id uuid.UUID) (database.Chirp, error)
	DeleteChirpByID(ctx context.Context, id uuid.UUID) error
}

func NewChirpHandler(store chirpStore, logger Logger, handshake *AuthHandshake) *ChirpHandler {
	return &ChirpHandler{store: store, logger: logger, handshake: handshake}
}

func (h *ChirpHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/chirps", h.handshake.RequireAuth(h.CreateChirp))
	mux.HandleFunc("GET /api/chirps", h.GetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", h.GetChirpByID)
	mux.Handle("DELETE /api/chirps/{id}", h.handshake.RequireAuth(h.DeleteChirpByID))
}

type CreateChirpRequest struct {
	Body string `json:"body"`
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toChirpResponse(chirp database.Chirp) ChirpResponse {
	return ChirpResponse{
		ID:        chirp.ID,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
	}
}

func (h *ChirpHandler) CreateChirp(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	var chirpRequestData CreateChirpRequest

	if err := decodeJSON(w, r, &chirpRequestData); err != nil {
		h.logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	cleanedBody, err := ValidateChirp(chirpRequestData.Body)
	if err != nil {
		msg := "Invalid chirp body"
		if errors.Is(err, ErrChirpEmpty) {
			msg = "Chirp body cannot be empty"
		} else if errors.Is(err, ErrChirpTooLong) {
			msg = "Chirp body exceeds maximum length of 140 characters"
		}
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	createdChirp, err := h.store.CreateChirp(r.Context(), database.CreateChirpParams{
		ID:     uuid.New(),
		Body:   cleanedBody,
		UserID: userID,
	})

	if err != nil {
		h.logger.Error("Failed to create chirp", err)
		writeError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}

	createdChirpDTO := toChirpResponse(createdChirp)

	writeJSON(w, http.StatusCreated, createdChirpDTO)
}

func (h *ChirpHandler) GetChirps(w http.ResponseWriter, r *http.Request) {
	authorId := r.URL.Query().Get("author_id")
	sort := r.URL.Query().Get("sort")
	var authorUUID uuid.UUID
	if authorId != "" {
		var err error
		authorUUID, err = uuid.Parse(authorId)
		if err != nil {
			h.logger.Error("Invalid author ID", err)
			writeError(w, http.StatusBadRequest, "Invalid author ID")
			return
		}
	}

	if sort != "" && sort != "asc" && sort != "desc" {
		writeError(w, http.StatusBadRequest, "Invalid sort parameter. Must be 'asc' or 'desc'")
		return
	}

	var chirps []database.Chirp
	var err error
	if sort == "desc" {
		chirps, err = h.store.GetAllChirpsDesc(r.Context(), uuid.NullUUID{UUID: authorUUID, Valid: authorId != ""})
	} else {
		chirps, err = h.store.GetAllChirpsAsc(r.Context(), uuid.NullUUID{UUID: authorUUID, Valid: authorId != ""})
	}

	if err != nil {
		h.logger.Error("Failed to get chirps", err)
		writeError(w, http.StatusInternalServerError, "Failed to get chirps")
		return
	}

	chirpResponses := make([]ChirpResponse, 0)
	for _, chirp := range chirps {
		chirpResponses = append(chirpResponses, toChirpResponse(chirp))
	}

	writeJSON(w, http.StatusOK, chirpResponses)
}

func (h *ChirpHandler) GetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("id")

	chirpUUID, err := uuid.Parse(chirpID)
	if err != nil {
		h.logger.Error("Invalid chirp ID", err)
		writeError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	chirp, err := h.store.GetChirpByID(r.Context(), chirpUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Chirp not found")
			return
		}

		h.logger.Error("Failed to get chirp by ID", err)
		writeError(w, http.StatusInternalServerError, "Failed to get chirp by ID")
		return
	}

	writeJSON(w, http.StatusOK, toChirpResponse(chirp))
}

func (h *ChirpHandler) DeleteChirpByID(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	chirpID := r.PathValue("id")

	chirpUUID, err := uuid.Parse(chirpID)
	if err != nil {
		h.logger.Error("Invalid chirp ID", err)
		writeError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	existingChirp, err := h.store.GetChirpByID(r.Context(), chirpUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Chirp not found")
			return
		}

		h.logger.Error("Failed to get chirp by ID", err)
		writeError(w, http.StatusInternalServerError, "Failed to get chirp by ID")
		return
	}

	if existingChirp.UserID != userID {
		writeError(w, http.StatusForbidden, "You are not authorized to delete this chirp")
		return
	}

	err = h.store.DeleteChirpByID(r.Context(), chirpUUID)
	if err != nil {
		h.logger.Error("Failed to delete chirp by ID", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete chirp by ID")
		return
	}

	writeNoContent(w)
}
