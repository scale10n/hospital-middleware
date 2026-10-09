package repository_test

import (
	"context"
	"testing"
	"time"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
)

func TestPatientRepository_NilDB(t *testing.T) {
	repo := repository.NewPatientRepository(nil)
	ctx := context.Background()

	_, _, err := repo.Search(ctx, repository.PatientSearchParams{HospitalID: "some-id"})
	if err == nil {
		t.Fatal("expected error with nil db, got nil")
	}

	_, err = repo.Create(ctx, &models.Patient{})
	if err == nil {
		t.Fatal("expected error with nil db, got nil")
	}

	err = repo.DeleteByID(ctx, "some-id")
	if err == nil {
		t.Fatal("expected error with nil db, got nil")
	}
}

func TestPatientRepository_LiveDB(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping live db test: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	hospRepo := repository.NewHospitalRepository(db)
	patientRepo := repository.NewPatientRepository(db)

	// Fetch 2 seeded hospitals to test multi-tenant scoping
	hosp1, err := hospRepo.FindByHN(ctx, "HOSP001")
	if err != nil || hosp1 == nil {
		t.Skipf("skipping test: seeded hospital HOSP001 not found: %v", err)
	}
	hosp2, err := hospRepo.FindByHN(ctx, "HOSP002")
	if err != nil || hosp2 == nil {
		t.Skipf("skipping test: seeded hospital HOSP002 not found: %v", err)
	}

	// Create test patients in hosp1 and hosp2
	dob1, _ := time.Parse("2006-01-02", "1990-05-15")
	patient1 := &models.Patient{
		HospitalID:   hosp1.ID,
		PatientHN:    "P-TEST-001",
		NationalID:   "1100509999001",
		PassportID:   "AA1234567",
		FirstNameTH:  "เรโปสมชาย",
		MiddleNameTH: "เรโปวิชัย",
		LastNameTH:   "เรโปใจดี",
		FirstNameEN:  "RepoSomchai",
		MiddleNameEN: "RepoWichai",
		LastNameEN:   "RepoJaidee",
		DateOfBirth:  dob1,
		PhoneNumber:  "0812345678",
		Email:        "somchai.test@example.com",
		Gender:       models.GenderMale,
	}

	dob2, _ := time.Parse("2006-01-02", "1995-10-20")
	patient2 := &models.Patient{
		HospitalID:   hosp2.ID,
		PatientHN:    "P-TEST-002",
		NationalID:   "2200509999002",
		PassportID:   "BB7654321",
		FirstNameTH:  "เรโปสมหญิง",
		MiddleNameTH: "",
		LastNameTH:   "เรโปใจดี",
		FirstNameEN:  "RepoSomying",
		MiddleNameEN: "",
		LastNameEN:   "RepoJaidee",
		DateOfBirth:  dob2,
		PhoneNumber:  "0898765432",
		Email:        "somying.test@example.com",
		Gender:       models.GenderFemale,
	}

	// Cleanup before & after
	_ = patientRepo.DeleteByID(ctx, patient1.ID)
	_ = patientRepo.DeleteByID(ctx, patient2.ID)
	_, _ = db.ExecContext(ctx, "DELETE FROM patient WHERE patient_hn IN ($1, $2) OR national_id IN ($3, $4)", "P-TEST-001", "P-TEST-002", "1100509999001", "2200509999002")
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM patient WHERE patient_hn IN ($1, $2) OR national_id IN ($3, $4)", "P-TEST-001", "P-TEST-002", "1100509999001", "2200509999002")
	}()

	created1, err := patientRepo.Create(ctx, patient1)
	if err != nil {
		t.Fatalf("failed to create patient1: %v", err)
	}
	defer func() { _ = patientRepo.DeleteByID(ctx, created1.ID) }()

	created2, err := patientRepo.Create(ctx, patient2)
	if err != nil {
		t.Fatalf("failed to create patient2: %v", err)
	}
	defer func() { _ = patientRepo.DeleteByID(ctx, created2.ID) }()

	// 1. Search by NationalID in hosp1
	results, total, err := patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID: hosp1.ID,
		NationalID: "1100509999001",
	})
	if err != nil {
		t.Fatalf("search by national_id error: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected 1 result, got total=%d len=%d", total, len(results))
	}
	if results[0].PatientHN != "P-TEST-001" {
		t.Errorf("expected P-TEST-001, got %s", results[0].PatientHN)
	}

	// 2. Hospital isolation: search for patient1's national_id with hosp2's ID
	results, total, err = patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID: hosp2.ID,
		NationalID: "1100509999001",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Errorf("expected 0 results due to hospital isolation, got total=%d", total)
	}

	// 3. Search case-insensitive and partial name (English): "reposomch"
	results, total, err = patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID: hosp1.ID,
		FirstName:  "reposomch",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected 1 result for 'reposomch', got total=%d", total)
	}
	if results[0].FirstNameEN != "RepoSomchai" {
		t.Errorf("expected FirstNameEN 'RepoSomchai', got %s", results[0].FirstNameEN)
	}

	// 4. Search partial Thai name: "โปสม"
	results, total, err = patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID: hosp1.ID,
		FirstName:  "โปสม",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected 1 result for 'โปสม', got total=%d", total)
	}

	// 5. AND logic: Matching first name "RepoSomchai" AND last name "RepoJaidee"
	results, total, err = patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID: hosp1.ID,
		FirstName:  "RepoSomchai",
		LastName:   "RepoJaidee",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 result with AND logic, got total=%d", total)
	}

	// 6. AND logic mismatch: First name "RepoSomchai" AND last name "NonExistent"
	results, total, err = patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID: hosp1.ID,
		FirstName:  "RepoSomchai",
		LastName:   "NonExistent",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 results with mismatched AND condition, got total=%d", total)
	}

	// 7. Search by DateOfBirth: "1990-05-15"
	results, total, err = patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID:  hosp1.ID,
		DateOfBirth: "1990-05-15",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 result for DOB, got %d", total)
	}

	// 8. All matching patients returned without pagination
	results, total, err = patientRepo.Search(ctx, repository.PatientSearchParams{
		HospitalID: hosp1.ID,
		LastName:   "RepoJaidee",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Errorf("expected total=1 and len=1, got total=%d len=%d", total, len(results))
	}
}
