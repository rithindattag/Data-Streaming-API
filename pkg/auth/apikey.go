package auth

import (
	"net/http"
	"os"
	"strings"
)

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

func ValidateAPIKey(apiKey string) bool {
	// Get the API key from environment variable
	validAPIKey := os.Getenv("API_KEY")
	
	// Compare the provided API key with the valid one
	return strings.TrimSpace(apiKey) == strings.TrimSpace(validAPIKey)
}
