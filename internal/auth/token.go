package auth

import (
	"errors"
	"net/http"
	"strings"
)

var ErrMissingToken = errors.New("token not present in Authorization header")

func GetBearerToken(headers http.Header) (string, error) {
	return getToken(headers, "Bearer ")
}

func GetAPIKey(headers http.Header) (string, error) {
	return getToken(headers, "ApiKey ")
}

func getToken(headers http.Header, prefix string) (string, error) {
	token, found := strings.CutPrefix(headers.Get("Authorization"), prefix)
	if !found || token == "" {
		return "", ErrMissingToken
	}

	return token, nil
}
