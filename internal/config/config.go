package config

import "os"

// Config holds application configuration.
type Config struct {
	Port        string
	DatabaseURL string
	Environment string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://pulsescore:pulsescore@localhost:5432/pulsescore_dev?sslmode=disable"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
