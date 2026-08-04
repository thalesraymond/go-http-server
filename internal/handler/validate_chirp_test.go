package handler

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateChirp(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantCleaned string
		wantErr     error
	}{
		{
			name:        "accepts valid chirp",
			body:        "Hello world",
			wantCleaned: "Hello world",
		},
		{
			name:        "cleans profanity",
			body:        "I heard kerfuffle is bad",
			wantCleaned: "I heard **** is bad",
		},
		{
			name:        "cleans multiple profanity words",
			body:        "kerfuffle sharbert fornax",
			wantCleaned: "**** **** ****",
		},
		{
			name:        "cleans case-insensitive profanity",
			body:        "Kerfuffle and SHARBERT",
			wantCleaned: "**** and ****",
		},
		{
			name:        "accepts exactly 140 characters",
			body:        strings.Repeat("a", 140),
			wantCleaned: strings.Repeat("a", 140),
		},
		{
			name:    "rejects empty body",
			body:    "",
			wantErr: ErrChirpEmpty,
		},
		{
			name:    "rejects body over 140 characters",
			body:    strings.Repeat("a", 141),
			wantErr: ErrChirpTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateChirp(tt.body)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateChirp(%q) error = %v, want %v", tt.body, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateChirp(%q): unexpected error %v", tt.body, err)
			}
			if got != tt.wantCleaned {
				t.Errorf("ValidateChirp(%q) = %q, want %q", tt.body, got, tt.wantCleaned)
			}
		})
	}
}
