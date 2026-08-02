package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/thalesraymond/go-http-server/internal/database"
)

type ChirpHandler struct {
	apiConfig *ApiConfig
}

func NewChirpHandler(apiConfig *ApiConfig) *ChirpHandler {
	return &ChirpHandler{apiConfig: apiConfig}
}

func (h *ChirpHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/chirps", h.HandlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", h.HandlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", h.HandlerGetChirpByID)
	mux.HandleFunc("PUT /api/chirps", h.HandlerUpdateChirpByID)
	mux.HandleFunc("DELETE /api/chirps", h.HandlerDeleteChirpByID)
}

type CreateChirpRequest struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *ChirpHandler) HandlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	var chirpRequestData CreateChirpRequest

	if err := decodeJSON(w, r, &chirpRequestData); err != nil {
		h.apiConfig.Logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	createdChirp, err := h.apiConfig.Database.CreateChirp(r.Context(), database.CreateChirpParams{
		ID:     uuid.New(),
		Body:   chirpRequestData.Body,
		UserID: chirpRequestData.UserID,
	})

	if err != nil {
		h.apiConfig.Logger.Error("Failed to create chirp", err)
		writeError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}

	createdChirpDTO := ChirpResponse{
		ID:        createdChirp.ID,
		Body:      createdChirp.Body,
		UserID:    createdChirp.UserID,
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
	}

	writeJSON(w, http.StatusCreated, createdChirpDTO)
}

func (h *ChirpHandler) HandlerGetChirps(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to get all chirps
}

func (h *ChirpHandler) HandlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to get a chirp by ID
}

func (h *ChirpHandler) HandlerUpdateChirpByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to update a chirp by ID
}

func (h *ChirpHandler) HandlerDeleteChirpByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to delete a chirp by ID
}
