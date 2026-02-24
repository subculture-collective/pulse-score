package service

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/onnwee/pulse-score/internal/repository"
)

// AlertRuleService handles alert rule business logic.
type AlertRuleService struct {
	alertRepo *repository.AlertRuleRepository
}

// NewAlertRuleService creates a new AlertRuleService.
func NewAlertRuleService(alertRepo *repository.AlertRuleRepository) *AlertRuleService {
	return &AlertRuleService{alertRepo: alertRepo}
}

// CreateAlertRuleRequest holds input for creating an alert rule.
type CreateAlertRuleRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	TriggerType string         `json:"trigger_type"`
	Conditions  map[string]any `json:"conditions"`
	Channel     string         `json:"channel"`
	Recipients  []string       `json:"recipients"`
	IsActive    *bool          `json:"is_active"`
}

// UpdateAlertRuleRequest holds input for updating an alert rule.
type UpdateAlertRuleRequest struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	TriggerType *string         `json:"trigger_type"`
	Conditions  *map[string]any `json:"conditions"`
	Channel     *string         `json:"channel"`
	Recipients  *[]string       `json:"recipients"`
	IsActive    *bool           `json:"is_active"`
}

var validTriggerTypes = map[string]bool{
	"score_drop":     true,
	"risk_change":    true,
	"payment_failed": true,
}

var validChannels = map[string]bool{
	"email": true,
}

// List returns all alert rules for an org.
func (s *AlertRuleService) List(ctx context.Context, orgID uuid.UUID) ([]*repository.AlertRule, error) {
	rules, err := s.alertRepo.List(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}
	return rules, nil
}

// GetByID returns a single alert rule.
func (s *AlertRuleService) GetByID(ctx context.Context, id, orgID uuid.UUID) (*repository.AlertRule, error) {
	rule, err := s.alertRepo.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("get alert rule: %w", err)
	}
	if rule == nil {
		return nil, &NotFoundError{Resource: "alert_rule", Message: "alert rule not found"}
	}
	return rule, nil
}

// Create creates a new alert rule.
func (s *AlertRuleService) Create(ctx context.Context, orgID, userID uuid.UUID, req CreateAlertRuleRequest) (*repository.AlertRule, error) {
	if err := s.validateCreate(req); err != nil {
		return nil, err
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	rule := &repository.AlertRule{
		OrgID:       orgID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		TriggerType: req.TriggerType,
		Conditions:  req.Conditions,
		Channel:     req.Channel,
		Recipients:  req.Recipients,
		IsActive:    isActive,
		CreatedBy:   &userID,
	}

	if err := s.alertRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create alert rule: %w", err)
	}

	return rule, nil
}

// Update updates an existing alert rule.
func (s *AlertRuleService) Update(ctx context.Context, id, orgID uuid.UUID, req UpdateAlertRuleRequest) (*repository.AlertRule, error) {
	rule, err := s.alertRepo.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("get alert rule: %w", err)
	}
	if rule == nil {
		return nil, &NotFoundError{Resource: "alert_rule", Message: "alert rule not found"}
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, &ValidationError{Field: "name", Message: "name is required"}
		}
		rule.Name = name
	}
	if req.Description != nil {
		rule.Description = strings.TrimSpace(*req.Description)
	}
	if req.TriggerType != nil {
		if !validTriggerTypes[*req.TriggerType] {
			return nil, &ValidationError{Field: "trigger_type", Message: "invalid trigger type"}
		}
		rule.TriggerType = *req.TriggerType
	}
	if req.Conditions != nil {
		if err := s.validateConditions(rule.TriggerType, *req.Conditions); err != nil {
			return nil, err
		}
		rule.Conditions = *req.Conditions
	}
	if req.Channel != nil {
		if !validChannels[*req.Channel] {
			return nil, &ValidationError{Field: "channel", Message: "invalid channel"}
		}
		rule.Channel = *req.Channel
	}
	if req.Recipients != nil {
		if err := s.validateRecipients(*req.Recipients); err != nil {
			return nil, err
		}
		rule.Recipients = *req.Recipients
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	if err := s.alertRepo.Update(ctx, rule); err != nil {
		if err == pgx.ErrNoRows {
			return nil, &NotFoundError{Resource: "alert_rule", Message: "alert rule not found"}
		}
		return nil, fmt.Errorf("update alert rule: %w", err)
	}

	return s.alertRepo.GetByID(ctx, id, orgID)
}

// Delete deletes an alert rule.
func (s *AlertRuleService) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	if err := s.alertRepo.Delete(ctx, id, orgID); err != nil {
		if err == pgx.ErrNoRows {
			return &NotFoundError{Resource: "alert_rule", Message: "alert rule not found"}
		}
		return fmt.Errorf("delete alert rule: %w", err)
	}
	return nil
}

func (s *AlertRuleService) validateCreate(req CreateAlertRuleRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if !validTriggerTypes[req.TriggerType] {
		return &ValidationError{Field: "trigger_type", Message: "invalid trigger type; must be score_drop, risk_change, or payment_failed"}
	}
	if req.Conditions == nil {
		return &ValidationError{Field: "conditions", Message: "conditions are required"}
	}
	if err := s.validateConditions(req.TriggerType, req.Conditions); err != nil {
		return err
	}
	channel := req.Channel
	if channel == "" {
		channel = "email"
	}
	if !validChannels[channel] {
		return &ValidationError{Field: "channel", Message: "invalid channel"}
	}
	if len(req.Recipients) == 0 {
		return &ValidationError{Field: "recipients", Message: "at least one recipient is required"}
	}
	return s.validateRecipients(req.Recipients)
}

func (s *AlertRuleService) validateConditions(triggerType string, conditions map[string]any) error {
	switch triggerType {
	case "score_drop":
		if _, ok := conditions["threshold"]; !ok {
			return &ValidationError{Field: "conditions.threshold", Message: "threshold is required for score_drop"}
		}
	case "risk_change":
		if _, ok := conditions["from"]; !ok {
			return &ValidationError{Field: "conditions.from", Message: "from is required for risk_change"}
		}
		if _, ok := conditions["to"]; !ok {
			return &ValidationError{Field: "conditions.to", Message: "to is required for risk_change"}
		}
	case "payment_failed":
		// No required conditions for payment_failed
	}
	return nil
}

func (s *AlertRuleService) validateRecipients(recipients []string) error {
	for _, r := range recipients {
		if _, err := mail.ParseAddress(r); err != nil {
			return &ValidationError{Field: "recipients", Message: fmt.Sprintf("invalid email: %s", r)}
		}
	}
	return nil
}
