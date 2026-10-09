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

	// 2. Seed Patient mockup records
	if err := SeedPatients(ctx, db); err != nil {
		return fmt.Errorf("patient seeder failed: %w", err)
	}

	return nil
}
