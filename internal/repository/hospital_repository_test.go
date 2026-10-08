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
}
