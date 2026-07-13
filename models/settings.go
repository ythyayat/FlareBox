package models

import "time"

// Settings represents application settings
type Settings struct {
	ID                     int       `json:"id"`
	WebhookAPIKey          string    `json:"webhook_api_key"`
	ClientAPIKey           string    `json:"client_api_key"`
	CleanupIntervalMinutes int       `json:"cleanup_interval_minutes"`
	CleanupInactiveHours   int       `json:"cleanup_inactive_hours"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// SettingsUpdateRequest represents settings update request
type SettingsUpdateRequest struct {
	WebhookAPIKey          *string `json:"webhook_api_key,omitempty"`
	ClientAPIKey           *string `json:"client_api_key,omitempty"`
	CleanupIntervalMinutes *int    `json:"cleanup_interval_minutes,omitempty"`
	CleanupInactiveHours   *int    `json:"cleanup_inactive_hours,omitempty"`
	Username               *string `json:"username,omitempty"`
	Password               *string `json:"password,omitempty"`
}

// SettingsResponse represents settings for dashboard display
type SettingsResponse struct {
	WebhookAPIKey          string   `json:"webhook_api_key"`
	ClientAPIKey           string   `json:"client_api_key"`
	CleanupIntervalMinutes int      `json:"cleanup_interval_minutes"`
	CleanupInactiveHours   int      `json:"cleanup_inactive_hours"`
	Domains                []string `json:"domains"`
	Username               string   `json:"username"`
}
