package database

import (
	"testing"
	"time"
)

func TestDefaultPoolConfig(t *testing.T) {
	url := "postgres://user:pass@localhost:5432/testdb"
	cfg := DefaultPoolConfig(url)

	if cfg.URL != url {
		t.Errorf("expected URL %q, got %q", url, cfg.URL)
	}
	if cfg.MaxConns != 10 {
		t.Errorf("expected MaxConns 10, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 2 {
		t.Errorf("expected MinConns 2, got %d", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Hour {
		t.Errorf("expected MaxConnLifetime 1h, got %v", cfg.MaxConnLifetime)
	}
	if cfg.HealthCheckPeriod != 30*time.Second {
		t.Errorf("expected HealthCheckPeriod 30s, got %v", cfg.HealthCheckPeriod)
	}
}
