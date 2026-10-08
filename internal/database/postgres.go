package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"hospital-middleware/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Default connection pool configurations
const (
	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 25
	defaultConnMaxLifetime = 5 * time.Minute
	defaultConnMaxIdleTime = 5 * time.Minute
	defaultPingTimeout     = 5 * time.Second
)

// New initializes a new PostgreSQL connection pool and verifies connectivity
func New(cfg config.DBConfig) (*sql.DB, error) {
	// Register and open pgx connection using DSN
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pooling with fallbacks to defaults
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = defaultMaxOpenConns
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = defaultMaxIdleConns
	}
	lifetime := time.Duration(cfg.ConnMaxLifetimeMin) * time.Minute
	if lifetime <= 0 {
		lifetime = defaultConnMaxLifetime
	}
	idleTime := time.Duration(cfg.ConnMaxIdleTimeMin) * time.Minute
	if idleTime <= 0 {
		idleTime = defaultConnMaxIdleTime
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)
	db.SetConnMaxIdleTime(idleTime)

	// Verify connectivity with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
