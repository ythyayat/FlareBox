package middleware

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"flarebox/storage"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key-change-this") // TODO: Load from config or generate on startup

// Claims represents JWT claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// SessionAuth validates JWT token for dashboard access
func SessionAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Check if session exists and is valid
		db := storage.GetDB()
		var expiresAt time.Time
		err = db.QueryRow(`
			SELECT expires_at FROM sessions 
			WHERE token = ? AND user_id = ?
		`, tokenString, claims.UserID).Scan(&expiresAt)

		if err == sql.ErrNoRows {
			http.Error(w, "Session not found", http.StatusUnauthorized)
			return
		}

		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if time.Now().After(expiresAt) {
			http.Error(w, "Session expired", http.StatusUnauthorized)
			return
		}

		// Store user info in request context for handlers to use
		r.Header.Set("X-User-ID", string(rune(claims.UserID)))
		r.Header.Set("X-Username", claims.Username)

		next(w, r)
	}
}

// GenerateToken generates a new JWT token
func GenerateToken(userID int, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// SessionAuthCookie validates JWT token from cookie (for web UI)
func SessionAuthCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from cookie
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		tokenString := cookie.Value

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Check if session exists and is valid
		db := storage.GetDB()
		var expiresAt time.Time
		err = db.QueryRow(`
			SELECT expires_at FROM sessions 
			WHERE token = ? AND user_id = ?
		`, tokenString, claims.UserID).Scan(&expiresAt)

		if err == sql.ErrNoRows {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if time.Now().After(expiresAt) {
			db.Exec("DELETE FROM sessions WHERE token = ?", tokenString)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Get must_change_password flag
		var mustChangePassword bool
		db.QueryRow("SELECT must_change_password FROM users WHERE id = ?", claims.UserID).Scan(&mustChangePassword)

		// Store user info in request headers for handlers to use
		r.Header.Set("X-User-ID", string(rune(claims.UserID)))
		r.Header.Set("X-Username", claims.Username)
		if mustChangePassword {
			r.Header.Set("X-Must-Change-Password", "true")
		}

		next.ServeHTTP(w, r)
	})
}
