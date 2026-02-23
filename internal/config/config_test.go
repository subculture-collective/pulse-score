package config

import (
	"os"
	"testing"
	"time"
)

func clearEnv() {
	for _, key := range []string{
		"PORT", "HOST", "ENVIRONMENT", "DATABASE_URL",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS",
		"DB_MAX_CONN_LIFETIME", "DB_HEALTH_CHECK_SEC",
		"READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT",
		"CORS_ALLOWED_ORIGINS", "RATE_LIMIT_RPM",
	} {
		os.Unsetenv(key)
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv()

	cfg := Load()

	if cfg.Server.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Server.Environment != "development" {
		t.Errorf("expected default environment development, got %s", cfg.Server.Environment)
	}
	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("expected read timeout 5s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 10*time.Second {
		t.Errorf("expected write timeout 10s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("expected max open conns 25, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("expected max idle conns 5, got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.MaxConnLifetime != 3600 {
		t.Errorf("expected max conn lifetime 3600, got %d", cfg.Database.MaxConnLifetime)
	}
	if cfg.Database.HealthCheckSec != 30 {
		t.Errorf("expected health check sec 30, got %d", cfg.Database.HealthCheckSec)
	}
	if cfg.Rate.RequestsPerMinute != 100 {
		t.Errorf("expected rate limit 100, got %d", cfg.Rate.RequestsPerMinute)
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("expected default CORS origin, got %v", cfg.CORS.AllowedOrigins)
	}
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv()
	os.Setenv("PORT", "3000")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com, https://app.example.com")
	os.Setenv("RATE_LIMIT_RPM", "200")
	defer clearEnv()

	cfg := Load()

	if cfg.Server.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Server.Port)
	}
	if cfg.Server.Environment != "production" {
		t.Errorf("expected environment production, got %s", cfg.Server.Environment)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Errorf("expected 2 CORS origins, got %d", len(cfg.CORS.AllowedOrigins))
	}
	if cfg.Rate.RequestsPerMinute != 200 {
		t.Errorf("expected rate limit 200, got %d", cfg.Rate.RequestsPerMinute)
	}
}

func TestValidateProduction(t *testing.T) {
	clearEnv()
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("DATABASE_URL", "")
	defer clearEnv()

	cfg := Load()
	// In production with empty DATABASE_URL env var, it will use fallback
	// Explicitly clear it to test validation
	cfg.Database.URL = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing DATABASE_URL in production")
	}
}

func TestValidateDevelopment(t *testing.T) {
	clearEnv()
	cfg := Load()

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no validation error in development, got %v", err)
	}
}

func TestIsProd(t *testing.T) {
	clearEnv()
	cfg := Load()
	if cfg.IsProd() {
		t.Error("expected IsProd false in development")
	}

	os.Setenv("ENVIRONMENT", "production")
	defer clearEnv()
	cfg = Load()
	if !cfg.IsProd() {
		t.Error("expected IsProd true in production")
	}
}
