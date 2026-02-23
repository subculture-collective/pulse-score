package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	CORS     CORSConfig
	Rate     RateConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         string
	Host         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
}

// CORSConfig holds CORS middleware settings.
type CORSConfig struct {
	AllowedOrigins []string
}

// RateConfig holds rate limiting settings.
type RateConfig struct {
	RequestsPerMinute int
}

// IsProd returns true if the environment is production.
func (c *Config) IsProd() bool {
	return c.Server.Environment == "production"
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("HOST", "0.0.0.0"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  getDuration("READ_TIMEOUT", 5*time.Second),
			WriteTimeout: getDuration("WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDuration("IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			URL:          getEnv("DATABASE_URL", "postgres://pulsescore:pulsescore@localhost:5432/pulsescore_dev?sslmode=disable"),
			MaxOpenConns: getInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getInt("DB_MAX_IDLE_CONNS", 5),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		},
		Rate: RateConfig{
			RequestsPerMinute: getInt("RATE_LIMIT_RPM", 100),
		},
	}
}

// Validate checks required configuration for production.
func (c *Config) Validate() error {
	if c.IsProd() {
		if c.Database.URL == "" {
			return fmt.Errorf("DATABASE_URL is required in production")
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getEnvSlice(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var result []string
	for _, s := range splitAndTrim(v) {
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func splitAndTrim(s string) []string {
	var parts []string
	for _, p := range []byte(s) {
		if p == ',' {
			parts = append(parts, "")
		} else {
			if len(parts) == 0 {
				parts = append(parts, "")
			}
			parts[len(parts)-1] += string(p)
		}
	}
	// Trim spaces
	for i, p := range parts {
		trimmed := ""
		start := 0
		end := len(p) - 1
		for start <= end && p[start] == ' ' {
			start++
		}
		for end >= start && p[end] == ' ' {
			end--
		}
		if start <= end {
			trimmed = p[start : end+1]
		}
		parts[i] = trimmed
	}
	return parts
}
