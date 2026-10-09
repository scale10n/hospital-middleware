package repository_test

import (
	"context"
	"testing"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
)

func TestStaffRepository_LiveDB(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping live db test: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	hospRepo := repository.NewHospitalRepository(db)
	staffRepo := repository.NewStaffRepository(db)

	// Fetch a seeded hospital to get a valid hospital UUID
	hosp, err := hospRepo.FindByHN(ctx, "HOSP001")
	if err != nil || hosp == nil {
		t.Skipf("skipping test: seeded hospital HOSP001 not available: %v", err)
	}

	testUsername := "repo_test_staff"

	// Cleanup test record before and after
	_, _ = db.ExecContext(ctx, "DELETE FROM staff WHERE hospital_id = $1 AND username = $2", hosp.ID, testUsername)
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM staff WHERE hospital_id = $1 AND username = $2", hosp.ID, testUsername)
	}()

	// 1. Check exists before create
	existsBefore, err := staffRepo.ExistsByHospitalAndUsername(ctx, hosp.ID, testUsername)
	if err != nil {
		t.Fatalf("unexpected error checking existence: %v", err)
	}
	if existsBefore {
		t.Errorf("expected staff to not exist before creation")
	}

	// 2. Create staff
	newStaff := &models.Staff{
		HospitalID:   hosp.ID,
		Username:     testUsername,
		PasswordHash: "$2a$10$hashedpasswordplaceholder",
	}

	created, err := staffRepo.Create(ctx, newStaff)
	if err != nil {
		t.Fatalf("unexpected error creating staff: %v", err)
	}
	if created.ID == "" {
		t.Errorf("expected non-empty ID for created staff")
	}
	if created.Username != testUsername {
		t.Errorf("expected username %q, got %q", testUsername, created.Username)
	}
	if created.HospitalID != hosp.ID {
		t.Errorf("expected hospital ID %q, got %q", hosp.ID, created.HospitalID)
	}

	// 3. Check exists after create
	existsAfter, err := staffRepo.ExistsByHospitalAndUsername(ctx, hosp.ID, testUsername)
	if err != nil {
		t.Fatalf("unexpected error checking existence: %v", err)
	}
	if !existsAfter {
		t.Errorf("expected staff to exist after creation")
	}

	// 4. Find by hospital and username
	found, err := staffRepo.FindByHospitalAndUsername(ctx, hosp.ID, testUsername)
	if err != nil {
		t.Fatalf("unexpected error finding staff: %v", err)
	}
	if found == nil || found.ID != created.ID {
		t.Errorf("expected found staff ID %v, got %+v", created.ID, found)
	}

	// 5. Find by username
	foundByUname, err := staffRepo.FindByUsername(ctx, testUsername)
	if err != nil {
		t.Fatalf("unexpected error finding staff by username: %v", err)
	}
	if foundByUname == nil || foundByUname.ID != created.ID {
		t.Errorf("expected found staff ID %v, got %+v", created.ID, foundByUname)
	}

	// 6. Find non-existent username
	notFound, err := staffRepo.FindByUsername(ctx, "non_existent_staff_xyz")
	if err != nil {
		t.Fatalf("unexpected error querying non-existent username: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent staff, got %+v", notFound)
	}

	// 7. Find non-existent staff by hospital and username
	notFoundByHosp, err := staffRepo.FindByHospitalAndUsername(ctx, hosp.ID, "non_existent_staff_xyz")
	if err != nil {
		t.Fatalf("unexpected error querying non-existent staff by hospital and username: %v", err)
	}
	if notFoundByHosp != nil {
		t.Errorf("expected nil for non-existent staff, got %+v", notFoundByHosp)
	}

	// 8. SQL Injection payload safety
	sqlInjPayload := "' OR '1'='1'; DROP TABLE staff; --"
	injFound, err := staffRepo.FindByHospitalAndUsername(ctx, hosp.ID, sqlInjPayload)
	if err != nil {
		t.Fatalf("unexpected error executing query with SQL injection string: %v", err)
	}
	if injFound != nil {
		t.Errorf("expected nil for SQL injection query, got %+v", injFound)
	}
}
