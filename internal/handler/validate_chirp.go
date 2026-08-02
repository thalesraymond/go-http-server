package handler

import (
	"strings"
)

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
