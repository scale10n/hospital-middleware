package repository

import (
	"context"
	"database/sql"
	"errors"

	"hospital-middleware/internal/models"
)

// StaffRepository defines the data access contract for staff members.
type StaffRepository interface {
	Create(ctx context.Context, staff *models.Staff) (*models.Staff, error)
	FindByHospitalAndUsername(ctx context.Context, hospitalID string, username string) (*models.Staff, error)
	FindByUsername(ctx context.Context, username string) (*models.Staff, error)
	ExistsByHospitalAndUsername(ctx context.Context, hospitalID string, username string) (bool, error)
}

type sqlStaffRepository struct {
	db *sql.DB
}

// NewStaffRepository constructs a new PostgreSQL staff repository.
func NewStaffRepository(db *sql.DB) StaffRepository {
	return &sqlStaffRepository{db: db}
}

// Create inserts a new staff record into table staff.
func (r *sqlStaffRepository) Create(ctx context.Context, staff *models.Staff) (*models.Staff, error) {
	query := `
		INSERT INTO staff (hospital_id, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, hospital_id, username, password_hash, created_at, updated_at
	`
	var s models.Staff
	err := r.db.QueryRowContext(ctx, query, staff.HospitalID, staff.Username, staff.PasswordHash).
		Scan(&s.ID, &s.HospitalID, &s.Username, &s.PasswordHash, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByHospitalAndUsername retrieves a staff record by hospital ID and username.
func (r *sqlStaffRepository) FindByHospitalAndUsername(ctx context.Context, hospitalID string, username string) (*models.Staff, error) {
	query := `
		SELECT id, hospital_id, username, password_hash, created_at, updated_at
		FROM staff
		WHERE hospital_id = $1 AND username = $2
	`
	var s models.Staff
	err := r.db.QueryRowContext(ctx, query, hospitalID, username).
		Scan(&s.ID, &s.HospitalID, &s.Username, &s.PasswordHash, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// FindByUsername retrieves a staff record by username.
func (r *sqlStaffRepository) FindByUsername(ctx context.Context, username string) (*models.Staff, error) {
	query := `
		SELECT id, hospital_id, username, password_hash, created_at, updated_at
		FROM staff
		WHERE username = $1
	`
	var s models.Staff
	err := r.db.QueryRowContext(ctx, query, username).
		Scan(&s.ID, &s.HospitalID, &s.Username, &s.PasswordHash, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// ExistsByHospitalAndUsername checks if a staff record with given hospital ID and username exists.
func (r *sqlStaffRepository) ExistsByHospitalAndUsername(ctx context.Context, hospitalID string, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM staff WHERE hospital_id = $1 AND username = $2)`
	err := r.db.QueryRowContext(ctx, query, hospitalID, username).Scan(&exists)
	return exists, err
}
