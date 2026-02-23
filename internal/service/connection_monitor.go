package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/stripe/stripe-go/v81"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"

	"github.com/onnwee/pulse-score/internal/repository"
)

// ConnectionMonitorService periodically checks the health of Stripe connections.
type ConnectionMonitorService struct {
	connRepo *repository.IntegrationConnectionRepository
	oauthSvc *StripeOAuthService
	interval time.Duration
}

// NewConnectionMonitorService creates a new ConnectionMonitorService.
func NewConnectionMonitorService(
	connRepo *repository.IntegrationConnectionRepository,
	oauthSvc *StripeOAuthService,
	intervalMinutes int,
) *ConnectionMonitorService {
	return &ConnectionMonitorService{
		connRepo: connRepo,
		oauthSvc: oauthSvc,
		interval: time.Duration(intervalMinutes) * time.Minute,
	}
}

// Start begins the periodic connection health check. Cancel the context to stop.
func (s *ConnectionMonitorService) Start(ctx context.Context) {
	slog.Info("connection monitor started", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("connection monitor stopped")
			return
		case <-ticker.C:
			s.checkAll(ctx)
		}
	}
}

func (s *ConnectionMonitorService) checkAll(ctx context.Context) {
	conns, err := s.connRepo.ListActiveByProvider(ctx, "stripe")
	if err != nil {
		slog.Error("monitor: failed to list connections", "error", err)
		return
	}

	for _, conn := range conns {
		if err := s.checkConnection(ctx, conn); err != nil {
			slog.Error("monitor: connection check failed",
				"org_id", conn.OrgID,
				"error", err,
			)
		}
	}
}

func (s *ConnectionMonitorService) checkConnection(ctx context.Context, conn *repository.IntegrationConnection) error {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, conn.OrgID)
	if err != nil {
		// Token decrypt/retrieval failed — mark as error
		if err := s.connRepo.UpdateSyncStatus(ctx, conn.OrgID, "stripe", "error", nil); err != nil {
			slog.Error("monitor: failed to update status", "error", err)
		}
		return err
	}

	// Do a lightweight API call to verify the token works
	client := stripecustomer.Client{B: stripe.GetBackend(stripe.APIBackend), Key: accessToken}
	params := &stripe.CustomerListParams{}
	params.Limit = stripe.Int64(1)
	iter := client.List(params)

	// Try to iterate once
	if iter.Next() {
		// Connection is healthy
		return nil
	}

	if iterErr := iter.Err(); iterErr != nil {
		slog.Warn("monitor: Stripe API call failed",
			"org_id", conn.OrgID,
			"error", iterErr,
		)

		if err := s.connRepo.UpdateErrorCount(ctx, conn.OrgID, "stripe", iterErr.Error()); err != nil {
			slog.Error("monitor: failed to update error count", "error", err)
		}

		// Check if we've hit too many consecutive errors
		updatedConn, err := s.connRepo.GetByOrgAndProvider(ctx, conn.OrgID, "stripe")
		if err == nil && updatedConn != nil {
			errorCount := 0
			if v, ok := updatedConn.Metadata["error_count"]; ok {
				if n, ok := v.(float64); ok {
					errorCount = int(n)
				}
			}

			// Disable after 5 consecutive failures
			if errorCount >= 5 {
				slog.Error("monitor: disabling connection after too many failures",
					"org_id", conn.OrgID,
					"error_count", errorCount,
				)
				if err := s.connRepo.UpdateSyncStatus(ctx, conn.OrgID, "stripe", "disconnected", nil); err != nil {
					slog.Error("monitor: failed to disable connection", "error", err)
				}
			}
		}

		return iterErr
	}

	// No error, just no customers — that's fine
	return nil
}
