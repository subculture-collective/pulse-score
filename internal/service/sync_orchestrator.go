package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// SyncOrchestratorService orchestrates the full sync pipeline:
// customers → subscriptions → payments → MRR calculation.
type SyncOrchestratorService struct {
	connRepo  *repository.IntegrationConnectionRepository
	syncSvc   *StripeSyncService
	mrrSvc    *MRRService
}

// NewSyncOrchestratorService creates a new SyncOrchestratorService.
func NewSyncOrchestratorService(
	connRepo *repository.IntegrationConnectionRepository,
	syncSvc *StripeSyncService,
	mrrSvc *MRRService,
) *SyncOrchestratorService {
	return &SyncOrchestratorService{
		connRepo: connRepo,
		syncSvc:  syncSvc,
		mrrSvc:   mrrSvc,
	}
}

// SyncResult contains the results of a full sync.
type SyncResult struct {
	Customers     *SyncProgress `json:"customers"`
	Subscriptions *SyncProgress `json:"subscriptions"`
	Payments      *SyncProgress `json:"payments"`
	MRRUpdated    int           `json:"mrr_updated"`
	Duration      string        `json:"duration"`
	Error         string        `json:"error,omitempty"`
}

// RunFullSync runs the complete sync pipeline for an org.
func (s *SyncOrchestratorService) RunFullSync(ctx context.Context, orgID uuid.UUID) *SyncResult {
	start := time.Now()
	result := &SyncResult{}

	// Mark sync in progress
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "stripe", "syncing", nil); err != nil {
		slog.Error("failed to update sync status", "error", err)
	}

	// Step 1: Sync customers
	custProgress, err := s.syncSvc.SyncCustomers(ctx, orgID)
	result.Customers = custProgress
	if err != nil {
		result.Error = fmt.Sprintf("customer sync failed: %v", err)
		s.markSyncError(ctx, orgID, result.Error)
		result.Duration = time.Since(start).String()
		return result
	}

	// Step 2: Sync subscriptions
	subProgress, err := s.syncSvc.SyncSubscriptions(ctx, orgID)
	result.Subscriptions = subProgress
	if err != nil {
		result.Error = fmt.Sprintf("subscription sync failed: %v", err)
		s.markSyncError(ctx, orgID, result.Error)
		result.Duration = time.Since(start).String()
		return result
	}

	// Step 3: Sync payments
	payProgress, err := s.syncSvc.SyncPayments(ctx, orgID)
	result.Payments = payProgress
	if err != nil {
		result.Error = fmt.Sprintf("payment sync failed: %v", err)
		s.markSyncError(ctx, orgID, result.Error)
		result.Duration = time.Since(start).String()
		return result
	}

	// Step 4: Calculate MRR for all customers
	if err := s.mrrSvc.CalculateForOrg(ctx, orgID); err != nil {
		slog.Error("MRR calculation failed during full sync", "org_id", orgID, "error", err)
	}

	// Mark sync complete
	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "stripe", "active", &now); err != nil {
		slog.Error("failed to update sync status", "error", err)
	}

	result.Duration = time.Since(start).String()

	slog.Info("full sync complete",
		"org_id", orgID,
		"duration", result.Duration,
		"customers", result.Customers.Current,
		"subscriptions", result.Subscriptions.Current,
		"payments", result.Payments.Current,
	)

	return result
}

// RunIncrementalSync runs an incremental sync since the last sync time.
func (s *SyncOrchestratorService) RunIncrementalSync(ctx context.Context, orgID uuid.UUID, since time.Time) *SyncResult {
	start := time.Now()
	result := &SyncResult{}

	// Step 1: Incremental customer sync
	custProgress, err := s.syncSvc.SyncCustomersSince(ctx, orgID, since)
	result.Customers = custProgress
	if err != nil {
		result.Error = fmt.Sprintf("incremental customer sync failed: %v", err)
		s.markSyncError(ctx, orgID, result.Error)
		result.Duration = time.Since(start).String()
		return result
	}

	// Step 2: Full subscription sync (Stripe API doesn't support created filter well for subs)
	subProgress, err := s.syncSvc.SyncSubscriptions(ctx, orgID)
	result.Subscriptions = subProgress
	if err != nil {
		result.Error = fmt.Sprintf("subscription sync failed: %v", err)
		s.markSyncError(ctx, orgID, result.Error)
		result.Duration = time.Since(start).String()
		return result
	}

	// Step 3: Incremental payment sync
	payProgress, err := s.syncSvc.SyncPaymentsSince(ctx, orgID, since)
	result.Payments = payProgress
	if err != nil {
		result.Error = fmt.Sprintf("incremental payment sync failed: %v", err)
		s.markSyncError(ctx, orgID, result.Error)
		result.Duration = time.Since(start).String()
		return result
	}

	// Step 4: Recalculate MRR
	if err := s.mrrSvc.CalculateForOrg(ctx, orgID); err != nil {
		slog.Error("MRR calculation failed during incremental sync", "org_id", orgID, "error", err)
	}

	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "stripe", "active", &now); err != nil {
		slog.Error("failed to update sync status", "error", err)
	}

	result.Duration = time.Since(start).String()
	return result
}

func (s *SyncOrchestratorService) markSyncError(ctx context.Context, orgID uuid.UUID, errMsg string) {
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "stripe", "error", nil); err != nil {
		slog.Error("failed to update sync error status", "error", err)
	}
	if err := s.connRepo.UpdateErrorCount(ctx, orgID, "stripe", errMsg); err != nil {
		slog.Error("failed to update error count", "error", err)
	}
}
