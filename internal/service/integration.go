package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntegrationService handles integration management business logic.
type IntegrationService struct {
	connRepo     *repository.IntegrationConnectionRepository
	oauthSvc     *StripeOAuthService
	orchestrator *SyncOrchestratorService
}

// NewIntegrationService creates a new IntegrationService.
func NewIntegrationService(
	connRepo *repository.IntegrationConnectionRepository,
	oauthSvc *StripeOAuthService,
	orchestrator *SyncOrchestratorService,
) *IntegrationService {
	return &IntegrationService{
		connRepo:     connRepo,
		oauthSvc:     oauthSvc,
		orchestrator: orchestrator,
	}
}

// IntegrationSummary holds summary info for an integration connection.
type IntegrationSummary struct {
	Provider      string     `json:"provider"`
	Status        string     `json:"status"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
	LastSyncError string     `json:"last_sync_error,omitempty"`
	CustomerCount int        `json:"customer_count"`
	ConnectedAt   time.Time  `json:"connected_at"`
}

// IntegrationStatus holds detailed status for an integration.
type IntegrationStatus struct {
	IntegrationSummary
	ExternalAccountID string   `json:"external_account_id"`
	Scopes            []string `json:"scopes"`
}

// List returns all integration connections for an org.
func (s *IntegrationService) List(ctx context.Context, orgID uuid.UUID) ([]IntegrationSummary, error) {
	conns, err := s.connRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}

	summaries := make([]IntegrationSummary, len(conns))
	for i, conn := range conns {
		count, err := s.connRepo.GetCustomerCountBySource(ctx, orgID, conn.Provider)
		if err != nil {
			return nil, fmt.Errorf("get customer count: %w", err)
		}
		summaries[i] = IntegrationSummary{
			Provider:      conn.Provider,
			Status:        conn.Status,
			LastSyncAt:    conn.LastSyncAt,
			LastSyncError: conn.LastSyncError,
			CustomerCount: count,
			ConnectedAt:   conn.CreatedAt,
		}
	}

	return summaries, nil
}

// GetStatus returns detailed status for a specific integration provider.
func (s *IntegrationService) GetStatus(ctx context.Context, orgID uuid.UUID, provider string) (*IntegrationStatus, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, provider)
	if err != nil {
		return nil, fmt.Errorf("get integration status: %w", err)
	}
	if conn == nil {
		return nil, &NotFoundError{Resource: "integration", Message: fmt.Sprintf("no %s integration found", provider)}
	}

	count, err := s.connRepo.GetCustomerCountBySource(ctx, orgID, provider)
	if err != nil {
		return nil, fmt.Errorf("get customer count: %w", err)
	}

	return &IntegrationStatus{
		IntegrationSummary: IntegrationSummary{
			Provider:      conn.Provider,
			Status:        conn.Status,
			LastSyncAt:    conn.LastSyncAt,
			LastSyncError: conn.LastSyncError,
			CustomerCount: count,
			ConnectedAt:   conn.CreatedAt,
		},
		ExternalAccountID: conn.ExternalAccountID,
		Scopes:            conn.Scopes,
	}, nil
}

// TriggerSync triggers a sync for a specific integration provider.
func (s *IntegrationService) TriggerSync(ctx context.Context, orgID uuid.UUID, provider string) error {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, provider)
	if err != nil {
		return fmt.Errorf("get integration: %w", err)
	}
	if conn == nil {
		return &NotFoundError{Resource: "integration", Message: fmt.Sprintf("no %s integration found", provider)}
	}
	if conn.Status != "active" {
		return &ValidationError{Field: "status", Message: "integration is not active"}
	}

	// Fire async sync
	go s.orchestrator.RunFullSync(context.Background(), orgID)

	return nil
}

// Disconnect removes an integration connection.
func (s *IntegrationService) Disconnect(ctx context.Context, orgID uuid.UUID, provider string) error {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, provider)
	if err != nil {
		return fmt.Errorf("get integration: %w", err)
	}
	if conn == nil {
		return &NotFoundError{Resource: "integration", Message: fmt.Sprintf("no %s integration found", provider)}
	}

	if err := s.connRepo.Delete(ctx, orgID, provider); err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}

	return nil
}
