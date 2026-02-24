package scoring

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

// ConfigService manages scoring configuration per org.
type ConfigService struct {
	configRepo *repository.ScoringConfigRepository
	scheduler  *ScoreScheduler
}

// NewConfigService creates a new ConfigService.
func NewConfigService(configRepo *repository.ScoringConfigRepository, scheduler *ScoreScheduler) *ConfigService {
	return &ConfigService{
		configRepo: configRepo,
		scheduler:  scheduler,
	}
}

// GetConfig returns the scoring config for an org, creating defaults if needed.
func (s *ConfigService) GetConfig(ctx context.Context, orgID uuid.UUID) (*repository.ScoringConfig, error) {
	config, err := s.configRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get scoring config: %w", err)
	}
	if config == nil {
		config, err = s.configRepo.CreateDefault(ctx, orgID)
		if err != nil {
			return nil, fmt.Errorf("create default config: %w", err)
		}
	}
	return config, nil
}

// UpdateConfigRequest holds the fields for updating scoring config.
type UpdateConfigRequest struct {
	Weights    map[string]float64 `json:"weights"`
	Thresholds map[string]int     `json:"thresholds"`
}

// UpdateConfig validates and updates the scoring config, then triggers recalculation.
func (s *ConfigService) UpdateConfig(ctx context.Context, orgID uuid.UUID, req UpdateConfigRequest) (*repository.ScoringConfig, error) {
	if req.Weights != nil {
		if err := repository.ValidateWeights(req.Weights); err != nil {
			return nil, &service.ValidationError{Field: "weights", Message: err.Error()}
		}
	}
	if req.Thresholds != nil {
		if err := repository.ValidateThresholds(req.Thresholds); err != nil {
			return nil, &service.ValidationError{Field: "thresholds", Message: err.Error()}
		}
	}

	// Get existing or create default
	config, err := s.GetConfig(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if req.Weights != nil {
		config.Weights = req.Weights
	}
	if req.Thresholds != nil {
		config.Thresholds = req.Thresholds
	}

	if err := s.configRepo.Upsert(ctx, config); err != nil {
		return nil, fmt.Errorf("update scoring config: %w", err)
	}

	// Trigger async recalculation for the org
	if s.scheduler != nil {
		go func() { _ = s.scheduler.RecalculateOrg(context.Background(), orgID) }()
	}

	return config, nil
}
