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
	"github.com/onnwee/pulse-score/internal/service/scoring"
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

	// API documentation (no auth)
	docsHandler := handler.NewDocsHandler("docs/openapi.yaml")
	r.Get("/api/docs/openapi.yaml", docsHandler.ServeSpec)
	r.Get("/api/docs", docsHandler.ServeUI)
	r.Get("/api/docs/*", docsHandler.ServeUI)

	// API v1 route group
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"pong"}`))
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

			// Stripe integration repositories
			connRepo := repository.NewIntegrationConnectionRepository(pool.P)
			customerRepo := repository.NewCustomerRepository(pool.P)
			subRepo := repository.NewStripeSubscriptionRepository(pool.P)
			paymentRepo := repository.NewStripePaymentRepository(pool.P)
			eventRepo := repository.NewCustomerEventRepository(pool.P)

			// Stripe services
			stripeOAuthSvc := service.NewStripeOAuthService(service.StripeOAuthConfig{
				ClientID:         cfg.Stripe.ClientID,
				SecretKey:        cfg.Stripe.SecretKey,
				OAuthRedirectURL: cfg.Stripe.OAuthRedirectURL,
				EncryptionKey:    cfg.Stripe.EncryptionKey,
			}, connRepo)

			stripeSyncSvc := service.NewStripeSyncService(
				customerRepo, subRepo, paymentRepo, eventRepo,
				stripeOAuthSvc, cfg.Stripe.PaymentSyncDays,
			)

			mrrSvc := service.NewMRRService(customerRepo, subRepo, eventRepo)
			paymentHealthSvc := service.NewPaymentHealthService(paymentRepo, eventRepo, customerRepo)
			paymentRecencySvc := service.NewPaymentRecencyService(paymentRepo, subRepo)

			syncOrchestrator := service.NewSyncOrchestratorService(connRepo, stripeSyncSvc, mrrSvc)

			stripeWebhookSvc := service.NewStripeWebhookService(
				cfg.Stripe.WebhookSecret,
				connRepo, customerRepo, subRepo, paymentRepo, eventRepo,
				mrrSvc, paymentHealthSvc,
			)

			// Health scoring engine
			scoringConfigRepo := repository.NewScoringConfigRepository(pool.P)
			healthScoreRepo := repository.NewHealthScoreRepository(pool.P)

			paymentRecencyFactor := scoring.NewPaymentRecencyFactor(paymentRecencySvc)
			mrrTrendFactor := scoring.NewMRRTrendFactor(customerRepo, eventRepo)
			failedPaymentsFactor := scoring.NewFailedPaymentsFactor(paymentHealthSvc, paymentRepo)
			supportTicketsFactor := scoring.NewSupportTicketsFactor(eventRepo)
			engagementFactor := scoring.NewEngagementFactor(eventRepo)

			scoreAggregator := scoring.NewScoreAggregator(
				[]scoring.ScoreFactor{
					paymentRecencyFactor,
					mrrTrendFactor,
					failedPaymentsFactor,
					supportTicketsFactor,
					engagementFactor,
				},
				scoringConfigRepo,
			)

			changeDetector := scoring.NewChangeDetector(eventRepo, cfg.Scoring.ChangeDelta)
			riskCategorizer := scoring.NewRiskCategorizer(healthScoreRepo)

			scoreScheduler := scoring.NewScoreScheduler(
				scoreAggregator, healthScoreRepo, customerRepo, connRepo, changeDetector,
				time.Duration(cfg.Scoring.RecalcIntervalMin)*time.Minute,
				cfg.Scoring.Workers,
			)

			scoringConfigSvc := scoring.NewConfigService(scoringConfigRepo, scoreScheduler)

			// HubSpot repositories
			hubspotContactRepo := repository.NewHubSpotContactRepository(pool.P)
			hubspotDealRepo := repository.NewHubSpotDealRepository(pool.P)
			hubspotCompanyRepo := repository.NewHubSpotCompanyRepository(pool.P)

			// HubSpot services
			hubspotOAuthSvc := service.NewHubSpotOAuthService(service.HubSpotOAuthConfig{
				ClientID:         cfg.HubSpot.ClientID,
				ClientSecret:     cfg.HubSpot.ClientSecret,
				OAuthRedirectURL: cfg.HubSpot.OAuthRedirectURL,
				EncryptionKey:    cfg.HubSpot.EncryptionKey,
			}, connRepo)

			hubspotClient := service.NewHubSpotClient()

			hubspotSyncSvc := service.NewHubSpotSyncService(
				hubspotOAuthSvc, hubspotClient,
				hubspotContactRepo, hubspotDealRepo, hubspotCompanyRepo,
				customerRepo, eventRepo,
			)

			customerMergeSvc := service.NewCustomerMergeService(customerRepo, hubspotContactRepo)

			hubspotWebhookSvc := service.NewHubSpotWebhookService(
				cfg.HubSpot.ClientSecret,
				hubspotSyncSvc, customerMergeSvc,
				connRepo, hubspotContactRepo, hubspotDealRepo, hubspotCompanyRepo, eventRepo,
			)

			hubspotOrchestrator := service.NewHubSpotSyncOrchestratorService(
				connRepo, hubspotSyncSvc, customerMergeSvc,
			)

			// Start background services
			bgCtx, bgCancel := context.WithCancel(context.Background())
			defer bgCancel()

			if cfg.Stripe.SyncIntervalMin > 0 {
				syncScheduler := service.NewSyncSchedulerService(connRepo, syncOrchestrator, hubspotOrchestrator, cfg.Stripe.SyncIntervalMin)
				go syncScheduler.Start(bgCtx)
			}

			if cfg.Scoring.RecalcIntervalMin > 0 {
				go scoreScheduler.Start(bgCtx)
			}

			connMonitor := service.NewConnectionMonitorService(connRepo, stripeOAuthSvc, hubspotOAuthSvc, hubspotClient, 60)
			go connMonitor.Start(bgCtx)

			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
			r.Post("/auth/password-reset/request", authHandler.RequestPasswordReset)
			r.Post("/auth/password-reset/complete", authHandler.CompletePasswordReset)

			// Invitation acceptance (public — no auth required)
			r.Post("/invitations/accept", invitationHandler.Accept)

			// Stripe webhook (public — verified by signature)
			webhookHandler := handler.NewWebhookStripeHandler(stripeWebhookSvc)
			r.Post("/webhooks/stripe", webhookHandler.HandleWebhook)

			// HubSpot webhook (public — verified by signature)
			hubspotWebhookHandler := handler.NewWebhookHubSpotHandler(hubspotWebhookSvc)
			r.Post("/webhooks/hubspot", hubspotWebhookHandler.HandleWebhook)

			// Protected routes (JWT required)
			r.Group(func(r chi.Router) {
				r.Use(middleware.JWTAuth(jwtMgr))
				r.Use(middleware.TenantIsolation(orgRepo))

				// Organization routes
				orgSvc := service.NewOrganizationService(pool.P, orgRepo)
				orgHandler := handler.NewOrganizationHandler(orgSvc)
				r.Post("/organizations", orgHandler.Create)
				r.Get("/organizations/current", orgHandler.GetCurrent)
				r.Patch("/organizations/current", orgHandler.UpdateCurrent)

				// User profile routes
				userSvc := service.NewUserService(userRepo, orgRepo)
				userHandler := handler.NewUserHandler(userSvc)
				r.Get("/users/me", userHandler.GetProfile)
				r.Patch("/users/me", userHandler.UpdateProfile)

				// Customer routes
				customerSvc := service.NewCustomerService(customerRepo, healthScoreRepo, subRepo, eventRepo)
				customerHandler := handler.NewCustomerHandler(customerSvc)
				r.Get("/customers", customerHandler.List)
				r.Get("/customers/{id}", customerHandler.GetDetail)
				r.Get("/customers/{id}/events", customerHandler.ListEvents)

				// Dashboard routes
				dashboardSvc := service.NewDashboardService(customerRepo, healthScoreRepo)
				dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
				r.Get("/dashboard/summary", dashboardHandler.GetSummary)
				r.Get("/dashboard/score-distribution", dashboardHandler.GetScoreDistribution)

				// Integration management routes (admin+ required)
				integrationSvc := service.NewIntegrationService(connRepo, stripeOAuthSvc, syncOrchestrator)
				integrationHandler := handler.NewIntegrationHandler(integrationSvc)
				r.Route("/integrations", func(r chi.Router) {
					r.Get("/", integrationHandler.List)
					r.Route("/{provider}", func(r chi.Router) {
						r.Use(middleware.RequireRole("admin"))
						r.Get("/status", integrationHandler.GetStatus)
						r.Post("/sync", integrationHandler.TriggerSync)
						r.Delete("/", integrationHandler.Disconnect)
					})
				})

				// Member management routes (admin+ required)
				memberSvc := service.NewMemberService(orgRepo)
				memberHandler := handler.NewMemberHandler(memberSvc)
				r.Get("/members", memberHandler.List)
				r.Route("/members/{id}", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Patch("/role", memberHandler.UpdateRole)
					r.Delete("/", memberHandler.Remove)
				})

				// Invitation routes (admin+ required)
				r.Route("/invitations", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Post("/", invitationHandler.Create)
					r.Get("/", invitationHandler.List)
					r.Delete("/{id}", invitationHandler.Revoke)
				})

				// Alert rule routes (admin+ required)
				alertRuleRepo := repository.NewAlertRuleRepository(pool.P)
				alertRuleSvc := service.NewAlertRuleService(alertRuleRepo)
				alertRuleHandler := handler.NewAlertRuleHandler(alertRuleSvc)
				r.Route("/alerts/rules", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Get("/", alertRuleHandler.List)
					r.Post("/", alertRuleHandler.Create)
					r.Get("/{id}", alertRuleHandler.Get)
					r.Patch("/{id}", alertRuleHandler.Update)
					r.Delete("/{id}", alertRuleHandler.Delete)
				})

				// Stripe integration routes (admin+ required)
				stripeHandler := handler.NewIntegrationStripeHandler(stripeOAuthSvc, syncOrchestrator)
				r.Route("/integrations/stripe", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Get("/connect", stripeHandler.Connect)
					r.Get("/callback", stripeHandler.Callback)
					r.Get("/status", stripeHandler.Status)
					r.Delete("/", stripeHandler.Disconnect)
					r.Post("/sync", stripeHandler.TriggerSync)
				})

				// HubSpot integration routes (admin+ required)
				hubspotHandler := handler.NewIntegrationHubSpotHandler(hubspotOAuthSvc, hubspotOrchestrator)
				r.Route("/integrations/hubspot", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Get("/connect", hubspotHandler.Connect)
					r.Get("/callback", hubspotHandler.Callback)
					r.Get("/status", hubspotHandler.Status)
					r.Delete("/", hubspotHandler.Disconnect)
					r.Post("/sync", hubspotHandler.TriggerSync)
				})

				// Health scoring routes
				scoringHandler := handler.NewScoringHandler(scoringConfigSvc, riskCategorizer, scoreScheduler)
				r.Route("/scoring", func(r chi.Router) {
					r.Get("/risk-distribution", scoringHandler.GetRiskDistribution)
					r.Get("/histogram", scoringHandler.GetScoreHistogram)
					r.Post("/customers/{id}/recalculate", scoringHandler.RecalculateCustomer)
					r.Route("/config", func(r chi.Router) {
						r.Use(middleware.RequireRole("admin"))
						r.Get("/", scoringHandler.GetConfig)
						r.Put("/", scoringHandler.UpdateConfig)
					})
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
