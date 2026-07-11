package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"simple-email-server/models"
	"simple-email-server/storage"
)

// WebhookHandler handles incoming webhook requests from Cloudflare Email Worker
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.To == "" || req.From == "" {
		http.Error(w, "Missing required fields: 'to' and 'from' are required", http.StatusBadRequest)
		return
	}

	// Create email object
	email := models.Email{
		Subject:        req.Subject,
		Sender:         req.From,
		Date:           time.Now(),
		Body:           req.Body,
		HTMLBody:       req.HTMLBody,
		HasAttachments: req.HasAttachments,
	}

	// Save email to storage
	if err := storage.SaveEmail(req.To, email); err != nil {
		http.Error(w, "Failed to save email: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Email received and stored",
		"email_id": email.ID,
	})
}
