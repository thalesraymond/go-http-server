package handler

import (
	"errors"
	"strings"
)

const maxChirpLength = 140

var (
	ErrChirpEmpty   = errors.New("chirp body cannot be empty")
	ErrChirpTooLong = errors.New("chirp body exceeds maximum length of 140 characters")
)

var profanityList = []string{
	"kerfuffle",
	"sharbert",
	"fornax",
}

func ValidateChirp(body string) (string, error) {
	if len(body) == 0 {
		return "", ErrChirpEmpty
	}

	if len(body) > maxChirpLength {
		return "", ErrChirpTooLong
	}

	return replaceProfanity(body), nil
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

			result = result[:idx] + "****" + result[idx+len(word):]

			lowerText = strings.ToLower(result)
		}
	}
	return result
}
