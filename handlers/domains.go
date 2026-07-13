package handlers

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"flarebox/models"
	"flarebox/storage"
)

const dataDir = "./data"

// GetRandomDomainsHandler returns random active domains
func GetRandomDomainsHandler(w http.ResponseWriter, r *http.Request) {
	// Get limit from query parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	db := storage.GetDB()

	// Get active domains
	rows, err := db.Query("SELECT domain FROM domains WHERE is_active = 1 ORDER BY RANDOM() LIMIT ?", limit)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			continue
		}
		domains = append(domains, domain)
	}

	if len(domains) == 0 {
		domains = []string{} // Return empty array instead of null
	}

	response := models.DomainListResponse{
		Domains: domains,
		Total:   len(domains),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRandomEmailHandler generates a random unique email address
func GetRandomEmailHandler(w http.ResponseWriter, r *http.Request) {
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

	response := models.RandomEmailResponse{
		Email:    username + "@" + domain,
		Username: username,
		Domain:   domain,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Word lists for generating human-like usernames (Western + Indonesian names)
var firstNames = []string{
	"john", "sarah", "michael", "emily", "david", "emma",
	"james", "olivia", "robert", "sophia", "william", "ava",
	"thomas", "isabella", "daniel", "mia", "matthew", "charlotte",
	"joseph", "amelia", "charles", "harper", "chris", "lily",
	"alex", "grace", "ryan", "zoey", "kevin", "hannah",
	"budi", "siti", "andi", "dewi", "agus", "rani",
	"rudi", "maya", "hendra", "fitri", "dimas", "lina",
	"arif", "sri", "fajar", "ayu", "rizki", "indah",
	"bambang", "putri", "adi", "bayu", "dian", "wulan", "eka",
}

var lastNames = []string{
	"smith", "johnson", "williams", "brown", "jones", "garcia",
	"miller", "davis", "martinez", "anderson", "taylor", "thomas",
	"moore", "martin", "jackson", "thompson", "white", "lopez",
	"lee", "gonzalez", "harris", "clark", "lewis", "robinson",
	"walker", "hall", "allen", "young", "king", "wright",
	"santoso", "wijaya", "kusuma", "pratama", "nugroho",
	"saputra", "putra", "wibowo", "setiawan", "gunawan",
	"permana", "sutanto", "hartono", "susanto", "budiman",
}

var adjectives = []string{
	"happy", "cool", "smart", "bright", "quick", "lucky",
	"brave", "kind", "swift", "bold", "wise", "calm",
	"wild", "free", "true", "blue", "red", "silver",
	"golden", "royal", "super", "mega", "ultra", "prime",
}

var nouns = []string{
	"cat", "bear", "star", "wolf", "eagle", "fox",
	"lion", "tiger", "panda", "hawk", "dragon", "phoenix",
	"knight", "warrior", "ninja", "wizard", "hunter", "ranger",
	"storm", "thunder", "shadow", "flame", "frost", "wind",
}

// generateRandomUsername generates a human-like random username
func generateRandomUsername(length int) string {
	// Get random pattern (0-6)
	patternBytes := make([]byte, 1)
	rand.Read(patternBytes)
	pattern := int(patternBytes[0]) % 7

	var username string

	switch pattern {
	case 0:
		// firstname.lastname
		username = randomFromSlice(firstNames) + "." + randomFromSlice(lastNames)
	case 1:
		// firstname_lastname
		username = randomFromSlice(firstNames) + "_" + randomFromSlice(lastNames)
	case 2:
		// firstnamelastname
		username = randomFromSlice(firstNames) + randomFromSlice(lastNames)
	case 3:
		// firstname + number (2-4 digits)
		username = randomFromSlice(firstNames) + randomDigits(2, 4)
	case 4:
		// initial + lastname (e.g., jsmith)
		firstName := randomFromSlice(firstNames)
		username = string(firstName[0]) + randomFromSlice(lastNames)
	case 5:
		// adjective + noun
		username = randomFromSlice(adjectives) + randomFromSlice(nouns)
	case 6:
		// adjective + noun + number
		username = randomFromSlice(adjectives) + randomFromSlice(nouns) + randomDigits(1, 3)
	}

	return username
}

// randomFromSlice returns a random element from a string slice
func randomFromSlice(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	bytes := make([]byte, 2)
	rand.Read(bytes)
	index := int(bytes[0])<<8 | int(bytes[1])
	return slice[index%len(slice)]
}

// randomDigits generates a random number with specified min and max digits
func randomDigits(minDigits, maxDigits int) string {
	bytes := make([]byte, 1)
	rand.Read(bytes)

	// Determine number of digits
	digitCount := minDigits
	if maxDigits > minDigits {
		digitCount = minDigits + (int(bytes[0]) % (maxDigits - minDigits + 1))
	}

	// Generate the number
	numBytes := make([]byte, 2)
	rand.Read(numBytes)
	num := int(numBytes[0])<<8 | int(numBytes[1])

	// Calculate max value for digit count (e.g., 99 for 2 digits, 999 for 3 digits)
	maxVal := 1
	for i := 0; i < digitCount; i++ {
		maxVal *= 10
	}
	maxVal--

	// Ensure minimum value (e.g., 10 for 2 digits, 100 for 3 digits)
	minVal := 1
	if digitCount > 1 {
		for i := 0; i < digitCount-1; i++ {
			minVal *= 10
		}
	}

	result := minVal + (num % (maxVal - minVal + 1))
	return strconv.Itoa(result)
}

// isEmailExists checks if an email address already exists in storage
func isEmailExists(domain, username string) bool {
	filePath := filepath.Join(dataDir, domain, username+".json")
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// generateUniqueUsername generates a unique username for the given domain
// It tries up to maxAttempts times, then falls back to adding a timestamp
func generateUniqueUsername(domain string) string {
	maxAttempts := 10

	for i := 0; i < maxAttempts; i++ {
		username := generateRandomUsername(0) // length parameter is no longer used
		if !isEmailExists(domain, username) {
			return username
		}
	}

	// Fallback: append timestamp to guarantee uniqueness
	username := generateRandomUsername(0)
	timestamp := strconv.FormatInt(time.Now().Unix()%10000, 10)
	return username + timestamp
}
