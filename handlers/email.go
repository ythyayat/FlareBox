package handlers

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"flarebox/storage"

	"github.com/gorilla/mux"
)

// GetEmailsHandler handles GET requests to retrieve emails for a specific address
// Note: This handler is protected by ClientAuth middleware
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

// DashboardAddressesHandler returns HTML for the email addresses list
func DashboardAddressesHandler(w http.ResponseWriter, r *http.Request) {
	addresses, err := storage.GetAllEmailAddresses()
	if err != nil {
		http.Error(w, "Failed to retrieve email addresses", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if len(addresses) == 0 {
		w.Write([]byte(`<div class="empty-inbox"><p>No emails yet. Generate an email address and send some test emails!</p></div>`))
		return
	}

	// Render email addresses
	for _, addr := range addresses {
		timeAgo := formatTimeAgo(addr.LatestDate)
		safeID := "emails-" + sanitizeForID(addr.Domain) + "-" + sanitizeForID(addr.Username)
		html := `<div class="email-address-item">
			<div class="email-list-container" id="` + safeID + `"></div>
			<div class="email-address-row" 
				hx-get="/dashboard/emails/` + addr.Domain + `/` + addr.Username + `"
				hx-target="#` + safeID + `"
				hx-swap="innerHTML">
				<div class="address-info">
					<span class="email-address">` + addr.Address + `</span>
					<span class="email-meta">
						<span class="email-count">` + strconv.Itoa(addr.Count) + ` email` + plural(addr.Count) + `</span>
						<span class="email-time">` + timeAgo + `</span>
					</span>
				</div>
				<span class="material-icons expand-icon">expand_more</span>
			</div>
		</div>`
		w.Write([]byte(html))
	}
}

// DashboardEmailsHandler returns HTML for the email list of a specific address
func DashboardEmailsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]
	username := vars["username"]

	response, err := storage.GetEmails(domain, username, 1, 100)
	if err != nil {
		http.Error(w, "Failed to retrieve emails", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if len(response.Emails) == 0 {
		w.Write([]byte(`<div class="no-emails"><p>No emails found</p></div>`))
		return
	}

	w.Write([]byte(`<div class="emails-expanded">`))
	for _, email := range response.Emails {
		timeStr := email.Date.Format("Jan 2, 3:04 PM")
		html := `<div class="email-item">
			<div class="email-header" 
				hx-get="/dashboard/email/` + domain + `/` + username + `/` + strconv.Itoa(email.ID) + `"
				hx-target="next .email-body-container"
				hx-swap="innerHTML">
				<div class="email-subject">` + escapeHTML(email.Subject) + `</div>
				<div class="email-meta-row">
					<span class="email-sender">from: ` + escapeHTML(email.Sender) + `</span>
					<span class="email-time">` + timeStr + `</span>
				</div>
			</div>
			<div class="email-body-container"></div>
		</div>`
		w.Write([]byte(html))
	}
	w.Write([]byte(`</div>`))
}

// DashboardEmailBodyHandler returns HTML for a specific email body
func DashboardEmailBodyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]
	username := vars["username"]
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	email, err := storage.GetEmailByID(domain, username, id)
	if err != nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	body := email.Body
	if email.HTMLBody != "" {
		body = email.HTMLBody
	}

	html := `<div class="email-body">
		<div class="email-body-content">` + body + `</div>
	</div>`

	w.Write([]byte(html))
}

// Helper functions

func sanitizeForID(s string) string {
	return strings.ReplaceAll(s, ".", "-")
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return strconv.Itoa(mins) + " min" + plural(mins) + " ago"
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return strconv.Itoa(hours) + " hour" + plural(hours) + " ago"
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		return strconv.Itoa(days) + " day" + plural(days) + " ago"
	} else {
		return t.Format("Jan 2, 2006")
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func escapeHTML(s string) string {
	return html.EscapeString(s)
}
