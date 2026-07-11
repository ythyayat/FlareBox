package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"simple-email-server/handlers"
	"simple-email-server/middleware"
	"simple-email-server/storage"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Get configuration from environment
	port := getEnv("PORT", "8080")
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY environment variable is required")
	}

	cleanupIntervalMin := getEnvAsInt("CLEANUP_INTERVAL_MINUTES", 30)
	cleanupInactiveHours := getEnvAsInt("CLEANUP_INACTIVE_HOURS", 6)

	// Create router
	r := mux.NewRouter()

	// Routes
	r.HandleFunc("/api/webhook", middleware.APIKeyAuth(handlers.WebhookHandler)).Methods("POST")
	r.HandleFunc("/api/email/{domain}/{username}/", handlers.GetEmailsHandler).Methods("GET")
	
	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Start cleanup goroutine
	stopCleanup := make(chan bool)
	go runCleanupJob(time.Duration(cleanupIntervalMin)*time.Minute, 
		time.Duration(cleanupInactiveHours)*time.Hour, stopCleanup)

	// Setup graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint
		log.Println("Shutting down gracefully...")
		stopCleanup <- true
		os.Exit(0)
	}()

	// Start server
	addr := ":" + port
	log.Printf("Server starting on port %s", port)
	log.Printf("Cleanup job will run every %d minutes", cleanupIntervalMin)
	log.Printf("Files inactive for %d hours will be deleted", cleanupInactiveHours)
	
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

// runCleanupJob periodically cleans up inactive email files
func runCleanupJob(interval, inactivityDuration time.Duration, stop chan bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run cleanup immediately on start
	log.Println("Running initial cleanup check...")
	if err := storage.CleanupInactiveFiles(inactivityDuration); err != nil {
		log.Printf("Cleanup error: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			log.Println("Running scheduled cleanup check...")
			if err := storage.CleanupInactiveFiles(inactivityDuration); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
		case <-stop:
			log.Println("Cleanup job stopped")
			return
		}
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as integer with a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
