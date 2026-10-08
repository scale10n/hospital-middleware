package database

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies all pending migrations using the given filesystem.
func RunMigrations(db *sql.DB, fsys fs.FS) error {
	if db == nil {
		return errors.New("cannot run migrations on nil db")
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver for migration: %w", err)
	}

	sourceDriver, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("[INFO] Database migrations: schema is already up to date")
			return nil
		}
		return fmt.Errorf("failed to apply database migrations: %w", err)
	}

	log.Println("[INFO] Database migrations applied successfully")
	return nil
}
