package middleware

import (
	"net/http"
	"os"
)

// APIKeyAuth is a middleware that checks for a valid API key
func APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("API_KEY")
		if apiKey == "" {
			http.Error(w, "Server configuration error: API_KEY not set", http.StatusInternalServerError)
			return
		}

		// Check API key from header
		providedKey := r.Header.Get("X-API-Key")
		
		// If not in header, check query parameter
		if providedKey == "" {
			providedKey = r.URL.Query().Get("api_key")
		}

		if providedKey == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		if providedKey != apiKey {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
