package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// SyncSchedulerService runs periodic incremental syncs for all active connections.
type SyncSchedulerService struct {
	connRepo              *repository.IntegrationConnectionRepository
	orchestrator          *SyncOrchestratorService
	hubspotOrchestrator   *HubSpotSyncOrchestratorService
	interval              time.Duration

	// Per-connection lock to prevent overlapping syncs
	locks map[uuid.UUID]*sync.Mutex
	mu    sync.Mutex
}

// NewSyncSchedulerService creates a new SyncSchedulerService.
func NewSyncSchedulerService(
	connRepo *repository.IntegrationConnectionRepository,
	orchestrator *SyncOrchestratorService,
	hubspotOrchestrator *HubSpotSyncOrchestratorService,
	intervalMinutes int,
) *SyncSchedulerService {
	return &SyncSchedulerService{
		connRepo:            connRepo,
		orchestrator:        orchestrator,
		hubspotOrchestrator: hubspotOrchestrator,
		interval:            time.Duration(intervalMinutes) * time.Minute,
		locks:               make(map[uuid.UUID]*sync.Mutex),
	}
}

// Start begins the periodic sync scheduler. Cancel the context to stop.
func (s *SyncSchedulerService) Start(ctx context.Context) {
	slog.Info("sync scheduler started", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("sync scheduler stopped")
			return
		case <-ticker.C:
			s.runCycle(ctx)
		}
	}
}

func (s *SyncSchedulerService) runCycle(ctx context.Context) {
	// Stripe connections
	conns, err := s.connRepo.ListActiveByProvider(ctx, "stripe")
	if err != nil {
		slog.Error("scheduler: failed to list stripe connections", "error", err)
	} else {
		for _, conn := range conns {
			lock := s.getLock(conn.OrgID)
			if !lock.TryLock() {
				slog.Debug("scheduler: skipping org (sync in progress)", "org_id", conn.OrgID)
				continue
			}

			go func(orgID uuid.UUID, lastSync *time.Time) {
				defer lock.Unlock()

				syncCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				defer cancel()

				if lastSync != nil {
					s.orchestrator.RunIncrementalSync(syncCtx, orgID, *lastSync)
				} else {
					s.orchestrator.RunFullSync(syncCtx, orgID)
				}
			}(conn.OrgID, conn.LastSyncAt)
		}
	}

	// HubSpot connections
	if s.hubspotOrchestrator != nil {
		hsConns, err := s.connRepo.ListActiveByProvider(ctx, "hubspot")
		if err != nil {
			slog.Error("scheduler: failed to list hubspot connections", "error", err)
		} else {
			for _, conn := range hsConns {
				lock := s.getLock(conn.OrgID)
				if !lock.TryLock() {
					slog.Debug("scheduler: skipping hubspot org (sync in progress)", "org_id", conn.OrgID)
					continue
				}

				go func(orgID uuid.UUID, lastSync *time.Time) {
					defer lock.Unlock()

					syncCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
					defer cancel()

					if lastSync != nil {
						s.hubspotOrchestrator.RunIncrementalSync(syncCtx, orgID, *lastSync)
					} else {
						s.hubspotOrchestrator.RunFullSync(syncCtx, orgID)
					}
				}(conn.OrgID, conn.LastSyncAt)
			}
		}
	}
}

func (s *SyncSchedulerService) getLock(orgID uuid.UUID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.locks[orgID]; !ok {
		s.locks[orgID] = &sync.Mutex{}
	}
	return s.locks[orgID]
}
