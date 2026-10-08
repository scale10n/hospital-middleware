package repository

import (
	"context"
	"database/sql"

	"hospital-middleware/internal/models"
)

// SessionRepository defines the data access contract for staff sessions.
type StaffSessionRepository interface {
	Create(ctx context.Context, session *models.StaffSession) (*models.StaffSession, error)
	FindByToken(ctx context.Context, token string) (*models.StaffSession, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByStaffID(ctx context.Context, staffID string) error
}

type sqlStaffSessionRepository struct {
	db *sql.DB
}

// NewSessionRepository constructs a new PostgreSQL session repository.
func NewStaffSessionRepository(db *sql.DB) StaffSessionRepository {
	return &sqlStaffSessionRepository{db: db}
}

// Create inserts a new session record into table staff_session.
func (r *sqlStaffSessionRepository) Create(ctx context.Context, session *models.StaffSession) (*models.StaffSession, error) {
	query := `
		INSERT INTO staff_session (staff_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, staff_id, token, expires_at, created_at, updated_at
	`
	var s models.StaffSession
	err := r.db.QueryRowContext(ctx, query, session.StaffID, session.Token, session.ExpiresAt).
		Scan(&s.ID, &s.StaffID, &s.Token, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByToken retrieves a session by its token string.
func (r *sqlStaffSessionRepository) FindByToken(ctx context.Context, token string) (*models.StaffSession, error) {
	query := `
		SELECT id, staff_id, token, expires_at, created_at, updated_at
		FROM staff_session
		WHERE token = $1
	`
	var s models.StaffSession
	err := r.db.QueryRowContext(ctx, query, token).
		Scan(&s.ID, &s.StaffID, &s.Token, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// DeleteByToken deletes a session by token string.
func (r *sqlStaffSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM staff_session WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

// DeleteByStaffID deletes all sessions for a specific staff member.
func (r *sqlStaffSessionRepository) DeleteByStaffID(ctx context.Context, staffID string) error {
	query := `DELETE FROM staff_session WHERE staff_id = $1`
	_, err := r.db.ExecContext(ctx, query, staffID)
	return err
}
