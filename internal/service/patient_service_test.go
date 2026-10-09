package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/service"
)

type mockPatientRepository struct {
	searchFunc         func(ctx context.Context, params repository.PatientSearchParams) ([]*models.PatientWithHospital, int, error)
	findByIDFunc       func(ctx context.Context, hospitalID string, id string) (*models.PatientWithHospital, error)
	findByIdentityFunc func(ctx context.Context, hospitalID string, identity string) (*models.PatientWithHospital, error)
	createFunc         func(ctx context.Context, p *models.Patient) (*models.Patient, error)
	deleteFunc         func(ctx context.Context, id string) error
}

func (m *mockPatientRepository) Search(ctx context.Context, params repository.PatientSearchParams) ([]*models.PatientWithHospital, int, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, params)
	}
	return nil, 0, nil
}

func (m *mockPatientRepository) FindByID(ctx context.Context, hospitalID string, id string) (*models.PatientWithHospital, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, hospitalID, id)
	}
	return nil, nil
}

func (m *mockPatientRepository) FindByIdentity(ctx context.Context, hospitalID string, identity string) (*models.PatientWithHospital, error) {
	if m.findByIdentityFunc != nil {
		return m.findByIdentityFunc(ctx, hospitalID, identity)
	}
	return nil, nil
}

func (m *mockPatientRepository) Create(ctx context.Context, p *models.Patient) (*models.Patient, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, p)
	}
	return p, nil
}

