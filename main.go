package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flarebox/handlers"
	"flarebox/middleware"
	"flarebox/storage"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./settings.db"
	}

	// Initialize database
	if err := storage.InitDB(dbPath); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer storage.CloseDB()

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "2525"
	}

	// Initialize templates
	if err := handlers.InitTemplates(); err != nil {
		log.Fatal("Failed to load templates:", err)
	}

	// Setup router
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// UI Routes (public)
	r.HandleFunc("/login", handlers.ServeLogin).Methods("GET")
	r.HandleFunc("/login", handlers.HandleLoginForm).Methods("POST")
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}).Methods("GET")

	// UI Routes (authenticated with cookie)
	uiAuth := r.NewRoute().Subrouter()
	uiAuth.Use(middleware.SessionAuthCookie)
	uiAuth.HandleFunc("/dashboard", handlers.ServeDashboard).Methods("GET")
	uiAuth.HandleFunc("/settings", handlers.ServeSettings).Methods("GET")
	uiAuth.HandleFunc("/logout", handlers.HandleLogoutUI).Methods("POST")
	uiAuth.HandleFunc("/partials/api-keys", handlers.ServeAPIKeysPartial).Methods("GET")
	uiAuth.HandleFunc("/partials/domains", handlers.ServeDomainsPartial).Methods("GET")
	uiAuth.HandleFunc("/partials/settings", handlers.ServeSettingsPartial).Methods("GET")
	uiAuth.HandleFunc("/partials/generate-email", handlers.ServeGeneratorPartial).Methods("GET")

	// Dashboard inbox routes
	uiAuth.HandleFunc("/dashboard/addresses", handlers.DashboardAddressesHandler).Methods("GET")
	uiAuth.HandleFunc("/dashboard/emails/{domain}/{username}", handlers.DashboardEmailsHandler).Methods("GET")
	uiAuth.HandleFunc("/dashboard/email/{domain}/{username}/{id}", handlers.DashboardEmailBodyHandler).Methods("GET")
	uiAuth.HandleFunc("/add-domain", handlers.HandleAddDomain).Methods("POST")
	uiAuth.HandleFunc("/delete-domain/{domain}", handlers.HandleDeleteDomain).Methods("DELETE")
	uiAuth.HandleFunc("/update-settings", handlers.HandleUpdateSettingsUI).Methods("POST")
	uiAuth.HandleFunc("/regenerate-key", handlers.HandleRegenerateKeyUI).Methods("POST")
	uiAuth.HandleFunc("/change-password", handlers.HandleChangePasswordUI).Methods("POST")
	uiAuth.HandleFunc("/change-username", handlers.HandleChangeUsernameUI).Methods("POST")

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Webhook endpoint (protected by webhook API key)
	api.HandleFunc("/webhook", middleware.WebhookAuth(handlers.WebhookHandler)).Methods("POST")

	// Email retrieval endpoint (protected by client API key)
	api.HandleFunc("/email/{domain}/{username}/", middleware.ClientAuth(handlers.GetEmailsHandler)).Methods("GET")

	// Random domains and email endpoints (protected by client API key)
	api.HandleFunc("/random-domains/", middleware.ClientAuth(handlers.GetRandomDomainsHandler)).Methods("GET")
	api.HandleFunc("/random-email", middleware.ClientAuth(handlers.GetRandomEmailHandler)).Methods("GET")

	// Start cleanup goroutine with database-based settings
	stopCleanup := make(chan bool)
	go runCleanupJob(stopCleanup)

	// Setup graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint
		log.Println("Shutting down gracefully...")
		stopCleanup <- true
		storage.CloseDB()
		os.Exit(0)
	}()

	// Start server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("🚀 FlareBox")
	log.Printf("📍 Server running on http://localhost:%s", port)
	log.Println("🔑 Default login: admin / 123456 (change after first login)")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// runCleanupJob periodically cleans up inactive email files based on database settings
func runCleanupJob(stop chan bool) {
	// Get cleanup settings from database
	db := storage.GetDB()
	var intervalMinutes, inactiveHours int
	err := db.QueryRow("SELECT cleanup_interval_minutes, cleanup_inactive_hours FROM settings WHERE id = 1").
		Scan(&intervalMinutes, &inactiveHours)

	if err != nil {
		log.Printf("Failed to get cleanup settings, using defaults: %v", err)
		intervalMinutes = 30
		inactiveHours = 6
	}

	interval := time.Duration(intervalMinutes) * time.Minute
	inactivityDuration := time.Duration(inactiveHours) * time.Hour

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run cleanup immediately on start (silent)
	storage.CleanupInactiveFiles(inactivityDuration)

	for {
		select {
		case <-ticker.C:
			// Refresh settings from database each time
			db := storage.GetDB()
			db.QueryRow("SELECT cleanup_interval_minutes, cleanup_inactive_hours FROM settings WHERE id = 1").
				Scan(&intervalMinutes, &inactiveHours)
			inactivityDuration = time.Duration(inactiveHours) * time.Hour

			// Run cleanup silently (only log errors)
			if err := storage.CleanupInactiveFiles(inactivityDuration); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
		case <-stop:
			log.Println("Cleanup job stopped")
			return
		}
	}
}
