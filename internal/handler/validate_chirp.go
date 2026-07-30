package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type chirpRequest struct {
	Body string `json:"body"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type validResponse struct {
	Valid bool `json:"valid"`
}

func returnWithError(w http.ResponseWriter, statusCode int, errorMessage string) {
	returnWithJSON(w, statusCode, errorResponse{
		Error: errorMessage,
	})
}

func returnWithJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func ValidateChirpHandler(w http.ResponseWriter, r *http.Request) {

	const maxChirpLength = 140

	var req chirpRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		returnWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Body) == 0 {
		returnWithError(w, http.StatusBadRequest, "Chirp body cannot be empty")
		return
	}

	if len(req.Body) > maxChirpLength {
		returnWithError(w, http.StatusBadRequest, "Chirp body exceeds maximum length of "+strconv.Itoa(maxChirpLength)+" characters")
		return
	}

	returnWithJSON(w, http.StatusOK, validResponse{
		Valid: true,
	})
}
