package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type chirpRequest struct {
	Body string `json:"body"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type validResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

var profanityList = []string{
	"kerfuffle",
	"sharbert",
	"fornax",
}

type profanityResult struct {
	HasProfanity bool   `json:"has_profanity"`
	CleanedBody  string `json:"cleaned_body"`
}

func containsProfanity(text string) profanityResult {
	cleaned := replaceProfanity(text)
	return profanityResult{
		HasProfanity: cleaned != text,
		CleanedBody:  cleaned,
	}
}

func replaceProfanity(text string) string {
	result := text
	for _, word := range profanityList {
		lowerWord := strings.ToLower(word)
		lowerText := strings.ToLower(result)
		for {
			idx := strings.Index(lowerText, lowerWord)
			if idx == -1 {
				break
			}
			// Replace the slice from idx to idx+len(word) with "****"
			result = result[:idx] + "****" + result[idx+len(word):]
			// Update lowerText for next iteration (since result changed)
			lowerText = strings.ToLower(result)
		}
	}
	return result
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

	result := containsProfanity(req.Body)

	returnWithJSON(w, http.StatusOK, validResponse{
		CleanedBody: result.CleanedBody,
	})
}
