package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"hospital-middleware/internal/models"
)

// PatientSearchParams holds filter parameters for patient queries.
type PatientSearchParams struct {
	HospitalID  string
	NationalID  string
	PassportID  string
	FirstName   string
	MiddleName  string
	LastName    string
	DateOfBirth string // Format: YYYY-MM-DD
	PhoneNumber string
	Email       string
}

// PatientRepository defines data persistence operations for patients.
type PatientRepository interface {
	Search(ctx context.Context, params PatientSearchParams) ([]*models.PatientWithHospital, int, error)
	FindByID(ctx context.Context, hospitalID string, id string) (*models.PatientWithHospital, error)
	FindByIdentity(ctx context.Context, hospitalID string, identity string) (*models.PatientWithHospital, error)
	Create(ctx context.Context, p *models.Patient) (*models.Patient, error)
	DeleteByID(ctx context.Context, id string) error
}

type patientRepository struct {
	db *sql.DB
}

// NewPatientRepository constructs a new PatientRepository.
func NewPatientRepository(db *sql.DB) PatientRepository {
	return &patientRepository{db: db}
}

// escapeLike escapes characters that have special meaning in SQL LIKE / ILIKE patterns.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// Search queries patients matching ALL provided filter criteria within the staff's hospital.
// It supports case-insensitive partial matching (ILIKE) on first, middle, and last names in Thai and English.
func (r *patientRepository) Search(ctx context.Context, params PatientSearchParams) ([]*models.PatientWithHospital, int, error) {
	if r.db == nil {
		return nil, 0, errors.New("database connection is nil")
	}

	var conditions []string
	var args []any
	argIdx := 1

	// Strict hospital tenant isolation
	conditions = append(conditions, fmt.Sprintf("p.hospital_id = $%d", argIdx))
	args = append(args, params.HospitalID)
	argIdx++

	if params.NationalID != "" {
		conditions = append(conditions, fmt.Sprintf("p.national_id = $%d", argIdx))
		args = append(args, params.NationalID)
		argIdx++
	}

	if params.PassportID != "" {
		conditions = append(conditions, fmt.Sprintf("p.passport_id ILIKE $%d ESCAPE '\\'", argIdx))
		args = append(args, escapeLike(params.PassportID))
		argIdx++
	}

	if params.DateOfBirth != "" {
		conditions = append(conditions, fmt.Sprintf("p.date_of_birth = $%d::DATE", argIdx))
		args = append(args, params.DateOfBirth)
		argIdx++
	}

	if params.PhoneNumber != "" {
		conditions = append(conditions, fmt.Sprintf("p.phone_number = $%d", argIdx))
		args = append(args, params.PhoneNumber)
		argIdx++
	}

	if params.Email != "" {
		conditions = append(conditions, fmt.Sprintf("p.email ILIKE $%d ESCAPE '\\'", argIdx))
		args = append(args, escapeLike(params.Email))
		argIdx++
	}

	if params.FirstName != "" {
		pattern := "%" + escapeLike(params.FirstName) + "%"
		conditions = append(conditions, fmt.Sprintf("(p.first_name_th ILIKE $%d ESCAPE '\\' OR p.first_name_en ILIKE $%d ESCAPE '\\')", argIdx, argIdx))
		args = append(args, pattern)
		argIdx++
	}

	if params.MiddleName != "" {
		pattern := "%" + escapeLike(params.MiddleName) + "%"
		conditions = append(conditions, fmt.Sprintf("(p.middle_name_th ILIKE $%d ESCAPE '\\' OR p.middle_name_en ILIKE $%d ESCAPE '\\')", argIdx, argIdx))
		args = append(args, pattern)
		argIdx++
	}

	if params.LastName != "" {
		pattern := "%" + escapeLike(params.LastName) + "%"
		conditions = append(conditions, fmt.Sprintf("(p.last_name_th ILIKE $%d ESCAPE '\\' OR p.last_name_en ILIKE $%d ESCAPE '\\')", argIdx, argIdx))
		args = append(args, pattern)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	selectQuery := fmt.Sprintf(`
		SELECT 
			p.id,
			p.hospital_id,
			p.patient_hn,
			COALESCE(p.national_id, ''),
			COALESCE(p.passport_id, ''),
			COALESCE(p.first_name_th, ''),
			COALESCE(p.middle_name_th, ''),
			COALESCE(p.last_name_th, ''),
			COALESCE(p.first_name_en, ''),
			COALESCE(p.middle_name_en, ''),
			COALESCE(p.last_name_en, ''),
			p.date_of_birth,
			COALESCE(p.phone_number, ''),
			COALESCE(p.email, ''),
			p.gender,
			p.created_at,
			p.updated_at,
			h.hn AS hospital_hn,
			h.name AS hospital_name
		FROM patient p
		JOIN hospital h ON p.hospital_id = h.id
		WHERE %s
		ORDER BY p.created_at DESC, p.id
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query patients: %w", err)
	}
	defer rows.Close()

	patients := make([]*models.PatientWithHospital, 0)
	for rows.Next() {
		var item models.PatientWithHospital
		var dob time.Time
		var genderStr string

		if err := rows.Scan(
			&item.ID,
			&item.HospitalID,
			&item.PatientHN,
			&item.NationalID,
			&item.PassportID,
			&item.FirstNameTH,
			&item.MiddleNameTH,
			&item.LastNameTH,
			&item.FirstNameEN,
			&item.MiddleNameEN,
			&item.LastNameEN,
			&dob,
			&item.PhoneNumber,
			&item.Email,
			&genderStr,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.HospitalHN,
			&item.HospitalName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan patient row: %w", err)
		}

		item.DateOfBirth = dob
		item.Gender = models.Gender(genderStr)
		patients = append(patients, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate patient rows: %w", err)
	}

	return patients, len(patients), nil
}

// Create inserts a new patient record into the database.
func (r *patientRepository) Create(ctx context.Context, p *models.Patient) (*models.Patient, error) {
	if r.db == nil {
		return nil, errors.New("database connection is nil")
	}

	query := `
		INSERT INTO patient (
			hospital_id,
			patient_hn,
			national_id,
			passport_id,
			first_name_th,
			middle_name_th,
			last_name_th,
			first_name_en,
			middle_name_en,
			last_name_en,
			date_of_birth,
			phone_number,
			email,
			gender
		) VALUES (
			$1, $2,
			NULLIF($3, ''), NULLIF($4, ''),
			NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''),
			NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
			$11,
			NULLIF($12, ''), NULLIF($13, ''),
			$14
		)
		RETURNING id, created_at, updated_at
	`

	var id string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(
		ctx,
		query,
		p.HospitalID,
		p.PatientHN,
		p.NationalID,
		p.PassportID,
		p.FirstNameTH,
		p.MiddleNameTH,
		p.LastNameTH,
		p.FirstNameEN,
		p.MiddleNameEN,
		p.LastNameEN,
		p.DateOfBirth,
		p.PhoneNumber,
		p.Email,
		string(p.Gender),
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert patient: %w", err)
	}

	p.ID = id
	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt
	return p, nil
}

// FindByID retrieves a patient by hospital ID and patient UUID.
// Returns nil, nil if the patient does not exist or belongs to another hospital.
func (r *patientRepository) FindByID(ctx context.Context, hospitalID string, id string) (*models.PatientWithHospital, error) {
	if r.db == nil {
		return nil, errors.New("database connection is nil")
	}

	query := `
		SELECT 
			p.id,
			p.hospital_id,
			p.patient_hn,
			COALESCE(p.national_id, ''),
			COALESCE(p.passport_id, ''),
			COALESCE(p.first_name_th, ''),
			COALESCE(p.middle_name_th, ''),
			COALESCE(p.last_name_th, ''),
			COALESCE(p.first_name_en, ''),
			COALESCE(p.middle_name_en, ''),
			COALESCE(p.last_name_en, ''),
			p.date_of_birth,
			COALESCE(p.phone_number, ''),
			COALESCE(p.email, ''),
			p.gender,
			p.created_at,
			p.updated_at,
			h.hn AS hospital_hn,
			h.name AS hospital_name
		FROM patient p
		JOIN hospital h ON p.hospital_id = h.id
		WHERE p.hospital_id = $1 AND p.id = $2
	`

	var item models.PatientWithHospital
	var dob time.Time
	var genderStr string

	err := r.db.QueryRowContext(ctx, query, hospitalID, id).Scan(
		&item.ID,
		&item.HospitalID,
		&item.PatientHN,
		&item.NationalID,
		&item.PassportID,
		&item.FirstNameTH,
		&item.MiddleNameTH,
		&item.LastNameTH,
		&item.FirstNameEN,
		&item.MiddleNameEN,
		&item.LastNameEN,
		&dob,
		&item.PhoneNumber,
		&item.Email,
		&genderStr,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.HospitalHN,
		&item.HospitalName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query patient by id: %w", err)
	}

	item.DateOfBirth = dob
	item.Gender = models.Gender(genderStr)
	return &item, nil
}

// FindByIdentity retrieves a patient by hospital ID and national_id OR passport_id.
// Returns nil, nil if the patient does not exist or belongs to another hospital.
func (r *patientRepository) FindByIdentity(ctx context.Context, hospitalID string, identity string) (*models.PatientWithHospital, error) {
	if r.db == nil {
		return nil, errors.New("database connection is nil")
	}

	query := `
		SELECT 
			p.id,
			p.hospital_id,
			p.patient_hn,
			COALESCE(p.national_id, ''),
			COALESCE(p.passport_id, ''),
			COALESCE(p.first_name_th, ''),
			COALESCE(p.middle_name_th, ''),
			COALESCE(p.last_name_th, ''),
			COALESCE(p.first_name_en, ''),
			COALESCE(p.middle_name_en, ''),
			COALESCE(p.last_name_en, ''),
			p.date_of_birth,
			COALESCE(p.phone_number, ''),
			COALESCE(p.email, ''),
			p.gender,
			p.created_at,
			p.updated_at,
			h.hn AS hospital_hn,
			h.name AS hospital_name
		FROM patient p
		JOIN hospital h ON p.hospital_id = h.id
		WHERE p.hospital_id = $1 AND (p.national_id = $2 OR p.passport_id ILIKE $2)
		LIMIT 1
	`

	var item models.PatientWithHospital
	var dob time.Time
	var genderStr string

	err := r.db.QueryRowContext(ctx, query, hospitalID, identity).Scan(
		&item.ID,
		&item.HospitalID,
		&item.PatientHN,
		&item.NationalID,
		&item.PassportID,
		&item.FirstNameTH,
		&item.MiddleNameTH,
		&item.LastNameTH,
		&item.FirstNameEN,
		&item.MiddleNameEN,
		&item.LastNameEN,
		&dob,
		&item.PhoneNumber,
		&item.Email,
		&genderStr,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.HospitalHN,
		&item.HospitalName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query patient by identity: %w", err)
	}

	item.DateOfBirth = dob
	item.Gender = models.Gender(genderStr)
	return &item, nil
}

// DeleteByID removes a patient record by ID (useful for test teardown).
func (r *patientRepository) DeleteByID(ctx context.Context, id string) error {
	if r.db == nil {
		return errors.New("database connection is nil")
	}
	_, err := r.db.ExecContext(ctx, "DELETE FROM patient WHERE id = $1", id)
	return err
}

