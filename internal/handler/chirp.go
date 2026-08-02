package handler

import (
	"database/sql"
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
	mux.Handle("POST /api/chirps", h.apiConfig.AuthMiddleware(http.HandlerFunc(h.CreateChirp)))
	mux.HandleFunc("GET /api/chirps", h.GetChirps)
	mux.HandleFunc("GET /api/chirps/{id}", h.GetChirpByID)
	mux.HandleFunc("PUT /api/chirps", h.UpdateChirpByID)
	mux.HandleFunc("DELETE /api/chirps", h.DeleteChirpByID)
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

func (h *ChirpHandler) CreateChirp(w http.ResponseWriter, r *http.Request) {
	const maxChirpLength = 140

	var chirpRequestData CreateChirpRequest

	if err := decodeJSON(w, r, &chirpRequestData); err != nil {
		h.apiConfig.Logger.Error("Invalid request payload", err)
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(chirpRequestData.Body) == 0 {
		writeError(w, http.StatusBadRequest, "Chirp body cannot be empty")
		return
	}

	if len(chirpRequestData.Body) > maxChirpLength {
		writeError(w, http.StatusBadRequest, "Chirp body exceeds maximum length of 140 characters")
		return
	}

	result := containsProfanity(chirpRequestData.Body)

	userID, _ := r.Context().Value(userIDKey).(uuid.UUID)

	createdChirp, err := h.apiConfig.Database.CreateChirp(r.Context(), database.CreateChirpParams{
		ID:     uuid.New(),
		Body:   result.CleanedBody,
		UserID: userID,
	})

	if err != nil {
		h.apiConfig.Logger.Error("Failed to create chirp", err)
		writeError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}

	createdChirpDTO := toChirpResponse(createdChirp)

	writeJSON(w, http.StatusCreated, createdChirpDTO)
}

func (h *ChirpHandler) GetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := h.apiConfig.Database.GetAllChirps(r.Context())
	if err != nil {
		h.apiConfig.Logger.Error("Failed to get chirps", err)
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
	if chirpID == "" {
		writeError(w, http.StatusBadRequest, "Missing chirp ID")
		return
	}

	chirpUUID, err := uuid.Parse(chirpID)
	if err != nil {
		h.apiConfig.Logger.Error("Invalid chirp ID", err)
		writeError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	chirp, err := h.apiConfig.Database.GetChirpByID(r.Context(), chirpUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Chirp not found")
			return
		}

		h.apiConfig.Logger.Error("Failed to get chirp by ID", err)
		writeError(w, http.StatusInternalServerError, "Failed to get chirp by ID")
		return
	}

	writeJSON(w, http.StatusOK, toChirpResponse(chirp))
}

func (h *ChirpHandler) UpdateChirpByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to update a chirp by ID
}

func (h *ChirpHandler) DeleteChirpByID(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to delete a chirp by ID
}
