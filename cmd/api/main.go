package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/config"
	"github.com/onnwee/pulse-score/internal/database"
	"github.com/onnwee/pulse-score/internal/handler"
	"github.com/onnwee/pulse-score/internal/middleware"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

func main() {
	// Structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Database connection (pgxpool)
	var pool *database.Pool
	if cfg.Database.URL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		dbPool, err := database.NewPool(ctx, database.PoolConfig{
			URL:               cfg.Database.URL,
			MaxConns:          int32(cfg.Database.MaxOpenConns),
			MinConns:          int32(cfg.Database.MaxIdleConns),
			MaxConnLifetime:   time.Duration(cfg.Database.MaxConnLifetime) * time.Second,
			HealthCheckPeriod: time.Duration(cfg.Database.HealthCheckSec) * time.Second,
		})
		if err != nil {
			slog.Warn("database not reachable at startup", "error", err)
		} else {
			pool = &database.Pool{P: dbPool}
			defer dbPool.Close()
		}
	}

	// JWT manager
	jwtMgr := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Organization-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httprate.LimitByIP(cfg.Rate.RequestsPerMinute, time.Minute))

	// Health checks (no auth required)
	health := handler.NewHealthHandler(pool)
	r.Get("/healthz", health.Liveness)
	r.Get("/readyz", health.Readiness)

	// API v1 route group
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"pong"}`))
		})

		// Auth routes (public)
		if pool != nil {
			userRepo := repository.NewUserRepository(pool.P)
			orgRepo := repository.NewOrganizationRepository(pool.P)
			refreshTokenRepo := repository.NewRefreshTokenRepository(pool.P)
			invitationRepo := repository.NewInvitationRepository(pool.P)
			passwordResetRepo := repository.NewPasswordResetRepository(pool.P)

			emailSvc := service.NewSendGridEmailService(service.SendGridConfig{
				APIKey:      cfg.SendGrid.APIKey,
				FromEmail:   cfg.SendGrid.FromEmail,
				FrontendURL: cfg.SendGrid.FrontendURL,
				DevMode:     !cfg.IsProd(),
			})

			authSvc := service.NewAuthService(pool.P, userRepo, orgRepo, refreshTokenRepo, passwordResetRepo, jwtMgr, cfg.JWT.RefreshTTL, emailSvc)
			authHandler := handler.NewAuthHandler(authSvc)

			invitationSvc := service.NewInvitationService(pool.P, invitationRepo, orgRepo, userRepo, emailSvc, jwtMgr)
			invitationHandler := handler.NewInvitationHandler(invitationSvc)

			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
			r.Post("/auth/password-reset/request", authHandler.RequestPasswordReset)
			r.Post("/auth/password-reset/complete", authHandler.CompletePasswordReset)

			// Invitation acceptance (public — no auth required)
			r.Post("/invitations/accept", invitationHandler.Accept)

			// Protected routes (JWT required)
			r.Group(func(r chi.Router) {
				r.Use(middleware.JWTAuth(jwtMgr))
				r.Use(middleware.TenantIsolation(orgRepo))

				// Organization routes
				orgSvc := service.NewOrganizationService(pool.P, orgRepo)
				orgHandler := handler.NewOrganizationHandler(orgSvc)
				r.Post("/organizations", orgHandler.Create)

				// User profile routes
				userSvc := service.NewUserService(userRepo, orgRepo)
				userHandler := handler.NewUserHandler(userSvc)
				r.Get("/users/me", userHandler.GetProfile)
				r.Patch("/users/me", userHandler.UpdateProfile)

				// Invitation routes (admin+ required)
				r.Route("/invitations", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Post("/", invitationHandler.Create)
					r.Get("/", invitationHandler.List)
					r.Delete("/{id}", invitationHandler.Revoke)
				})
			})
		}
	})

	// Server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("starting PulseScore API", "addr", addr, "env", cfg.Server.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
