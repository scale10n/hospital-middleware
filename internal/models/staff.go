package models

import "time"

// Staff represents the staff entity in the database.
type Staff struct {
	ID           string    `json:"id"`
	HospitalID   string    `json:"hospital_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
