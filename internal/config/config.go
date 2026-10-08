package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig
	DB     DBConfig
	JWT    JWTConfig
	CORS   CORSConfig
}

// CORSConfig holds Cross-Origin Resource Sharing configuration
type CORSConfig struct {
	AllowedOrigins []string
	MaxAgeSeconds  int
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret        string
	ExpirySeconds int
}

// ServerConfig holds HTTP server related configuration
type ServerConfig struct {
	Port string
	Mode string
}

// DBConfig holds PostgreSQL database connection configuration
type DBConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	DBName             string
	SSLMode            string
	AutoMigrate        bool
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
	ConnMaxIdleTimeMin int
}

// DSN returns standard PostgreSQL connection string format (key=value)
func (d *DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

// URL returns standard PostgreSQL connection URL format (RFC 3986 compliant with properly escaped credentials)
func (d *DBConfig) URL() string {
	var host string
	if d.Port != "" {
		host = net.JoinHostPort(d.Host, d.Port)
	} else {
		host = d.Host
	}

	u := url.URL{
		Scheme: "postgres",
		Host:   host,
		Path:   "/" + strings.TrimPrefix(d.DBName, "/"),
	}

	if d.User != "" {
		if d.Password != "" {
			u.User = url.UserPassword(d.User, d.Password)
		} else {
			u.User = url.User(d.User)
		}
	}

	if d.SSLMode != "" {
		u.RawQuery = "sslmode=" + url.QueryEscape(d.SSLMode)
	}

	return u.String()
}

// Validate verifies that required application configuration values are present and valid
func (c *Config) Validate() error {
	// Validate Server Port
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	portNum, err := strconv.Atoi(c.Server.Port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid server port %q: must be between 1 and 65535", c.Server.Port)
	}

	// Validate DB Config
	if c.DB.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if c.DB.DBName == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	// Validate JWT Config
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt secret cannot be empty")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("jwt secret must be at least 32 characters long (got %d)", len(c.JWT.Secret))
	}
	if c.JWT.ExpirySeconds <= 0 {
		return fmt.Errorf("jwt expiry seconds must be greater than 0")
	}

	// Validate CORS Config
	if c.CORS.MaxAgeSeconds < 0 {
		return fmt.Errorf("cors max age seconds cannot be negative")
	}

	return nil
}

// Load loads configuration from environment variables and .env file
func Load() *Config {
	// Load .env file if it exists, otherwise rely on system environment variables
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		DB: DBConfig{
			Host:               getEnv("DB_HOST", "localhost"),
			Port:               getEnv("DB_PORT", "5432"),
			User:               getEnv("DB_USER", "admin"),
			Password:           getEnvAllowEmpty("DB_PASSWORD", "postgres"),
			DBName:             getEnv("DB_NAME", "hospital_middleware_db"),
			SSLMode:            getEnv("DB_SSLMODE", "disable"),
			AutoMigrate:        getEnvBool("DB_AUTO_MIGRATE", true),
			MaxOpenConns:       getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:       getEnvInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetimeMin: getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5),
			ConnMaxIdleTimeMin: getEnvInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 5),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "hospital-middleware-super-secret-key-32bytes"),
			ExpirySeconds: getEnvInt("JWT_EXPIRY_SECONDS", 86400),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseOrigins(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
			MaxAgeSeconds:  getEnvInt("CORS_MAX_AGE_SECONDS", 86400),
		},
	}
}

// parseOrigins parses a comma-separated list of allowed origins and trims trailing slashes.
func parseOrigins(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	var origins []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			trimmed = strings.TrimRight(trimmed, "/")
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}
	return origins
}

// getEnv retrieves environment variable, trims spaces, and returns default value if empty or not set
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.TrimSpace(val)
		if val != "" {
			return val
		}
	}
	return fallback
}

// getEnvAllowEmpty retrieves environment variable, trims spaces, and permits empty string if explicitly set
func getEnvAllowEmpty(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(val)
	}
	return fallback
}

// getEnvBool retrieves environment variable as boolean or returns default value if not set or invalid
func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.TrimSpace(val)
		if val != "" {
			parsed, err := strconv.ParseBool(val)
			if err == nil {
				return parsed
			}
		}
	}
	return fallback
}

// getEnvInt retrieves environment variable as integer or returns default value if not set or invalid
func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.TrimSpace(val)
		if val != "" {
			if parsed, err := strconv.Atoi(val); err == nil {
				return parsed
			}
		}
	}
	return fallback
}
