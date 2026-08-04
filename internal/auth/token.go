package auth

import (
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	return getToken(headers, "Bearer ")
}

func GetAPIKey(headers http.Header) (string, error) {
	return getToken(headers, "ApiKey ")
}

func getToken(headers http.Header, prefix string) (string, error) {
	token, found := strings.CutPrefix(headers.Get("Authorization"), prefix)
	if !found || token == "" {
		return "", http.ErrNoCookie
	}

	return token, nil
}
