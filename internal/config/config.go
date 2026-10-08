package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig
	DB     DBConfig
}

// ServerConfig holds HTTP server related configuration
type ServerConfig struct {
	Port string
	Mode string
}

// DBConfig holds PostgreSQL database connection configuration
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode     string
	AutoMigrate bool
}

// DSN returns standard PostgreSQL connection string format (key=value)
func (d *DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

// URL returns standard PostgreSQL connection URL format
func (d *DBConfig) URL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
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
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnv("DB_PORT", "5432"),
			User:        getEnv("DB_USER", "admin"),
			Password:    getEnv("DB_PASSWORD", "postgres"),
			DBName:      getEnv("DB_NAME", "hospital_middleware_db"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
			AutoMigrate: getEnvBool("DB_AUTO_MIGRATE", true),
		},
	}
}

// getEnv retrieves environment variable or returns default value if not set
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

// getEnvBool retrieves environment variable as boolean or returns default value if not set
func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		parsed, err := strconv.ParseBool(val)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
