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

func TestConfig_URL_SpecialCharacters(t *testing.T) {
	cfg := config.DBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "admin:special",
		Password: "P@ss:w/rd?#%",
		DBName:   "/hospital_db",
		SSLMode:  "disable",
	}

	url := cfg.URL()
	// RFC 3986 requires escaping reserved characters in credentials
	if !strings.Contains(url, "admin%3Aspecial:P%40ss%3Aw%2Frd%3F%23%25@localhost:5432/hospital_db") {
		t.Errorf("expected URL credentials to be RFC 3986 escaped, got: %s", url)
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

	if cfg.JWT.Secret == "" {
		t.Error("expected default JWT secret to be non-empty")
	}

	if cfg.JWT.ExpirySeconds <= 0 {
		t.Errorf("expected default JWT expiry seconds > 0, got %d", cfg.JWT.ExpirySeconds)
	}

	if len(cfg.CORS.AllowedOrigins) == 0 {
		t.Error("expected default CORS AllowedOrigins to not be empty")
	}

	if cfg.CORS.MaxAgeSeconds <= 0 {
		t.Errorf("expected default CORS max age seconds > 0, got %d", cfg.CORS.MaxAgeSeconds)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected default config to pass validation, got: %v", err)
	}
}

func TestConfig_Validate(t *testing.T) {
	validConfig := func() *config.Config {
		return &config.Config{
			Server: config.ServerConfig{Port: "8080", Mode: "debug"},
			DB: config.DBConfig{
				Host:   "localhost",
				Port:   "5432",
				DBName: "hospital_db",
			},
			JWT: config.JWTConfig{
				Secret:        "this-is-a-valid-secret-key-that-is-at-least-32-chars-long",
				ExpirySeconds: 86400,
			},
			CORS: config.CORSConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
				MaxAgeSeconds:  86400,
			},
		}
	}

	t.Run("valid configuration passes", func(t *testing.T) {
		cfg := validConfig()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid config to pass, got: %v", err)
		}
	})

	t.Run("empty port fails", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Port = ""
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "port cannot be empty") {
			t.Fatalf("expected error for empty port, got: %v", err)
		}
	})

	t.Run("invalid port number fails", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Port = "70000"
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "invalid server port") {
			t.Fatalf("expected error for port > 65535, got: %v", err)
		}
	})

	t.Run("missing db host fails", func(t *testing.T) {
		cfg := validConfig()
		cfg.DB.Host = ""
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "database host") {
			t.Fatalf("expected error for empty db host, got: %v", err)
		}
	})

	t.Run("missing db name fails", func(t *testing.T) {
		cfg := validConfig()
		cfg.DB.DBName = ""
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "database name") {
			t.Fatalf("expected error for empty db name, got: %v", err)
		}
	})

	t.Run("short jwt secret fails", func(t *testing.T) {
		cfg := validConfig()
		cfg.JWT.Secret = "short-secret"
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "at least 32 characters") {
			t.Fatalf("expected error for short secret, got: %v", err)
		}
	})

	t.Run("invalid jwt expiry fails", func(t *testing.T) {
		cfg := validConfig()
		cfg.JWT.ExpirySeconds = 0
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "expiry seconds") {
			t.Fatalf("expected error for zero expiry, got: %v", err)
		}
	})

	t.Run("negative cors max age fails", func(t *testing.T) {
		cfg := validConfig()
		cfg.CORS.MaxAgeSeconds = -1
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "cors max age") {
			t.Fatalf("expected error for negative cors max age, got: %v", err)
		}
	})
}

func TestConfig_Load_TrimSpace(t *testing.T) {
	t.Setenv("PORT", "  9090  ")
	t.Setenv("JWT_EXPIRY_SECONDS", "  3600  ")
	t.Setenv("DB_AUTO_MIGRATE", "  false  ")
	t.Setenv("CORS_ALLOWED_ORIGINS", "  http://localhost:3000/ , http://localhost:5173/  ")

	cfg := config.Load()
	if cfg.Server.Port != "9090" {
		t.Errorf("expected trimmed port '9090', got %q", cfg.Server.Port)
	}
	if cfg.JWT.ExpirySeconds != 3600 {
		t.Errorf("expected trimmed expiry seconds 3600, got %d", cfg.JWT.ExpirySeconds)
	}
	if cfg.DB.AutoMigrate {
		t.Error("expected trimmed auto migrate to be false")
	}

	// Verify trailing slashes were stripped
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d", len(cfg.CORS.AllowedOrigins))
	}
	if cfg.CORS.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected trailing slash stripped 'http://localhost:3000', got %q", cfg.CORS.AllowedOrigins[0])
	}
	if cfg.CORS.AllowedOrigins[1] != "http://localhost:5173" {
		t.Errorf("expected trailing slash stripped 'http://localhost:5173', got %q", cfg.CORS.AllowedOrigins[1])
	}
}

func TestConfig_EmptyDBPassword(t *testing.T) {
	t.Setenv("DB_PASSWORD", "")
	cfg := config.Load()
	if cfg.DB.Password != "" {
		t.Errorf("expected explicitly empty password to remain empty string, got %q", cfg.DB.Password)
	}
}
