package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	OnboardingStepWelcome  = "welcome"
	OnboardingStepStripe   = "stripe"
	OnboardingStepHubSpot  = "hubspot"
	OnboardingStepIntercom = "intercom"
	OnboardingStepPreview  = "preview"
)

var onboardingSteps = []string{
	OnboardingStepWelcome,
	OnboardingStepStripe,
	OnboardingStepHubSpot,
	OnboardingStepIntercom,
	OnboardingStepPreview,
}

const (
	OnboardingEventStepStarted   = "step_started"
	OnboardingEventStepCompleted = "step_completed"
	OnboardingEventStepSkipped   = "step_skipped"
	OnboardingEventCompleted     = "onboarding_completed"
	OnboardingEventAbandoned     = "onboarding_abandoned"
)

var onboardingEvents = []string{
	OnboardingEventStepStarted,
	OnboardingEventStepCompleted,
	OnboardingEventStepSkipped,
	OnboardingEventCompleted,
	OnboardingEventAbandoned,
}

// OnboardingService handles onboarding state transitions and analytics.
type OnboardingService struct {
	statusRepo *repository.OnboardingStatusRepository
	eventRepo  *repository.OnboardingEventRepository
}

// NewOnboardingService creates a new OnboardingService.
func NewOnboardingService(
	statusRepo *repository.OnboardingStatusRepository,
	eventRepo *repository.OnboardingEventRepository,
) *OnboardingService {
	return &OnboardingService{statusRepo: statusRepo, eventRepo: eventRepo}
}

