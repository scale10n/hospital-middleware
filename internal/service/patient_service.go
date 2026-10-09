package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hospital-middleware/internal/repository"
)

var (
	// ErrEmptySearchCriteria is returned when no search parameters are provided.
	ErrEmptySearchCriteria = errors.New("at least one search criterion must be provided")
	// ErrInvalidDateOfBirth is returned when date_of_birth does not match YYYY-MM-DD.
	ErrInvalidDateOfBirth = errors.New("invalid date_of_birth format: must be YYYY-MM-DD")
	// ErrInvalidHospitalID is returned when staff hospital context is missing.
	ErrInvalidHospitalID = errors.New("hospital ID is required")
)

// PatientSearchInput holds the user-supplied query or payload for patient search.
type PatientSearchInput struct {
	NationalID  string `json:"national_id"`
	PassportID  string `json:"passport_id"`
	FirstName   string `json:"first_name"`
	MiddleName  string `json:"middle_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
}

// Validate sanitizes input and enforces search criteria & pagination constraints.
func (in *PatientSearchInput) Validate() error {
	in.NationalID = strings.TrimSpace(in.NationalID)
	in.PassportID = strings.TrimSpace(in.PassportID)
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.MiddleName = strings.TrimSpace(in.MiddleName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.DateOfBirth = strings.TrimSpace(in.DateOfBirth)
	in.PhoneNumber = strings.TrimSpace(in.PhoneNumber)
	in.Email = strings.TrimSpace(in.Email)

	// At least one search criterion must be provided
	if in.NationalID == "" &&
		in.PassportID == "" &&
		in.FirstName == "" &&
		in.MiddleName == "" &&
		in.LastName == "" &&
		in.DateOfBirth == "" &&
		in.PhoneNumber == "" &&
		in.Email == "" {
		return ErrEmptySearchCriteria
	}

	// Validate date_of_birth format if provided
	if in.DateOfBirth != "" {
		if _, err := time.Parse("2006-01-02", in.DateOfBirth); err != nil {
			return ErrInvalidDateOfBirth
		}
	}

	return nil
}

// PatientResponse represents a serialized patient record in API responses.
type PatientResponse struct {
	// Unique identifier (UUID) of the patient record
	ID string `json:"id" example:"e7b23c21-1234-4567-89ab-cdef01234567"`
	// Hospital Number (HN) assigned to the patient within the hospital
	PatientHN string `json:"patient_hn" example:"P00020"`
	// Patient first name in Thai
	FirstNameTH string `json:"first_name_th" example:"ชลธิชา"`
	// Patient middle name in Thai (optional)
	MiddleNameTH string `json:"middle_name_th" example:""`
	// Patient last name in Thai
	LastNameTH string `json:"last_name_th" example:"สว่างจิตต์"`
	// Patient first name in English
	FirstNameEN string `json:"first_name_en" example:"Chonthicha"`
	// Patient middle name in English (optional)
	MiddleNameEN string `json:"middle_name_en" example:""`
	// Patient last name in English
	LastNameEN string `json:"last_name_en" example:"Sawangjit"`
	// Date of birth formatted as YYYY-MM-DD
	DateOfBirth string `json:"date_of_birth" example:"2004-10-15"`
	// 13-digit Thai National Identification Number
	NationalID string `json:"national_id" example:"1100505566779"`
	// Passport number for foreign patients
	PassportID string `json:"passport_id" example:""`
	// Contact telephone number
	PhoneNumber string `json:"phone_number" example:"0832223333"`
	// Contact email address
	Email string `json:"email" example:"chonthicha.s@example.com"`
	// Biological gender ('M' for Male, 'F' for Female, 'O' for Other)
	Gender string `json:"gender" example:"F"`
}

// PatientSearchResult contains patients matching the search criteria.
type PatientSearchResult struct {
	// List of matched patient records
	Patients []PatientResponse `json:"patients"`
	// Total count of patients matching the criteria
	Total int `json:"total" example:"1"`
}

// PatientService defines business operations for patients.
type PatientService interface {
	SearchPatients(ctx context.Context, hospitalID string, input PatientSearchInput) (*PatientSearchResult, error)
}

type patientService struct {
	patientRepo repository.PatientRepository
}

// NewPatientService constructs a new PatientService.
func NewPatientService(patientRepo repository.PatientRepository) PatientService {
	return &patientService{patientRepo: patientRepo}
}

// SearchPatients validates the search input and queries patients scoped to the staff's hospital.
func (s *patientService) SearchPatients(ctx context.Context, hospitalID string, input PatientSearchInput) (*PatientSearchResult, error) {
	hospitalID = strings.TrimSpace(hospitalID)
	if hospitalID == "" {
		return nil, ErrInvalidHospitalID
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	if s.patientRepo == nil {
		return nil, errors.New("patient repository is not configured")
	}

	searchParams := repository.PatientSearchParams{
		HospitalID:  hospitalID,
		NationalID:  input.NationalID,
		PassportID:  input.PassportID,
		FirstName:   input.FirstName,
		MiddleName:  input.MiddleName,
		LastName:    input.LastName,
		DateOfBirth: input.DateOfBirth,
		PhoneNumber: input.PhoneNumber,
		Email:       input.Email,
	}

	records, total, err := s.patientRepo.Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}

	patientResponses := make([]PatientResponse, 0, len(records))
	for _, r := range records {
		patientResponses = append(patientResponses, PatientResponse{
			ID:           r.ID,
			PatientHN:    r.PatientHN,
			FirstNameTH:  r.FirstNameTH,
			MiddleNameTH: r.MiddleNameTH,
			LastNameTH:   r.LastNameTH,
			FirstNameEN:  r.FirstNameEN,
			MiddleNameEN: r.MiddleNameEN,
			LastNameEN:   r.LastNameEN,
			DateOfBirth:  r.DateOfBirth.Format("2006-01-02"),
			NationalID:   r.NationalID,
			PassportID:   r.PassportID,
			PhoneNumber:  r.PhoneNumber,
			Email:        r.Email,
			Gender:       string(r.Gender),
		})
	}

	return &PatientSearchResult{
		Patients: patientResponses,
		Total:    total,
	}, nil
}
