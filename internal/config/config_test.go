package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Unset any env vars that might be set
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("ENVIRONMENT")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected default environment development, got %s", cfg.Environment)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("ENVIRONMENT", "production")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENVIRONMENT")
	}()

	cfg := Load()

	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected environment production, got %s", cfg.Environment)
	}
}
