package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func ValidateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Body string `json:"body"`
	}

	type ValidResponse struct {
		Valid bool `json:"valid"`
	}

	type ErrorResponse struct {
		Error string `json:"error"`
	}

	const maxChirpLength = 140

	decoder := json.NewDecoder(r.Body)

	var req Request
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	if err := decoder.Decode(&req); err != nil {
		resp := ErrorResponse{
			Error: "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(req.Body) == 0 {
		resp := ErrorResponse{
			Error: "Chirp body cannot be empty",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(req.Body) > maxChirpLength {
		resp := ErrorResponse{
			Error: "Chirp body exceeds maximum length of " + strconv.Itoa(maxChirpLength) + " characters",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.WriteHeader(http.StatusOK)

	resp := ValidResponse{
		Valid: true,
	}

	json.NewEncoder(w).Encode(resp)
}
