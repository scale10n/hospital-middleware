package seeds

import (
	"context"
	"database/sql"
	"fmt"
)

// RunAll executes all registered database seeders in logical sequence.
func RunAll(ctx context.Context, db *sql.DB) error {
	// 1. Seed Hospital master records
	if err := SeedHospitals(ctx, db); err != nil {
		return fmt.Errorf("hospital seeder failed: %w", err)
	}

	// Additional future seeders (e.g. Staff, Patients) can be chained here:
	// if err := SeedStaff(ctx, db); err != nil { return err }

	return nil
}
