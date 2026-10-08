package database_test

import (
	"context"
	"testing"
	"time"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
)

// TestConnect_Success tests that database.New successfully connects to PostgreSQL
func TestConnect_Success(t *testing.T) {
	cfg := config.Load()

	db, err := database.New(cfg.DB)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("expected successful ping, got error: %v", err)
	}

	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		t.Fatalf("failed to execute test query: %v", err)
	}

	if result != 1 {
		t.Errorf("expected query result to be 1, got %d", result)
	}
}

// TestConnect_InvalidPassword tests that invalid credentials return an error
func TestConnect_InvalidPassword(t *testing.T) {
	cfg := config.Load()
	badDBConfig := cfg.DB
	badDBConfig.Password = "wrong_password_xyz"

	db, err := database.New(badDBConfig)
	if err == nil {
		if db != nil {
			_ = db.Close()
		}
		t.Fatal("expected error with wrong password, but connection succeeded")
	}
}

// TestConnect_InvalidHost tests that an unreachable host fails and times out gracefully
func TestConnect_InvalidHost(t *testing.T) {
	badDBConfig := config.DBConfig{
		Host:     "192.0.2.1", // RFC 5737 TEST-NET-1 (non-routable IP)
		Port:     "5432",
		User:     "admin",
		Password: "password",
		DBName:   "test_db",
		SSLMode:  "disable",
	}

	start := time.Now()
	db, err := database.New(badDBConfig)
	elapsed := time.Since(start)

	if err == nil {
		if db != nil {
			_ = db.Close()
		}
		t.Fatal("expected connection to fail for unreachable host, but got nil error")
	}

	// Ensure the connection attempt did not hang indefinitely
	if elapsed > 10*time.Second {
		t.Errorf("connection timeout took too long: %v", elapsed)
	}
}
