package models

import (
	"time"
)

// StaffSession represents an active staff authentication session in PostgreSQL.
type StaffSession struct {
	ID        string    `json:"id"`
	StaffID   string    `json:"staff_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
