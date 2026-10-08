package seeds

import (
	"context"
	"testing"
)

func TestHospitalSeeds_DataIntegrity(t *testing.T) {
	if len(DefaultHospitals) == 0 {
		t.Fatal("expected DefaultHospitals to contain records, got 0")
	}

	seenHN := make(map[string]bool)
	for i, h := range DefaultHospitals {
		if h.HN == "" {
			t.Errorf("record %d: HN cannot be empty", i)
		}
		if h.Name == "" {
			t.Errorf("record %d: Name cannot be empty", i)
		}
		if seenHN[h.HN] {
			t.Errorf("record %d: duplicate HN %q found", i, h.HN)
		}
		seenHN[h.HN] = true
	}
}

func TestSeedHospitals_NilDB(t *testing.T) {
	ctx := context.Background()
	err := SeedHospitals(ctx, nil)
	if err == nil {
		t.Fatal("expected error when passing nil db, got nil")
	}
}

func TestRunAll_NilDB(t *testing.T) {
	ctx := context.Background()
	err := RunAll(ctx, nil)
	if err == nil {
		t.Fatal("expected error when passing nil db, got nil")
	}
}