// OnboardingStatusResponse is the response for onboarding status reads/writes.
type OnboardingStatusResponse struct {
	CurrentStep    string         `json:"current_step"`
	CompletedSteps []string       `json:"completed_steps"`
	SkippedSteps   []string       `json:"skipped_steps"`
	StepPayloads   map[string]any `json:"step_payloads"`
	CompletedAt    *time.Time     `json:"completed_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// UpdateOnboardingStatusRequest is the request to update onboarding progression.
type UpdateOnboardingStatusRequest struct {
	StepID     string         `json:"step_id"`
	Action     string         `json:"action"`
	CurrentStep string        `json:"current_step"`
	Payload    map[string]any `json:"payload"`
	Metadata   map[string]any `json:"metadata"`
	DurationMS *int64         `json:"duration_ms"`
}

// OnboardingAnalyticsResponse exposes funnel metrics for the organization.
type OnboardingAnalyticsResponse struct {
	OverallCompletionRate float64                            `json:"overall_completion_rate"`
	AverageStepDurationMS float64                            `json:"average_step_duration_ms"`
	StepMetrics           []repository.OnboardingStepMetrics `json:"step_metrics"`
}

// GetStatus returns onboarding status for an org, creating a default row if missing.
func (s *OnboardingService) GetStatus(ctx context.Context, orgID uuid.UUID) (*OnboardingStatusResponse, error) {
	status, err := s.getOrCreateStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toOnboardingStatusResponse(status), nil
}

// UpdateStatus applies a step/action transition and records analytics events.
func (s *OnboardingService) UpdateStatus(ctx context.Context, orgID uuid.UUID, req UpdateOnboardingStatusRequest) (*OnboardingStatusResponse, error) {
	if !slices.Contains(onboardingEvents, req.Action) {
		return nil, &ValidationError{Field: "action", Message: "invalid onboarding action"}
	}

	if req.StepID != "" && !isValidOnboardingStep(req.StepID) {
		return nil, &ValidationError{Field: "step_id", Message: "invalid onboarding step"}
	}
	if req.CurrentStep != "" && !isValidOnboardingStep(req.CurrentStep) {
		return nil, &ValidationError{Field: "current_step", Message: "invalid onboarding step"}
	}

	status, err := s.getOrCreateStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if status.StepPayloads == nil {
		status.StepPayloads = map[string]any{}
	}
	if req.StepID != "" && req.Payload != nil {
		status.StepPayloads[req.StepID] = req.Payload
	}

	switch req.Action {
	case OnboardingEventStepStarted:
		if req.StepID == "" {
			return nil, &ValidationError{Field: "step_id", Message: "step_id is required for step_started"}
		}
		status.CurrentStep = req.StepID

	case OnboardingEventStepCompleted:
		if req.StepID == "" {
			return nil, &ValidationError{Field: "step_id", Message: "step_id is required for step_completed"}
		}
		status.CompletedSteps = addUniqueStep(status.CompletedSteps, req.StepID)
		status.SkippedSteps = removeStep(status.SkippedSteps, req.StepID)

		if req.CurrentStep != "" {
			status.CurrentStep = req.CurrentStep
		} else if next, ok := nextOnboardingStep(req.StepID); ok {
			status.CurrentStep = next
		}

	case OnboardingEventStepSkipped:
		if req.StepID == "" {
			return nil, &ValidationError{Field: "step_id", Message: "step_id is required for step_skipped"}
		}
		status.SkippedSteps = addUniqueStep(status.SkippedSteps, req.StepID)
		status.CompletedSteps = removeStep(status.CompletedSteps, req.StepID)

		if req.CurrentStep != "" {
			status.CurrentStep = req.CurrentStep
		} else if next, ok := nextOnboardingStep(req.StepID); ok {
			status.CurrentStep = next
		}

	case OnboardingEventCompleted:
		now := time.Now()
		status.CompletedAt = &now
		status.CurrentStep = OnboardingStepPreview
		status.CompletedSteps = addUniqueStep(status.CompletedSteps, OnboardingStepPreview)

	case OnboardingEventAbandoned:
		if req.StepID == "" {
			req.StepID = status.CurrentStep
		}
	}

	if err := s.statusRepo.Upsert(ctx, status); err != nil {
		return nil, fmt.Errorf("update onboarding status: %w", err)
	}

	event := &repository.OnboardingEvent{
		OrgID:      orgID,
		StepID:     req.StepID,
		EventType:  req.Action,
		OccurredAt: time.Now(),
		DurationMS: req.DurationMS,
		Metadata:   req.Metadata,
	}
	if event.StepID == "" {
		event.StepID = status.CurrentStep
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("record onboarding event: %w", err)
	}

	return toOnboardingStatusResponse(status), nil
}

// Complete marks onboarding as completed.
func (s *OnboardingService) Complete(ctx context.Context, orgID uuid.UUID) (*OnboardingStatusResponse, error) {
	return s.UpdateStatus(ctx, orgID, UpdateOnboardingStatusRequest{
		StepID: OnboardingStepPreview,
		Action: OnboardingEventCompleted,
	})
}

// Reset resets onboarding status so users can re-run the wizard.
func (s *OnboardingService) Reset(ctx context.Context, orgID uuid.UUID) (*OnboardingStatusResponse, error) {
	if err := s.statusRepo.Reset(ctx, orgID, OnboardingStepWelcome); err != nil {
		return nil, fmt.Errorf("reset onboarding status: %w", err)
	}

	if err := s.eventRepo.Create(ctx, &repository.OnboardingEvent{
		OrgID:      orgID,
		StepID:     OnboardingStepWelcome,
		EventType:  OnboardingEventAbandoned,
		OccurredAt: time.Now(),
		Metadata:   map[string]any{"reason": "manual_reset"},
	}); err != nil {
		return nil, fmt.Errorf("record onboarding reset event: %w", err)
	}

	status, err := s.getOrCreateStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toOnboardingStatusResponse(status), nil
}

// GetAnalytics returns onboarding funnel analytics for an organization.
func (s *OnboardingService) GetAnalytics(ctx context.Context, orgID uuid.UUID) (*OnboardingAnalyticsResponse, error) {
	summary, err := s.eventRepo.GetAnalyticsSummary(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get onboarding analytics: %w", err)
	}

	return &OnboardingAnalyticsResponse{
		OverallCompletionRate: summary.OverallCompletionRate,
		AverageStepDurationMS: summary.AverageStepDurationMS,
		StepMetrics:           summary.StepMetrics,
	}, nil
}

func (s *OnboardingService) getOrCreateStatus(ctx context.Context, orgID uuid.UUID) (*repository.OnboardingStatus, error) {
	status, err := s.statusRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get onboarding status: %w", err)
	}
	if status != nil {
		if status.StepPayloads == nil {
			status.StepPayloads = map[string]any{}
		}
		return status, nil
	}

	status = &repository.OnboardingStatus{
		OrgID:          orgID,
		CurrentStep:    OnboardingStepWelcome,
		CompletedSteps: []string{},
		SkippedSteps:   []string{},
		StepPayloads:   map[string]any{},
	}
	if err := s.statusRepo.Upsert(ctx, status); err != nil {
		return nil, fmt.Errorf("create onboarding status: %w", err)
	}

	return status, nil
}

func toOnboardingStatusResponse(status *repository.OnboardingStatus) *OnboardingStatusResponse {
	completed := append([]string(nil), status.CompletedSteps...)
	skipped := append([]string(nil), status.SkippedSteps...)
	payloads := map[string]any{}
	for k, v := range status.StepPayloads {
		payloads[k] = v
	}

	return &OnboardingStatusResponse{
		CurrentStep:    status.CurrentStep,
		CompletedSteps: completed,
		SkippedSteps:   skipped,
		StepPayloads:   payloads,
		CompletedAt:    status.CompletedAt,
		UpdatedAt:      status.UpdatedAt,
	}
}

func isValidOnboardingStep(step string) bool {
	return slices.Contains(onboardingSteps, step)
}

func nextOnboardingStep(current string) (string, bool) {
	for i, step := range onboardingSteps {
		if step != current {
			continue
		}
		if i+1 >= len(onboardingSteps) {
			return "", false
		}
		return onboardingSteps[i+1], true
	}
	return "", false
}

func addUniqueStep(steps []string, step string) []string {
	if step == "" || slices.Contains(steps, step) {
		return steps
	}
	return append(steps, step)
}

func removeStep(steps []string, target string) []string {
	if target == "" {
		return steps
	}
	result := make([]string, 0, len(steps))
	for _, step := range steps {
		if step == target {
			continue
		}
		result = append(result, step)
	}
	return result
}
