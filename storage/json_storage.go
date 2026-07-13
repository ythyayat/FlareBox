package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"flarebox/models"
)

const (
	dataDir = "data"
)

var (
	mu        sync.RWMutex
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

// EmailAddressSummary represents an email address with metadata
type EmailAddressSummary struct {
	Address    string    `json:"address"`
	Domain     string    `json:"domain"`
	Username   string    `json:"username"`
	Count      int       `json:"count"`
	LatestDate time.Time `json:"latest_date"`
}

// GetAllEmailAddresses returns all email addresses with message counts and latest dates
func GetAllEmailAddresses() ([]EmailAddressSummary, error) {
	mu.RLock()
	defer mu.RUnlock()

	var summaries []EmailAddressSummary

	// Check if data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return summaries, nil
	}

	// Walk through all domain directories
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// Read the email file
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var emailList models.EmailList
		if err := json.Unmarshal(data, &emailList); err != nil {
			return nil // Skip files we can't parse
		}

		// Skip empty email lists
		if len(emailList.Emails) == 0 {
			return nil
		}

		// Extract domain and username from path
		rel, _ := filepath.Rel(dataDir, path)
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) != 2 {
			return nil
		}

		domain := parts[0]
		username := strings.TrimSuffix(parts[1], ".json")

		// Find latest email date
		latestDate := emailList.Emails[0].Date
		for _, email := range emailList.Emails {
			if email.Date.After(latestDate) {
				latestDate = email.Date
			}
		}

		summaries = append(summaries, EmailAddressSummary{
			Address:    fmt.Sprintf("%s@%s", username, domain),
			Domain:     domain,
			Username:   username,
			Count:      len(emailList.Emails),
			LatestDate: latestDate,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan email files: %w", err)
	}

	// Sort by latest date (newest first)
	for i := 0; i < len(summaries)-1; i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[j].LatestDate.After(summaries[i].LatestDate) {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}

	return summaries, nil
}

// GetEmailByID retrieves a specific email by ID for a given address
func GetEmailByID(domain, username string, id int) (*models.Email, error) {
	mu.RLock()
	defer mu.RUnlock()

	filePath := filepath.Join(dataDir, domain, username+".json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("email address not found")
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

	// Find email by ID
	for _, email := range emailList.Emails {
		if email.ID == id {
			return &email, nil
		}
	}

	return nil, fmt.Errorf("email not found")
}

// parseEmailAddress splits email address into domain and username
func parseEmailAddress(email string) (domain, username string, err error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid email address: %s", email)
	}
	return parts[1], parts[0], nil
}
