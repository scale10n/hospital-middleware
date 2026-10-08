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

func TestStaffSessionRepository_LiveDB(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping live db test: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	hospRepo := repository.NewHospitalRepository(db)
	staffRepo := repository.NewStaffRepository(db)
	staffSessionRepo := repository.NewStaffSessionRepository(db)

	hosp, err := hospRepo.FindByHN(ctx, "HOSP001")
	if err != nil || hosp == nil {
		t.Skipf("skipping test: seeded hospital HOSP001 not available: %v", err)
	}

	testUsername := "session_test_staff"
	_, _ = db.ExecContext(ctx, "DELETE FROM staff WHERE hospital_id = $1 AND username = $2", hosp.ID, testUsername)
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM staff WHERE hospital_id = $1 AND username = $2", hosp.ID, testUsername)
	}()

	staff, err := staffRepo.Create(ctx, &models.Staff{
		HospitalID:   hosp.ID,
		Username:     testUsername,
		PasswordHash: "$2a$10$hashedpasswordplaceholder",
	})
	if err != nil {
		t.Fatalf("failed to create test staff: %v", err)
	}

	testToken := "test-jwt-token-string-12345"

	// 1. Create session
	newSession := &models.StaffSession{
		StaffID:   staff.ID,
		Token:     testToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	createdSession, err := staffSessionRepo.Create(ctx, newSession)
	if err != nil {
		t.Fatalf("failed to create staff session: %v", err)
	}
	if createdSession.ID == "" {
		t.Errorf("expected non-empty session ID")
	}
	if createdSession.Token != testToken {
		t.Errorf("expected token %q, got %q", testToken, createdSession.Token)
	}
	if createdSession.StaffID != staff.ID {
		t.Errorf("expected staff ID %q, got %q", staff.ID, createdSession.StaffID)
	}

	// 2. Find by token
	found, err := staffSessionRepo.FindByToken(ctx, testToken)
	if err != nil {
		t.Fatalf("failed to find session by token: %v", err)
	}
	if found == nil || found.ID != createdSession.ID {
		t.Errorf("expected found session ID %q, got %+v", createdSession.ID, found)
	}

	// 3. Delete by token
	if err := staffSessionRepo.DeleteByToken(ctx, testToken); err != nil {
		t.Fatalf("failed to delete session by token: %v", err)
	}

	notFound, err := staffSessionRepo.FindByToken(ctx, testToken)
	if err != nil {
		t.Fatalf("unexpected error after delete: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil after session deletion, got: %+v", notFound)
	}
}
