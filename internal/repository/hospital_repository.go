package repository

import (
	"context"
	"database/sql"
	"errors"

	"hospital-middleware/internal/models"
)

// HospitalRepository defines the data access contract for hospitals.
type HospitalRepository interface {
	ExistsByHN(ctx context.Context, hn string) (bool, error)
	FindByHN(ctx context.Context, hn string) (*models.Hospital, error)
}

type sqlHospitalRepository struct {
	db *sql.DB
}

// NewHospitalRepository constructs a new PostgreSQL hospital repository.
func NewHospitalRepository(db *sql.DB) HospitalRepository {
	return &sqlHospitalRepository{db: db}
}

// ExistsByHN checks if a hospital with the given HN exists in table hospital.
func (r *sqlHospitalRepository) ExistsByHN(ctx context.Context, hn string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM hospital WHERE hn = $1)`
	err := r.db.QueryRowContext(ctx, query, hn).Scan(&exists)
	return exists, err
}

// FindByHN retrieves a hospital record by its HN code.
func (r *sqlHospitalRepository) FindByHN(ctx context.Context, hn string) (*models.Hospital, error) {
	query := `SELECT id, hn, name, created_at, updated_at FROM hospital WHERE hn = $1`
	var h models.Hospital
	err := r.db.QueryRowContext(ctx, query, hn).Scan(&h.ID, &h.HN, &h.Name, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &h, nil
}
