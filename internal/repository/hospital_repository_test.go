package repository_test

import (
	"context"
	"testing"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/repository"
)

func TestHospitalRepository_LiveDB(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping live db test: %v", err)
	}
	defer db.Close()

	repo := repository.NewHospitalRepository(db)
	ctx := context.Background()

	// 1. Existing HN (seeded in database)
	exists, err := repo.ExistsByHN(ctx, "HOSP001")
	if err != nil {
		t.Fatalf("unexpected error checking HOSP001: %v", err)
	}
	if !exists {
		t.Errorf("expected HOSP001 to exist in database")
	}

	h, err := repo.FindByHN(ctx, "HOSP001")
	if err != nil {
		t.Fatalf("unexpected error finding HOSP001: %v", err)
	}
	if h == nil || h.HN != "HOSP001" {
		t.Errorf("expected hospital with HN HOSP001, got %+v", h)
	}

	// 2. Non-existent HN
	notExists, err := repo.ExistsByHN(ctx, "NON_EXISTENT_HN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notExists {
		t.Errorf("expected NON_EXISTENT_HN to not exist")
	}

	// 3. FindByID
	if h != nil {
		byID, err := repo.FindByID(ctx, h.ID)
		if err != nil {
			t.Fatalf("unexpected error finding hospital by ID: %v", err)
		}
		if byID == nil || byID.ID != h.ID {
			t.Errorf("expected hospital ID %v, got %+v", h.ID, byID)
		}
	}

	// 4. Non-existent ID
	badID, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("unexpected error finding non-existent ID: %v", err)
	}
	if badID != nil {
		t.Errorf("expected nil for non-existent ID, got %+v", badID)
	}

	// 5. SQL Injection payload safety
	sqlInjHN := "' OR '1'='1' --"
	injExists, err := repo.ExistsByHN(ctx, sqlInjHN)
	if err != nil {
		t.Fatalf("unexpected error querying with SQL injection payload: %v", err)
	}
	if injExists {
		t.Errorf("expected SQL injection payload to not match any hospital")
	}

	injHosp, err := repo.FindByHN(ctx, sqlInjHN)
	if err != nil {
		t.Fatalf("unexpected error querying with SQL injection payload: %v", err)
	}
	if injHosp != nil {
		t.Errorf("expected nil for SQL injection payload, got %+v", injHosp)
	}
}
