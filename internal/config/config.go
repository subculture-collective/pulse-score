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
	JWT      JWTConfig
	SendGrid SendGridConfig
	Stripe   StripeConfig
	Scoring  ScoringConfig
}

// ScoringConfig holds health score engine settings.
type ScoringConfig struct {
	RecalcIntervalMin int
	Workers           int
	ChangeDelta       float64
}

// StripeConfig holds Stripe OAuth and webhook settings.
type StripeConfig struct {
	ClientID         string
	SecretKey        string
	WebhookSecret    string
	OAuthRedirectURL string
	EncryptionKey    string // 32-byte hex-encoded AES key for token encryption
	SyncIntervalMin  int
	PaymentSyncDays  int
}

// SendGridConfig holds email sending settings.
type SendGridConfig struct {
	APIKey      string
	FromEmail   string
	FrontendURL string
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
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
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	MaxConnLifetime int // seconds
	HealthCheckSec  int // seconds
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
			URL:             getEnv("DATABASE_URL", "postgres://pulsescore:pulsescore@localhost:5434/pulsescore_dev?sslmode=disable"),
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 5),
			MaxConnLifetime: getInt("DB_MAX_CONN_LIFETIME", 3600),
			HealthCheckSec:  getInt("DB_HEALTH_CHECK_SEC", 30),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		},
		Rate: RateConfig{
			RequestsPerMinute: getInt("RATE_LIMIT_RPM", 100),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "dev-secret-change-me-in-production"),
			AccessTTL:  getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		SendGrid: SendGridConfig{
			APIKey:      getEnv("SENDGRID_API_KEY", ""),
			FromEmail:   getEnv("SENDGRID_FROM_EMAIL", "noreply@pulsescore.com"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
		},
		Stripe: StripeConfig{
			ClientID:         getEnv("STRIPE_CLIENT_ID", ""),
			SecretKey:        getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret:    getEnv("STRIPE_WEBHOOK_SECRET", ""),
			OAuthRedirectURL: getEnv("STRIPE_OAUTH_REDIRECT_URL", "http://localhost:8080/api/v1/integrations/stripe/callback"),
			EncryptionKey:    getEnv("STRIPE_ENCRYPTION_KEY", ""),
			SyncIntervalMin:  getInt("STRIPE_SYNC_INTERVAL_MIN", 15),
			PaymentSyncDays:  getInt("STRIPE_PAYMENT_SYNC_DAYS", 90),
		},
		Scoring: ScoringConfig{
			RecalcIntervalMin: getInt("SCORE_RECALC_INTERVAL_MIN", 60),
			Workers:           getInt("SCORE_RECALC_WORKERS", 5),
			ChangeDelta:       float64(getInt("SCORE_CHANGE_DELTA", 10)),
		},
	}
}

// Validate checks required configuration for production.
func (c *Config) Validate() error {
	if c.IsProd() {
		if c.Database.URL == "" {
			return fmt.Errorf("DATABASE_URL is required in production")
		}
		if c.JWT.Secret == "" || c.JWT.Secret == "dev-secret-change-me-in-production" {
			return fmt.Errorf("JWT_SECRET must be set to a secure value in production")
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
