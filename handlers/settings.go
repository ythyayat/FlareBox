package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"flarebox/storage"
)

// RegenerateAPIKeyHandler regenerates an API key (used by UI)
func RegenerateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	keyType := r.URL.Query().Get("type") // "webhook" or "client"
	if keyType != "webhook" && keyType != "client" {
		http.Error(w, "Invalid key type. Must be 'webhook' or 'client'", http.StatusBadRequest)
		return
	}

	// Generate new key
	newKey := generateAPIKey(keyType[:2])

	db := storage.GetDB()
	column := keyType + "_api_key"

	_, err := db.Exec("UPDATE settings SET "+column+" = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1", newKey)
	if err != nil {
		http.Error(w, "Failed to regenerate API key", http.StatusInternalServerError)
		return
	}

	// Return the full API keys partial HTML
	// Fetch fresh keys from database
	var webhookKey, clientKey string
	err = db.QueryRow("SELECT webhook_api_key, client_api_key FROM settings WHERE id = 1").
		Scan(&webhookKey, &clientKey)
	if err != nil {
		http.Error(w, "Failed to load API keys", http.StatusInternalServerError)
		return
	}

	// Render the API keys partial template with fresh data
	data := map[string]interface{}{
		"WebhookKey": webhookKey,
		"ClientKey":  clientKey,
	}
	templates.ExecuteTemplate(w, "api-keys.html", data)
}

// generateAPIKey generates a secure random API key with prefix
func generateAPIKey(prefix string) string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes)
}
