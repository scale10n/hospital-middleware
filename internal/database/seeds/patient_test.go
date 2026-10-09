package seeds

import (
	"context"
	"testing"
)

func TestPatientSeeds_DataIntegrity(t *testing.T) {
	if len(DefaultPatients) == 0 {
		t.Fatal("expected DefaultPatients to contain records, got 0")
	}

	seenKey := make(map[string]bool)
	for i, p := range DefaultPatients {
		if p.HospitalHN == "" {
			t.Errorf("record %d: HospitalHN cannot be empty", i)
		}
		if p.PatientHN == "" {
			t.Errorf("record %d: PatientHN cannot be empty", i)
		}
		if p.NationalID == "" && p.PassportID == "" {
			t.Errorf("record %d: either NationalID or PassportID must be provided", i)
		}
		key := p.HospitalHN + ":" + p.PatientHN
		if seenKey[key] {
			t.Errorf("record %d: duplicate key %q found", i, key)
		}
		seenKey[key] = true
	}
}

func TestSeedPatients_NilDB(t *testing.T) {
	ctx := context.Background()
	err := SeedPatients(ctx, nil)
	if err == nil {
		t.Fatal("expected error when passing nil db, got nil")
	}
}
