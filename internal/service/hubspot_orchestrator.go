package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// HubSpotSyncResult contains the results of a full HubSpot sync.
type HubSpotSyncResult struct {
	Contacts     *SyncProgress        `json:"contacts"`
	Deals        *SyncProgress        `json:"deals"`
	Companies    *SyncProgress        `json:"companies"`
	Enriched     bool                 `json:"enriched"`
	Deduplicated *DeduplicationResult `json:"deduplicated,omitempty"`
	Duration     string               `json:"duration"`
	Errors       []string             `json:"errors,omitempty"`
}

// HubSpotSyncOrchestratorService orchestrates the full HubSpot sync pipeline.
type HubSpotSyncOrchestratorService struct {
	connRepo *repository.IntegrationConnectionRepository
	syncSvc  *HubSpotSyncService
	mergeSvc *CustomerMergeService
}

// NewHubSpotSyncOrchestratorService creates a new HubSpotSyncOrchestratorService.
func NewHubSpotSyncOrchestratorService(
	connRepo *repository.IntegrationConnectionRepository,
	syncSvc *HubSpotSyncService,
	mergeSvc *CustomerMergeService,
) *HubSpotSyncOrchestratorService {
	return &HubSpotSyncOrchestratorService{
		connRepo: connRepo,
		syncSvc:  syncSvc,
		mergeSvc: mergeSvc,
	}
}

// RunFullSync runs the complete HubSpot sync pipeline for an org.
func (s *HubSpotSyncOrchestratorService) RunFullSync(ctx context.Context, orgID uuid.UUID) *HubSpotSyncResult {
	start := time.Now()
	result := &HubSpotSyncResult{}

	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "hubspot", "syncing", nil); err != nil {
		slog.Error("failed to update hubspot sync status", "error", err)
	}

	// Step 1: Sync contacts
	contactProgress, err := s.syncSvc.SyncContacts(ctx, orgID)
	result.Contacts = contactProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("contact sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 2: Sync deals
	dealProgress, err := s.syncSvc.SyncDeals(ctx, orgID)
	result.Deals = dealProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("deal sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 3: Sync companies
	companyProgress, err := s.syncSvc.SyncCompanies(ctx, orgID)
	result.Companies = companyProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("company sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 4: Enrich customers with company data
	if err := s.syncSvc.EnrichCustomersWithCompanyData(ctx, orgID); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("enrichment: %v", err))
	} else {
		result.Enriched = true
	}

	// Step 5: Deduplication
	dedupResult, err := s.mergeSvc.DeduplicateCustomers(ctx, orgID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("dedup: %v", err))
	} else {
		result.Deduplicated = dedupResult
	}

	// Mark sync complete
	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "hubspot", "active", &now); err != nil {
		slog.Error("failed to update hubspot sync status", "error", err)
	}

	result.Duration = time.Since(start).String()

	slog.Info("hubspot full sync complete",
		"org_id", orgID,
		"duration", result.Duration,
		"errors", len(result.Errors),
	)

	return result
}

// RunIncrementalSync runs an incremental HubSpot sync for records modified since the given time.
func (s *HubSpotSyncOrchestratorService) RunIncrementalSync(ctx context.Context, orgID uuid.UUID, since time.Time) *HubSpotSyncResult {
	start := time.Now()
	result := &HubSpotSyncResult{}

	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "hubspot", "syncing", nil); err != nil {
		slog.Error("failed to update hubspot sync status", "error", err)
	}

	// Step 1: Sync contacts modified since
	contactProgress, err := s.syncSvc.SyncContactsSince(ctx, orgID, since)
	result.Contacts = contactProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("incremental contact sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 2: Sync deals modified since
	dealProgress, err := s.syncSvc.SyncDealsSince(ctx, orgID, since)
	result.Deals = dealProgress
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("incremental deal sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}

	// Step 3: Re-enrich customers (only for changed contacts)
	if err := s.syncSvc.EnrichCustomersWithCompanyData(ctx, orgID); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("enrichment: %v", err))
	} else {
		result.Enriched = true
	}

	// Step 4: Deduplication for any new records
	dedupResult, err := s.mergeSvc.DeduplicateCustomers(ctx, orgID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("dedup: %v", err))
	} else {
		result.Deduplicated = dedupResult
	}

	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, "hubspot", "active", &now); err != nil {
		slog.Error("failed to update hubspot sync status", "error", err)
	}

	result.Duration = time.Since(start).String()

	slog.Info("hubspot incremental sync complete",
		"org_id", orgID,
		"since", since,
		"duration", result.Duration,
		"errors", len(result.Errors),
	)

	return result
}

func (s *HubSpotSyncOrchestratorService) markSyncError(ctx context.Context, orgID uuid.UUID, errMsg string) {
	if err := s.connRepo.UpdateErrorCount(ctx, orgID, "hubspot", errMsg); err != nil {
		slog.Error("failed to update hubspot error count", "error", err)
	}
}
