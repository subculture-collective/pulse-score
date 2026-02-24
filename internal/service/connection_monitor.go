package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/stripe/stripe-go/v81"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"

	"github.com/onnwee/pulse-score/internal/repository"
)

// ConnectionMonitorService periodically checks the health of integration connections.
type ConnectionMonitorService struct {
	connRepo       *repository.IntegrationConnectionRepository
	oauthSvc       *StripeOAuthService
	hubspotOAuth   *HubSpotOAuthService
	hubspotClient  *HubSpotClient
	intercomOAuth  *IntercomOAuthService
	intercomClient *IntercomClient
	interval       time.Duration
}

// NewConnectionMonitorService creates a new ConnectionMonitorService.
func NewConnectionMonitorService(
	connRepo *repository.IntegrationConnectionRepository,
	oauthSvc *StripeOAuthService,
	hubspotOAuth *HubSpotOAuthService,
	hubspotClient *HubSpotClient,
	intercomOAuth *IntercomOAuthService,
	intercomClient *IntercomClient,
	intervalMinutes int,
) *ConnectionMonitorService {
	return &ConnectionMonitorService{
		connRepo:       connRepo,
		oauthSvc:       oauthSvc,
		hubspotOAuth:   hubspotOAuth,
		hubspotClient:  hubspotClient,
		intercomOAuth:  intercomOAuth,
		intercomClient: intercomClient,
		interval:       time.Duration(intervalMinutes) * time.Minute,
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
	// Check Stripe connections
	conns, err := s.connRepo.ListActiveByProvider(ctx, "stripe")
	if err != nil {
		slog.Error("monitor: failed to list stripe connections", "error", err)
	} else {
		for _, conn := range conns {
			if err := s.checkConnection(ctx, conn); err != nil {
				slog.Error("monitor: stripe connection check failed",
					"org_id", conn.OrgID,
					"error", err,
				)
			}
		}
	}

	// Check HubSpot connections
	if s.hubspotOAuth != nil && s.hubspotClient != nil {
		hsConns, err := s.connRepo.ListActiveByProvider(ctx, "hubspot")
		if err != nil {
			slog.Error("monitor: failed to list hubspot connections", "error", err)
		} else {
			for _, conn := range hsConns {
				if err := s.checkHubSpotConnection(ctx, conn); err != nil {
					slog.Error("monitor: hubspot connection check failed",
						"org_id", conn.OrgID,
						"error", err,
					)
				}
			}
		}
	}

	// Check Intercom connections
	if s.intercomOAuth != nil && s.intercomClient != nil {
		icConns, err := s.connRepo.ListActiveByProvider(ctx, "intercom")
		if err != nil {
			slog.Error("monitor: failed to list intercom connections", "error", err)
		} else {
			for _, conn := range icConns {
				if err := s.checkIntercomConnection(ctx, conn); err != nil {
					slog.Error("monitor: intercom connection check failed",
						"org_id", conn.OrgID,
						"error", err,
					)
				}
			}
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

func (s *ConnectionMonitorService) checkHubSpotConnection(ctx context.Context, conn *repository.IntegrationConnection) error {
	accessToken, err := s.hubspotOAuth.GetAccessToken(ctx, conn.OrgID)
	if err != nil {
		if err := s.connRepo.UpdateSyncStatus(ctx, conn.OrgID, "hubspot", "error", nil); err != nil {
			slog.Error("monitor: failed to update hubspot status", "error", err)
		}
		return err
	}

	// Lightweight API call: list 1 contact to verify token
	_, err = s.hubspotClient.ListContacts(ctx, accessToken, "")
	if err != nil {
		slog.Warn("monitor: HubSpot API call failed",
			"org_id", conn.OrgID,
			"error", err,
		)

		if err := s.connRepo.UpdateErrorCount(ctx, conn.OrgID, "hubspot", err.Error()); err != nil {
			slog.Error("monitor: failed to update hubspot error count", "error", err)
		}

		updatedConn, lookupErr := s.connRepo.GetByOrgAndProvider(ctx, conn.OrgID, "hubspot")
		if lookupErr == nil && updatedConn != nil {
			errorCount := 0
			if v, ok := updatedConn.Metadata["error_count"]; ok {
				if n, ok := v.(float64); ok {
					errorCount = int(n)
				}
			}
			if errorCount >= 5 {
				slog.Error("monitor: disabling hubspot connection after too many failures",
					"org_id", conn.OrgID,
					"error_count", errorCount,
				)
				if err := s.connRepo.UpdateSyncStatus(ctx, conn.OrgID, "hubspot", "disconnected", nil); err != nil {
					slog.Error("monitor: failed to disable hubspot connection", "error", err)
				}
			}
		}

		return err
	}

	return nil
}

func (s *ConnectionMonitorService) checkIntercomConnection(ctx context.Context, conn *repository.IntegrationConnection) error {
	accessToken, err := s.intercomOAuth.GetAccessToken(ctx, conn.OrgID)
	if err != nil {
		if err := s.connRepo.UpdateSyncStatus(ctx, conn.OrgID, "intercom", "error", nil); err != nil {
			slog.Error("monitor: failed to update intercom status", "error", err)
		}
		return err
	}

	// Lightweight API call: list 1 contact to verify token
	_, err = s.intercomClient.ListContacts(ctx, accessToken, "")
	if err != nil {
		slog.Warn("monitor: Intercom API call failed",
			"org_id", conn.OrgID,
			"error", err,
		)

		if err := s.connRepo.UpdateErrorCount(ctx, conn.OrgID, "intercom", err.Error()); err != nil {
			slog.Error("monitor: failed to update intercom error count", "error", err)
		}

		updatedConn, lookupErr := s.connRepo.GetByOrgAndProvider(ctx, conn.OrgID, "intercom")
		if lookupErr == nil && updatedConn != nil {
			errorCount := 0
			if v, ok := updatedConn.Metadata["error_count"]; ok {
				if n, ok := v.(float64); ok {
					errorCount = int(n)
				}
			}
			if errorCount >= 5 {
				slog.Error("monitor: disabling intercom connection after too many failures",
					"org_id", conn.OrgID,
					"error_count", errorCount,
				)
				if err := s.connRepo.UpdateSyncStatus(ctx, conn.OrgID, "intercom", "disconnected", nil); err != nil {
					slog.Error("monitor: failed to disable intercom connection", "error", err)
				}
			}
		}

		return err
	}

	return nil
}
