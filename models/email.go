package models

import "time"

// Email represents an email message stored in the system
type Email struct {
	ID             int       `json:"id"`
	Subject        string    `json:"subject"`
	Sender         string    `json:"sender"`
	Date           time.Time `json:"date"`
	Body           string    `json:"body"`
	HTMLBody       string    `json:"html_body"`
	HasAttachments bool      `json:"has_attachments"`
}

// EmailList represents a list of emails for a specific recipient
type EmailList struct {
	Emails []Email `json:"emails"`
}

// EmailResponse represents the paginated response for email retrieval
type EmailResponse struct {
	Emails  []Email `json:"emails"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	Limit   int     `json:"limit"`
	HasMore bool    `json:"has_more"`
}

// WebhookRequest represents the incoming webhook payload from Cloudflare Worker
type WebhookRequest struct {
	To             string            `json:"to"`
	From           string            `json:"from"`
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	HTMLBody       string            `json:"html_body,omitempty"`
	HasAttachments bool              `json:"has_attachments"`
	Headers        map[string]string `json:"headers,omitempty"`
}
