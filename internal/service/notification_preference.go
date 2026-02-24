package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// NotificationPreferenceService handles notification preference business logic.
type NotificationPreferenceService struct {
	prefRepo *repository.NotificationPreferenceRepository
}

// NewNotificationPreferenceService creates a new NotificationPreferenceService.
func NewNotificationPreferenceService(prefRepo *repository.NotificationPreferenceRepository) *NotificationPreferenceService {
	return &NotificationPreferenceService{prefRepo: prefRepo}
}

// UpdatePreferencesRequest holds input for updating notification preferences.
type UpdatePreferencesRequest struct {
	EmailEnabled    *bool        `json:"email_enabled"`
	InAppEnabled    *bool        `json:"in_app_enabled"`
	DigestEnabled   *bool        `json:"digest_enabled"`
	DigestFrequency *string      `json:"digest_frequency"`
	MutedRuleIDs    *[]uuid.UUID `json:"muted_rule_ids"`
}

var validDigestFrequencies = map[string]bool{
	"daily":  true,
	"weekly": true,
}

// Get returns the notification preferences for a user in an org.
func (s *NotificationPreferenceService) Get(ctx context.Context, userID, orgID uuid.UUID) (*repository.NotificationPreference, error) {
	return s.prefRepo.GetByUserAndOrg(ctx, userID, orgID)
}

// Update applies partial updates to notification preferences.
func (s *NotificationPreferenceService) Update(ctx context.Context, userID, orgID uuid.UUID, req UpdatePreferencesRequest) (*repository.NotificationPreference, error) {
	pref, err := s.prefRepo.GetByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	if req.EmailEnabled != nil {
		pref.EmailEnabled = *req.EmailEnabled
	}
	if req.InAppEnabled != nil {
		pref.InAppEnabled = *req.InAppEnabled
	}
	if req.DigestEnabled != nil {
		pref.DigestEnabled = *req.DigestEnabled
	}
	if req.DigestFrequency != nil {
		if !validDigestFrequencies[*req.DigestFrequency] {
			return nil, &ValidationError{Message: "digest_frequency must be 'daily' or 'weekly'"}
		}
		pref.DigestFrequency = *req.DigestFrequency
	}
	if req.MutedRuleIDs != nil {
		pref.MutedRuleIDs = *req.MutedRuleIDs
	}

	pref.UserID = userID
	pref.OrgID = orgID

	if err := s.prefRepo.Upsert(ctx, pref); err != nil {
		return nil, fmt.Errorf("update notification preferences: %w", err)
	}

	return pref, nil
}

// ShouldNotify checks whether a notification should be sent to recipients
// based on notification preferences and muted rules.
func (s *NotificationPreferenceService) ShouldNotifyEmail(ctx context.Context, userID, orgID uuid.UUID, ruleID uuid.UUID, _ time.Time) bool {
	pref, err := s.prefRepo.GetByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return true // default to notify on error
	}

	if !pref.EmailEnabled {
		return false
	}

	for _, mutedID := range pref.MutedRuleIDs {
		if mutedID == ruleID {
			return false
		}
	}

	return true
}
