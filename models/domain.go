package models

import "time"

// Domain represents an email domain for random email generation
type Domain struct {
	ID        int       `json:"id"`
	Domain    string    `json:"domain"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// DomainListResponse represents response for domain listing
type DomainListResponse struct {
	Domains []string `json:"domains"`
	Total   int      `json:"total"`
}

// RandomEmailResponse represents a generated random email
type RandomEmailResponse struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Domain   string `json:"domain"`
}

// DomainRequest represents request to add/update domains
type DomainRequest struct {
	Domains []string `json:"domains"`
}