func (m *mockPatientRepository) DeleteByID(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestPatientService_Validation(t *testing.T) {
	svc := service.NewPatientService(&mockPatientRepository{})
	ctx := context.Background()

	tests := []struct {
		name        string
		hospitalID  string
		input       service.PatientSearchInput
		expectedErr error
	}{
		{
			name:        "missing hospital id",
			hospitalID:  "",
			input:       service.PatientSearchInput{NationalID: "1234567890123"},
			expectedErr: service.ErrInvalidHospitalID,
		},
		{
			name:        "empty search criteria",
			hospitalID:  "hosp-1",
			input:       service.PatientSearchInput{},
			expectedErr: service.ErrEmptySearchCriteria,
		},
		{
			name:        "whitespace-only criteria",
			hospitalID:  "hosp-1",
			input:       service.PatientSearchInput{FirstName: "   ", LastName: " "},
			expectedErr: service.ErrEmptySearchCriteria,
		},
		{
			name:        "invalid date of birth format",
			hospitalID:  "hosp-1",
			input:       service.PatientSearchInput{DateOfBirth: "1990/05/15"},
			expectedErr: service.ErrInvalidDateOfBirth,
		},
		{
			name:        "invalid date of birth non-date string",
			hospitalID:  "hosp-1",
			input:       service.PatientSearchInput{DateOfBirth: "invalid-date"},
			expectedErr: service.ErrInvalidDateOfBirth,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SearchPatients(ctx, tc.hospitalID, tc.input)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestPatientService_SearchPatients_Success(t *testing.T) {
	dob, _ := time.Parse("2006-01-02", "1992-03-14")
	mockPatients := []*models.PatientWithHospital{
		{
			Patient: models.Patient{
				ID:           "pat-1",
				HospitalID:   "hosp-1",
				PatientHN:    "P001",
				NationalID:   "1100501234567",
				PassportID:   "AA1234567",
				FirstNameTH:  "สมชาย",
				MiddleNameTH: "วิชัย",
				LastNameTH:   "ใจดี",
				FirstNameEN:  "Somchai",
				MiddleNameEN: "Wichai",
				LastNameEN:   "Jaidee",
				DateOfBirth:  dob,
				PhoneNumber:  "0812345678",
				Email:        "somchai@example.com",
				Gender:       models.GenderMale,
			},
			HospitalHN:   "HOSP001",
			HospitalName: "Siriraj Hospital",
		},
	}

	var capturedParams repository.PatientSearchParams
	repo := &mockPatientRepository{
		searchFunc: func(ctx context.Context, params repository.PatientSearchParams) ([]*models.PatientWithHospital, int, error) {
			capturedParams = params
			return mockPatients, 1, nil
		},
	}

	svc := service.NewPatientService(repo)
	input := service.PatientSearchInput{
		FirstName:   "Somchai",
		LastName:    "Jaidee",
		DateOfBirth: "1992-03-14",
	}

	res, err := svc.SearchPatients(context.Background(), "hosp-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.HospitalID != "hosp-1" {
		t.Errorf("expected hospital ID 'hosp-1', got %s", capturedParams.HospitalID)
	}

	if res.Total != 1 {
		t.Errorf("expected total 1, got %d", res.Total)
	}
	if len(res.Patients) != 1 {
		t.Fatalf("expected 1 patient in response, got %d", len(res.Patients))
	}

	p := res.Patients[0]
	if p.ID != "pat-1" || p.PatientHN != "P001" {
		t.Errorf("unexpected patient ID or HN: %v", p)
	}
	if p.FirstNameTH != "สมชาย" || p.FirstNameEN != "Somchai" {
		t.Errorf("unexpected patient names: %v", p)
	}
	if p.DateOfBirth != "1992-03-14" {
		t.Errorf("expected DOB '1992-03-14', got %s", p.DateOfBirth)
	}
}

func TestPatientService_SearchPatients_EmptyResult(t *testing.T) {
	repo := &mockPatientRepository{
		searchFunc: func(ctx context.Context, params repository.PatientSearchParams) ([]*models.PatientWithHospital, int, error) {
			return []*models.PatientWithHospital{}, 0, nil
		},
	}

	svc := service.NewPatientService(repo)
	input := service.PatientSearchInput{
		NationalID: "9999999999999",
	}

	res, err := svc.SearchPatients(context.Background(), "hosp-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Total != 0 {
		t.Errorf("expected total 0, got %d", res.Total)
	}
	if res.Patients == nil {
		t.Fatal("expected non-nil empty slice for Patients, got nil")
	}
	if len(res.Patients) != 0 {
		t.Errorf("expected 0 patients, got %d", len(res.Patients))
	}
}

func TestPatientService_SearchPatients_RepoError(t *testing.T) {
	repo := &mockPatientRepository{
		searchFunc: func(ctx context.Context, params repository.PatientSearchParams) ([]*models.PatientWithHospital, int, error) {
			return nil, 0, errors.New("db query error")
		},
	}

	svc := service.NewPatientService(repo)
	input := service.PatientSearchInput{
		NationalID: "1100501234567",
	}

	_, err := svc.SearchPatients(context.Background(), "hosp-1", input)
	if err == nil {
		t.Fatal("expected repo error, got nil")
	}
}

func TestPatientService_GetPatientByID_Validation(t *testing.T) {
	svc := service.NewPatientService(&mockPatientRepository{})
	ctx := context.Background()

	tests := []struct {
		name        string
		hospitalID  string
		patientID   string
		expectedErr error
	}{
		{
			name:        "missing hospital ID",
			hospitalID:  "",
			patientID:   "1100501234567",
			expectedErr: service.ErrInvalidHospitalID,
		},
		{
			name:        "whitespace hospital ID",
			hospitalID:  "   ",
			patientID:   "1100501234567",
			expectedErr: service.ErrInvalidHospitalID,
		},
		{
			name:        "empty patient identifier",
			hospitalID:  "hosp-1",
			patientID:   "",
			expectedErr: service.ErrInvalidPatientID,
		},
		{
			name:        "whitespace patient identifier",
			hospitalID:  "hosp-1",
			patientID:   "   ",
			expectedErr: service.ErrInvalidPatientID,
		},
		{
			name:        "too short digits",
			hospitalID:  "hosp-1",
			patientID:   "123",
			expectedErr: service.ErrInvalidPatientID,
		},
		{
			name:        "too short alphanumeric",
			hospitalID:  "hosp-1",
			patientID:   "abc",
			expectedErr: service.ErrInvalidPatientID,
		},
		{
			name:        "UUID with hyphens is rejected (must be national_id or passport_id)",
			hospitalID:  "hosp-1",
			patientID:   "e7b23c21-1234-4567-89ab-cdef01234567",
			expectedErr: service.ErrInvalidPatientID,
		},
		{
			name:        "identifier with invalid symbols",
			hospitalID:  "hosp-1",
			patientID:   "110050123456!",
			expectedErr: service.ErrInvalidPatientID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetPatientByID(ctx, tc.hospitalID, tc.patientID)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestPatientService_GetPatientByID_NotFound(t *testing.T) {
	repo := &mockPatientRepository{
		findByIdentityFunc: func(ctx context.Context, hospitalID string, identity string) (*models.PatientWithHospital, error) {
			return nil, nil
		},
	}

	svc := service.NewPatientService(repo)
	res, err := svc.GetPatientByID(context.Background(), "hosp-1", "9999999999999")
	if !errors.Is(err, service.ErrPatientNotFound) {
		t.Fatalf("expected ErrPatientNotFound, got %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result, got %+v", res)
	}

	// Also test non-existent passport ID
	res2, err2 := svc.GetPatientByID(context.Background(), "hosp-1", "US999999999")
	if !errors.Is(err2, service.ErrPatientNotFound) {
		t.Fatalf("expected ErrPatientNotFound for passport, got %v", err2)
	}
	if res2 != nil {
		t.Errorf("expected nil result, got %+v", res2)
	}
}

func TestPatientService_GetPatientByID_Success(t *testing.T) {
	dob, _ := time.Parse("2006-01-02", "1995-10-25")
	validNationalID := "1100501234567"
	validPassportID := "US556677889"

	mockRecord := &models.PatientWithHospital{
		Patient: models.Patient{
			ID:           "e7b23c21-1234-4567-89ab-cdef01234567",
			HospitalID:   "hosp-1",
			PatientHN:    "P00099",
			NationalID:   validNationalID,
			PassportID:   validPassportID,
			FirstNameTH:  "สมชาย",
			MiddleNameTH: "วิชัย",
			LastNameTH:   "ใจดี",
			FirstNameEN:  "Somchai",
			MiddleNameEN: "Wichai",
			LastNameEN:   "Jaidee",
			DateOfBirth:  dob,
			PhoneNumber:  "0812345678",
			Email:        "somchai@example.com",
			Gender:       models.GenderMale,
		},
		HospitalHN:   "HOSP001",
		HospitalName: "Bangkok General",
	}

	repo := &mockPatientRepository{
		findByIdentityFunc: func(ctx context.Context, hospitalID string, identity string) (*models.PatientWithHospital, error) {
			if hospitalID == "hosp-1" && (identity == validNationalID || identity == validPassportID) {
				return mockRecord, nil
			}
			return nil, nil
		},
	}

	svc := service.NewPatientService(repo)

	// 1. Query by National ID
	pByNat, err := svc.GetPatientByID(context.Background(), "hosp-1", validNationalID)
	if err != nil {
		t.Fatalf("unexpected error searching by national ID: %v", err)
	}
	if pByNat == nil || pByNat.NationalID != validNationalID {
		t.Errorf("expected patient with national ID %s, got %+v", validNationalID, pByNat)
	}

	// 2. Query by Passport ID
	pByPass, err := svc.GetPatientByID(context.Background(), "hosp-1", validPassportID)
	if err != nil {
		t.Fatalf("unexpected error searching by passport ID: %v", err)
	}
	if pByPass == nil || pByPass.PassportID != validPassportID {
		t.Errorf("expected patient with passport ID %s, got %+v", validPassportID, pByPass)
	}
}

func TestPatientService_GetPatientByID_RepoError(t *testing.T) {
	repo := &mockPatientRepository{
		findByIdentityFunc: func(ctx context.Context, hospitalID string, identity string) (*models.PatientWithHospital, error) {
			return nil, errors.New("db error")
		},
	}

	svc := service.NewPatientService(repo)
	_, err := svc.GetPatientByID(context.Background(), "hosp-1", "1100501234567")
	if err == nil {
		t.Fatal("expected repo error, got nil")
	}
}

