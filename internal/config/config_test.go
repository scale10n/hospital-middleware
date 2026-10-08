package config_test

import (
	"strings"
	"testing"

	"hospital-middleware/internal/config"
)

func TestConfig_DSN(t *testing.T) {
	cfg := config.DBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "admin",
		Password: "secretpassword",
		DBName:   "hospital_middleware_db",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()

	expectedParts := []string{
		"host=localhost",
		"port=5432",
		"user=admin",
		"password=secretpassword",
		"dbname=hospital_middleware_db",
		"sslmode=disable",
	}

	for _, part := range expectedParts {
		if !strings.Contains(dsn, part) {
			t.Errorf("expected DSN to contain %q, but got %q", part, dsn)
		}
	}
}

func TestConfig_URL(t *testing.T) {
	cfg := config.DBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "admin",
		Password: "secretpassword",
		DBName:   "hospital_middleware_db",
		SSLMode:  "disable",
	}

	url := cfg.URL()
	expected := "postgres://admin:secretpassword@localhost:5432/hospital_middleware_db?sslmode=disable"
	if url != expected {
		t.Errorf("expected URL to be %q, got %q", expected, url)
	}
}

func TestConfig_LoadDefaults(t *testing.T) {
	cfg := config.Load()
	if cfg == nil {
		t.Fatal("expected config to not be nil")
	}

	if cfg.Server.Port == "" {
		t.Error("expected default port to be non-empty")
	}

	if cfg.DB.DBName == "" {
		t.Error("expected default DBName to be non-empty")
	}

	if !cfg.DB.AutoMigrate {
		t.Error("expected default AutoMigrate to be true")
	}
}
