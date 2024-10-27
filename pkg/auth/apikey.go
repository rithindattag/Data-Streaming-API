package auth

import (
	"crypto/subtle"
	"net/http"
)

const (
	ApiKey = "3645a8d2d3a7f6d1ebd053bffe4fb2eb"
)

var validAPIKeys = map[string]bool{
	ApiKey: true,
}

func APIKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if !ValidateAPIKey(key) {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func ValidateAPIKey(key string) bool {
	return subtle.ConstantTimeCompare([]byte(key), []byte(ApiKey)) == 1
}
