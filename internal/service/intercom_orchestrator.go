package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntercomSyncResult contains the results of a full Intercom sync.
type IntercomSyncResult struct {
	Contacts      *SyncProgress        `json:"contacts"`
	Conversations *SyncProgress        `json:"conversations"`
	Deduplicated  *DeduplicationResult `json:"deduplicated,omitempty"`
	Duration      string               `json:"duration"`
	Errors        []string             `json:"errors,omitempty"`
}

// IntercomSyncOrchestratorService orchestrates the full Intercom sync pipeline.
type IntercomSyncOrchestratorService struct {
	connRepo *repository.IntegrationConnectionRepository
	syncSvc  *IntercomSyncService
	mergeSvc *CustomerMergeService
}

// NewIntercomSyncOrchestratorService creates a new IntercomSyncOrchestratorService.
func NewIntercomSyncOrchestratorService(
	connRepo *repository.IntegrationConnectionRepository,
	syncSvc *IntercomSyncService,
	mergeSvc *CustomerMergeService,
) *IntercomSyncOrchestratorService {
	return &IntercomSyncOrchestratorService{
		connRepo: connRepo,
		syncSvc:  syncSvc,
		mergeSvc: mergeSvc,
	}
}

// RunFullSync runs the complete Intercom sync pipeline for an org.
func (s *IntercomSyncOrchestratorService) RunFullSync(ctx context.Context, orgID uuid.UUID) *IntercomSyncResult {
	start := time.Now()
	result := &IntercomSyncResult{}

	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "intercom", "syncing", nil); err != nil {
		slog.Error("failed to update intercom sync status", "error", err)
	}

	// Step 1: Sync contacts
	contactProgress, err := s.syncSvc.SyncContacts(ctx, orgID)
	result.Contacts = contactProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("contact sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 2: Sync conversations
	convProgress, err := s.syncSvc.SyncConversations(ctx, orgID)
	result.Conversations = convProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("conversation sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 3: Deduplication
	dedupResult, err := s.mergeSvc.DeduplicateCustomers(ctx, orgID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("dedup: %v", err))
	} else {
		result.Deduplicated = dedupResult
	}

	// Mark sync complete
	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "intercom", "active", &now); err != nil {
		slog.Error("failed to update intercom sync status", "error", err)
	}

	result.Duration = time.Since(start).String()

	slog.Info("intercom full sync complete",
		"org_id", orgID,
		"duration", result.Duration,
		"errors", len(result.Errors),
	)

	return result
}

// RunIncrementalSync runs an incremental Intercom sync for records modified since the given time.
func (s *IntercomSyncOrchestratorService) RunIncrementalSync(ctx context.Context, orgID uuid.UUID, since time.Time) *IntercomSyncResult {
	start := time.Now()
	result := &IntercomSyncResult{}

	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "intercom", "syncing", nil); err != nil {
		slog.Error("failed to update intercom sync status", "error", err)
	}

	// Step 1: Sync contacts modified since
	contactProgress, err := s.syncSvc.SyncContactsSince(ctx, orgID, since)
	result.Contacts = contactProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("incremental contact sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 2: Sync conversations modified since
	convProgress, err := s.syncSvc.SyncConversationsSince(ctx, orgID, since)
	result.Conversations = convProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("incremental conversation sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 3: Deduplication for any new records
	dedupResult, err := s.mergeSvc.DeduplicateCustomers(ctx, orgID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("dedup: %v", err))
	} else {
		result.Deduplicated = dedupResult
	}

	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "intercom", "active", &now); err != nil {
		slog.Error("failed to update intercom sync status", "error", err)
	}

	result.Duration = time.Since(start).String()

	slog.Info("intercom incremental sync complete",
		"org_id", orgID,
		"since", since,
		"duration", result.Duration,
		"errors", len(result.Errors),
	)

	return result
}

func (s *IntercomSyncOrchestratorService) markSyncError(ctx context.Context, orgID uuid.UUID, errMsg string) {
	if err := s.connRepo.UpdateErrorCount(ctx, orgID, "intercom", errMsg); err != nil {
		slog.Error("failed to update intercom error count", "error", err)
	}
}
