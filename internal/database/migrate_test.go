package database_test

import (
	"testing"
	"testing/fstest"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/migrations"
)

func TestRunMigrations_NilDB(t *testing.T) {
	mockFS := fstest.MapFS{
		"000001_test.up.sql":   &fstest.MapFile{Data: []byte("CREATE TABLE dummy (id INT);")},
		"000001_test.down.sql": &fstest.MapFile{Data: []byte("DROP TABLE dummy;")},
	}

	err := database.RunMigrations(nil, mockFS)
	if err == nil {
		t.Fatal("expected error when running migrations with nil db, got nil")
	}
}

// TestRunMigrations_RealFS tests applying the actual embedded SQL migrations from migrations.FS
func TestRunMigrations_RealFS(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL is not available (%v)", err)
	}
	defer db.Close()

	// 1. Apply all pending migrations using actual embedded SQL files
	if err := database.RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// 2. Test idempotency: re-running when already up-to-date must not error
	if err := database.RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("expected no error when re-running up-to-date migrations, got: %v", err)
	}

	// 3. Verify that expected domain tables exist
	expectedTables := []string{"hospital", "staff", "patient", "staff_session"}
	for _, table := range expectedTables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		)`
		if err := db.QueryRow(query, table).Scan(&exists); err != nil {
			t.Fatalf("failed to query table %s existence: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %q to exist in database, but it does not", table)
		}
	}
}
