package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"simple-email-server/storage"

	"github.com/gorilla/mux"
)

// GetEmailsHandler handles GET requests to retrieve emails for a specific address
func GetEmailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path parameters
	vars := mux.Vars(r)
	domain := vars["domain"]
	username := vars["username"]

	if domain == "" || username == "" {
		http.Error(w, "Missing domain or username in URL", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Limit maximum page size
	if limit > 100 {
		limit = 100
	}

	// Retrieve emails from storage
	response, err := storage.GetEmails(domain, username, page, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve emails: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
