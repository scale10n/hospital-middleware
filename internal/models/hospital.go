package models

import "time"

// Hospital represents the hospital entity in the database.
type Hospital struct {
	ID        string    `json:"id"`
	HN        string    `json:"hn"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
