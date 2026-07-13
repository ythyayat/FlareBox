package middleware

import (
	"database/sql"
	"net/http"

	"flarebox/storage"
)

// WebhookAuth validates the webhook API key (for Cloudflare Worker)
func WebhookAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedKey := r.Header.Get("X-API-Key")
		if providedKey == "" {
			providedKey = r.URL.Query().Get("api_key")
		}

		if providedKey == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		// Get webhook API key from database
		db := storage.GetDB()
		var webhookKey string
		err := db.QueryRow("SELECT webhook_api_key FROM settings WHERE id = 1").Scan(&webhookKey)
		if err == sql.ErrNoRows {
			http.Error(w, "Server configuration error", http.StatusInternalServerError)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if providedKey != webhookKey {
			http.Error(w, "Invalid webhook API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// ClientAuth validates the client API key (for data access endpoints)
func ClientAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedKey := r.Header.Get("X-API-Key")
		if providedKey == "" {
			providedKey = r.URL.Query().Get("api_key")
		}

		if providedKey == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		// Get client API key from database
		db := storage.GetDB()
		var clientKey string
		err := db.QueryRow("SELECT client_api_key FROM settings WHERE id = 1").Scan(&clientKey)
		if err == sql.ErrNoRows {
			http.Error(w, "Server configuration error", http.StatusInternalServerError)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if providedKey != clientKey {
			http.Error(w, "Invalid client API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
