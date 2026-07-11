package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"simple-email-server/models"
)

const (
	dataDir = "data"
)

var (
	mu       sync.RWMutex
	idCounter = make(map[string]int) // Track IDs per email address
)

// SaveEmail saves an email to the JSON storage
func SaveEmail(to string, email models.Email) error {
	mu.Lock()
	defer mu.Unlock()

	// Parse email address to extract domain and username
	domain, username, err := parseEmailAddress(to)
	if err != nil {
		return err
	}

	// Create directory structure
	dirPath := filepath.Join(dataDir, domain)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// File path for this email address
	filePath := filepath.Join(dirPath, username+".json")

	// Load existing emails
	emailList := models.EmailList{Emails: []models.Email{}}
	if data, err := os.ReadFile(filePath); err == nil {
		if err := json.Unmarshal(data, &emailList); err != nil {
			return fmt.Errorf("failed to parse existing emails: %w", err)
		}
	}

	// Generate unique ID
	key := fmt.Sprintf("%s/%s", domain, username)
	if idCounter[key] == 0 {
		// Find max ID from existing emails
		for _, e := range emailList.Emails {
			if e.ID > idCounter[key] {
				idCounter[key] = e.ID
			}
		}
	}
	idCounter[key]++
	email.ID = idCounter[key]

	// Append new email
	emailList.Emails = append(emailList.Emails, email)

	// Save to file
	data, err := json.MarshalIndent(emailList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal emails: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetEmails retrieves paginated emails for a specific email address
func GetEmails(domain, username string, page, limit int) (*models.EmailResponse, error) {
	mu.RLock()
	defer mu.RUnlock()

	filePath := filepath.Join(dataDir, domain, username+".json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &models.EmailResponse{
			Emails:  []models.Email{},
			Total:   0,
			Page:    page,
			Limit:   limit,
			HasMore: false,
		}, nil
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var emailList models.EmailList
	if err := json.Unmarshal(data, &emailList); err != nil {
		return nil, fmt.Errorf("failed to parse emails: %w", err)
	}

	total := len(emailList.Emails)

	// Calculate pagination
	start := (page - 1) * limit
	if start >= total {
		return &models.EmailResponse{
			Emails:  []models.Email{},
			Total:   total,
			Page:    page,
			Limit:   limit,
			HasMore: false,
		}, nil
	}

	end := start + limit
	if end > total {
		end = total
	}

	// Reverse order (newest first)
	reversedEmails := make([]models.Email, total)
	for i, email := range emailList.Emails {
		reversedEmails[total-1-i] = email
	}

	paginatedEmails := reversedEmails[start:end]
	hasMore := end < total

	return &models.EmailResponse{
		Emails:  paginatedEmails,
		Total:   total,
		Page:    page,
		Limit:   limit,
		HasMore: hasMore,
	}, nil
}

// CleanupInactiveFiles removes email files that haven't been modified in the specified duration
func CleanupInactiveFiles(inactivityDuration time.Duration) error {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	var cleanupErrors []string

	// Walk through the data directory
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a JSON file
		if filepath.Ext(path) == ".json" {
			// Check modification time
			if now.Sub(info.ModTime()) > inactivityDuration {
				// Delete the file
				if err := os.Remove(path); err != nil {
					cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete %s: %v", path, err))
				} else {
					fmt.Printf("Cleaned up inactive file: %s (last modified: %s)\n", path, info.ModTime().Format(time.RFC3339))
				}
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Clean up empty directories
	filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == dataDir {
			return err
		}

		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			os.Remove(path)
			fmt.Printf("Removed empty directory: %s\n", path)
		}

		return nil
	})

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup completed with errors: %s", strings.Join(cleanupErrors, "; "))
	}

	return nil
}

// parseEmailAddress splits email address into domain and username
func parseEmailAddress(email string) (domain, username string, err error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid email address: %s", email)
	}
	return parts[1], parts[0], nil
}
