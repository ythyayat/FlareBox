package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

// InitDB initializes the SQLite database
func InitDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Initialize default data
	if err := initializeDefaultData(); err != nil {
		return fmt.Errorf("failed to initialize default data: %w", err)
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *sql.DB {
	return db
}

// createTables creates all necessary tables
func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY DEFAULT 1,
		webhook_api_key TEXT NOT NULL,
		client_api_key TEXT NOT NULL,
		cleanup_interval_minutes INTEGER DEFAULT 30,
		cleanup_inactive_hours INTEGER DEFAULT 6,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CHECK (id = 1)
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		must_change_password BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain TEXT UNIQUE NOT NULL,
		is_active BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT UNIQUE NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_domains_active ON domains(is_active);
	`

	_, err := db.Exec(schema)
	return err
}

// initializeDefaultData initializes default settings, user, and domain
func initializeDefaultData() error {
	// Check if settings already exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Generate random API keys
		webhookKey := generateAPIKey("wh")
		clientKey := generateAPIKey("cl")

		// Insert default settings
		_, err = db.Exec(`
			INSERT INTO settings (id, webhook_api_key, client_api_key, cleanup_interval_minutes, cleanup_inactive_hours)
			VALUES (1, ?, ?, 30, 6)
		`, webhookKey, clientKey)
		if err != nil {
			return fmt.Errorf("failed to create default settings: %w", err)
		}

		log.Printf("✅ Default settings created")
		log.Printf("📧 Webhook API Key: %s", webhookKey)
		log.Printf("🔑 Client API Key: %s", clientKey)
	}

	// Check if default user exists
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Hash default password
		hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Insert default user
		_, err = db.Exec(`
			INSERT INTO users (username, password_hash, must_change_password)
			VALUES (?, ?, 1)
		`, "admin", string(hash))
		if err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}

		log.Printf("👤 Default user created - Username: admin, Password: 123456")
		log.Printf("⚠️  Please change the password after first login!")
	}

	// Check if default domain exists
	err = db.QueryRow("SELECT COUNT(*) FROM domains").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Add default domain
		defaultDomain := os.Getenv("DEFAULT_DOMAIN")
		if defaultDomain == "" {
			defaultDomain = "example.com"
		}

		_, err = db.Exec(`
			INSERT INTO domains (domain, is_active)
			VALUES (?, 1)
		`, defaultDomain)
		if err != nil {
			return fmt.Errorf("failed to create default domain: %w", err)
		}

		log.Printf("🌐 Default domain created: %s", defaultDomain)
	}

	return nil
}

// generateAPIKey generates a cryptographically secure API key with a prefix
func generateAPIKey(prefix string) string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes)
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
