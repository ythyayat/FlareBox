package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"flarebox/middleware"
	"flarebox/models"
	"flarebox/storage"

	"golang.org/x/crypto/bcrypt"
)

var templates *template.Template

// InitTemplates initializes all HTML templates
func InitTemplates() error {
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		return err
	}
	_, err = templates.ParseGlob("templates/partials/*.html")
	return err
}

// ServeLogin serves the login page
func ServeLogin(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":   "Login",
		"ShowNav": false,
	}

	// Parse both base and login templates together
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/login.html"))
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleLoginForm processes login form submission
func HandleLoginForm(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	db := storage.GetDB()
	var user models.User
	err := db.QueryRow(`
		SELECT id, username, password_hash, must_change_password 
		FROM users WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.MustChangePassword)

	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid username or password"))
		return
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Username)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Store session in database
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO sessions (user_id, token, expires_at)
		VALUES (?, ?, ?)
	`, user.ID, token, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set JWT in HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Return success - HTMX will handle redirect
	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

// ServeDashboard serves the main dashboard page
func ServeDashboard(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	mustChangePassword := r.Header.Get("X-Must-Change-Password") == "true"

	data := map[string]interface{}{
		"Title":              "Dashboard",
		"ShowNav":            true,
		"Username":           username,
		"MustChangePassword": mustChangePassword,
	}

	// Parse both base and dashboard templates together
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/dashboard.html"))
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleLogoutUI handles logout from UI
func HandleLogoutUI(w http.ResponseWriter, r *http.Request) {
	// Get token from cookie
	cookie, err := r.Cookie("session_token")
	if err == nil {
		// Delete session from database
		db := storage.GetDB()
		db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ServeAPIKeysPartial serves the API keys partial template
func ServeAPIKeysPartial(w http.ResponseWriter, r *http.Request) {
	db := storage.GetDB()
	var webhookKey, clientKey string
	err := db.QueryRow("SELECT webhook_api_key, client_api_key FROM settings WHERE id = 1").
		Scan(&webhookKey, &clientKey)

	if err != nil {
		http.Error(w, "Failed to load API keys", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"WebhookKey": webhookKey,
		"ClientKey":  clientKey,
	}
	templates.ExecuteTemplate(w, "api-keys.html", data)
}

// HandleRegenerateKeyUI handles API key regeneration from UI
func HandleRegenerateKeyUI(w http.ResponseWriter, r *http.Request) {
	keyType := r.URL.Query().Get("type")
	if keyType != "webhook" && keyType != "client" {
		http.Error(w, "Invalid key type", http.StatusBadRequest)
		return
	}

	// Use the existing regenerate handler logic
	RegenerateAPIKeyHandler(w, r)

	// Return just the new key to update in place
	// The handler already returns JSON with new_key
}

// ServeDomainsPartial serves the domains partial template
func ServeDomainsPartial(w http.ResponseWriter, r *http.Request) {
	db := storage.GetDB()
	rows, err := db.Query("SELECT domain FROM domains WHERE is_active = 1")
	if err != nil {
		http.Error(w, "Failed to load domains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err == nil {
			domains = append(domains, domain)
		}
	}

	data := map[string]interface{}{
		"Domains": domains,
	}
	templates.ExecuteTemplate(w, "domains.html", data)
}

// HandleAddDomain adds a new domain
func HandleAddDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.FormValue("domain")
	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	db := storage.GetDB()
	_, err := db.Exec("INSERT INTO domains (domain, is_active) VALUES (?, 1)", domain)
	if err != nil {
		http.Error(w, "Failed to add domain", http.StatusInternalServerError)
		return
	}

	// Return the new domain item HTML
	w.Write([]byte(`<li class="domain-item">
		<span class="domain-name">` + domain + `</span>
		<button hx-delete="/delete-domain/` + domain + `" hx-target="closest .domain-item" hx-swap="outerHTML" hx-confirm="Delete domain '` + domain + `'?" class="btn btn-sm btn-danger">
			Delete
		</button>
	</li>`))
}

// HandleDeleteDomain deletes a domain
func HandleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	// Extract domain from URL path
	domain := r.URL.Path[len("/delete-domain/"):]
	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	db := storage.GetDB()
	_, err := db.Exec("DELETE FROM domains WHERE domain = ?", domain)
	if err != nil {
		http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
		return
	}

	// Return empty (HTMX will remove the element)
	w.WriteHeader(http.StatusOK)
}

// ServeSettingsPartial serves the settings partial template
func ServeSettingsPartial(w http.ResponseWriter, r *http.Request) {
	db := storage.GetDB()
	var intervalMinutes, inactiveHours int
	err := db.QueryRow("SELECT cleanup_interval_minutes, cleanup_inactive_hours FROM settings WHERE id = 1").
		Scan(&intervalMinutes, &inactiveHours)

	if err != nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"CleanupIntervalMinutes": intervalMinutes,
		"CleanupInactiveHours":   inactiveHours,
	}
	templates.ExecuteTemplate(w, "settings.html", data)
}

// HandleUpdateSettingsUI handles settings update from UI
func HandleUpdateSettingsUI(w http.ResponseWriter, r *http.Request) {
	intervalStr := r.FormValue("cleanup_interval_minutes")
	inactiveStr := r.FormValue("cleanup_inactive_hours")

	db := storage.GetDB()
	_, err := db.Exec(`
		UPDATE settings 
		SET cleanup_interval_minutes = ?, cleanup_inactive_hours = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = 1
	`, intervalStr, inactiveStr)

	if err != nil {
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ServeGeneratorPartial generates and serves a random unique email
func ServeGeneratorPartial(w http.ResponseWriter, r *http.Request) {
	db := storage.GetDB()

	// Get a random active domain
	var domain string
	err := db.QueryRow("SELECT domain FROM domains WHERE is_active = 1 ORDER BY RANDOM() LIMIT 1").Scan(&domain)
	if err != nil {
		http.Error(w, "No active domains available", http.StatusNotFound)
		return
	}

	// Generate unique username for this domain
	username := generateUniqueUsername(domain)
	email := username + "@" + domain

	data := map[string]interface{}{
		"Email":    email,
		"Username": username,
		"Domain":   domain,
	}
	templates.ExecuteTemplate(w, "generator.html", data)
}

// HandleChangePasswordUI handles password change from UI
func HandleChangePasswordUI(w http.ResponseWriter, r *http.Request) {
	oldPassword := r.FormValue("old_password")
	newPassword := r.FormValue("new_password")
	username := r.Header.Get("X-Username")

	if username == "" {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	db := storage.GetDB()

	// Get current password hash
	var passwordHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&passwordHash)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(oldPassword)); err != nil {
		http.Error(w, "Invalid old password", http.StatusUnauthorized)
		return
	}

	// Validate new password
	if len(newPassword) < 8 {
		http.Error(w, "New password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update password and clear must_change_password flag
	_, err = db.Exec(`
		UPDATE users 
		SET password_hash = ?, must_change_password = 0, updated_at = CURRENT_TIMESTAMP
		WHERE username = ?
	`, string(newHash), username)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleChangeUsernameUI handles username change from UI
func HandleChangeUsernameUI(w http.ResponseWriter, r *http.Request) {
	newUsername := r.FormValue("new_username")
	currentPassword := r.FormValue("current_password")
	currentUsername := r.Header.Get("X-Username")

	if currentUsername == "" {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Validate new username
	if newUsername == "" || len(newUsername) < 3 {
		http.Error(w, "Username must be at least 3 characters", http.StatusBadRequest)
		return
	}

	if newUsername == currentUsername {
		http.Error(w, "New username must be different from current username", http.StatusBadRequest)
		return
	}

	db := storage.GetDB()

	// Get current password hash and user ID
	var passwordHash string
	var userID int
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", currentUsername).
		Scan(&userID, &passwordHash)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// Check if new username already exists
	var existingID int
	err = db.QueryRow("SELECT id FROM users WHERE username = ?", newUsername).Scan(&existingID)
	if err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Update username
	_, err = db.Exec(`
		UPDATE users 
		SET username = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newUsername, userID)
	if err != nil {
		http.Error(w, "Failed to update username", http.StatusInternalServerError)
		return
	}

	// Invalidate all sessions for this user (force re-login with new username)
	_, err = db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		http.Error(w, "Failed to clear sessions", http.StatusInternalServerError)
		return
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	// Redirect to login page with HTMX
	w.Header().Set("HX-Redirect", "/login")
	w.WriteHeader(http.StatusOK)
}

// ServeSettings serves the settings page
func ServeSettings(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")

	data := map[string]interface{}{
		"Title":    "Settings",
		"ShowNav":  true,
		"Username": username,
	}

	// Parse both base and settings templates together
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/settings.html"))
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
