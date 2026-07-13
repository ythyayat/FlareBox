package models

import "time"

// User represents a dashboard user
type User struct {
	ID                 int       `json:"id"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"` // Never send password hash in JSON
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token              string `json:"token"`
	MustChangePassword bool   `json:"must_change_password"`
	Username           string `json:"username"`
}

// ChangePasswordRequest represents password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
