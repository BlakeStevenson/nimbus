package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	// Database
	DatabaseURL string

	// Server
	Port int
	Host string

	// Authentication
	JWTSecret string

	// Environment
	Environment string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/nimbus?sslmode=disable"),
		Port:        getEnvAsInt("PORT", 8080),
		Host:        getEnv("HOST", "0.0.0.0"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}

	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
