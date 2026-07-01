package config

import (
	"fmt"
	"os"
)

// Config holds all configuration values for the catalog service.
type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	SeedFile         string
	Port             string
}

// Load reads configuration from environment variables, falling back to defaults.
func Load() *Config {
	return &Config{
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       getEnv("POSTGRES_DB", "catalog"),
		SeedFile:         getEnv("SEED_FILE", ""),
		Port:             getEnv("PORT", "8081"),
	}
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPassword,
		c.PostgresHost, c.PostgresPort,
		c.PostgresDB,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
