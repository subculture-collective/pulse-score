package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// AlertMatch represents a rule that matched for a specific customer.
type AlertMatch struct {
	Rule        *repository.AlertRule
	Customer    *repository.Customer
	TriggerData map[string]any
}

// AlertEngine evaluates alert rules against current data.
type AlertEngine struct {
	alertRules     *repository.AlertRuleRepository
	alertHistory   *repository.AlertHistoryRepository
	healthScores   *repository.HealthScoreRepository
	customers      *repository.CustomerRepository
	events         *repository.CustomerEventRepository
	defaultCooldown time.Duration
}

// NewAlertEngine creates a new AlertEngine.
func NewAlertEngine(
	alertRules *repository.AlertRuleRepository,
	alertHistory *repository.AlertHistoryRepository,
	healthScores *repository.HealthScoreRepository,
	customers *repository.CustomerRepository,
	events *repository.CustomerEventRepository,
	defaultCooldownHours int,
) *AlertEngine {
	return &AlertEngine{
		alertRules:      alertRules,
		alertHistory:    alertHistory,
		healthScores:    healthScores,
		customers:       customers,
		events:          events,
		defaultCooldown: time.Duration(defaultCooldownHours) * time.Hour,
	}
}

// EvaluateAll evaluates all active rules for an org and returns matches.
func (e *AlertEngine) EvaluateAll(ctx context.Context, orgID uuid.UUID) ([]AlertMatch, error) {
	rules, err := e.alertHistory.ListActiveRulesByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active rules: %w", err)
	}

	var allMatches []AlertMatch
	for _, rule := range rules {
		matches, err := e.EvaluateRule(ctx, rule, orgID)
		if err != nil {
			slog.Error("rule evaluation error",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"error", err,
			)
			continue
		}
		allMatches = append(allMatches, matches...)
	}

	return allMatches, nil
}

// EvaluateRule evaluates a single rule against all customers in an org.
func (e *AlertEngine) EvaluateRule(ctx context.Context, rule *repository.AlertRule, orgID uuid.UUID) ([]AlertMatch, error) {
	switch rule.TriggerType {
	case "score_below":
		return e.evaluateScoreBelow(ctx, rule, orgID)
	case "score_drop":
		return e.evaluateScoreDrop(ctx, rule, orgID)
	case "risk_change":
		return e.evaluateRiskChange(ctx, rule, orgID)
	case "payment_failed":
		return e.evaluateEventTrigger(ctx, rule, orgID, "payment.failed")
	default:
		return nil, fmt.Errorf("unknown trigger type: %s", rule.TriggerType)
	}
}

// EvaluateForCustomer evaluates all active rules for a single customer (used by real-time hook).
func (e *AlertEngine) EvaluateForCustomer(ctx context.Context, customerID, orgID uuid.UUID) ([]AlertMatch, error) {
	rules, err := e.alertHistory.ListActiveRulesByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active rules: %w", err)
	}

	customer, err := e.customers.GetByIDAndOrg(ctx, customerID, orgID)
	if err != nil || customer == nil {
		return nil, err
	}

	var allMatches []AlertMatch
	for _, rule := range rules {
		match, err := e.evaluateRuleForCustomer(ctx, rule, customer)
		if err != nil {
			slog.Error("rule evaluation error for customer",
				"rule_id", rule.ID,
				"customer_id", customerID,
				"error", err,
			)
			continue
		}
		if match != nil {
			allMatches = append(allMatches, *match)
		}
	}

	return allMatches, nil
}

func (e *AlertEngine) evaluateRuleForCustomer(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer) (*AlertMatch, error) {
	switch rule.TriggerType {
	case "score_below":
		return e.evaluateScoreBelowForCustomer(ctx, rule, customer)
	case "score_drop":
		return e.evaluateScoreDropForCustomer(ctx, rule, customer)
	case "risk_change":
		return e.evaluateRiskChangeForCustomer(ctx, rule, customer)
	case "payment_failed":
		return e.evaluateEventTriggerForCustomer(ctx, rule, customer, "payment.failed")
	default:
		return nil, nil
	}
}

// evaluateScoreBelow checks for customers with score below threshold.
func (e *AlertEngine) evaluateScoreBelow(ctx context.Context, rule *repository.AlertRule, orgID uuid.UUID) ([]AlertMatch, error) {
	threshold := getConditionInt(rule.Conditions, "threshold", 40)

	scores, err := e.healthScores.ListByOrg(ctx, orgID, repository.HealthScoreFilters{Limit: 1000})
	if err != nil {
		return nil, err
	}

	var matches []AlertMatch
	for _, score := range scores {
		if score.OverallScore >= threshold {
			continue
		}

		if e.isInCooldown(ctx, rule.ID, score.CustomerID) {
			continue
		}

		customer, err := e.customers.GetByIDAndOrg(ctx, score.CustomerID, orgID)
		if err != nil || customer == nil {
			continue
		}

		matches = append(matches, AlertMatch{
			Rule:     rule,
			Customer: customer,
			TriggerData: map[string]any{
				"customer_id": customer.ID.String(),
				"score":       score.OverallScore,
				"threshold":   threshold,
				"risk_level":  score.RiskLevel,
			},
		})
	}
	return matches, nil
}

func (e *AlertEngine) evaluateScoreBelowForCustomer(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer) (*AlertMatch, error) {
	threshold := getConditionInt(rule.Conditions, "threshold", 40)

	score, err := e.healthScores.GetByCustomerID(ctx, customer.ID, customer.OrgID)
	if err != nil || score == nil {
		return nil, err
	}

	if score.OverallScore >= threshold {
		return nil, nil
	}

	if e.isInCooldown(ctx, rule.ID, customer.ID) {
		return nil, nil
	}

	return &AlertMatch{
		Rule:     rule,
		Customer: customer,
		TriggerData: map[string]any{
			"customer_id": customer.ID.String(),
			"score":       score.OverallScore,
			"threshold":   threshold,
			"risk_level":  score.RiskLevel,
		},
	}, nil
}

// evaluateScoreDrop checks for customers whose score dropped significantly.
func (e *AlertEngine) evaluateScoreDrop(ctx context.Context, rule *repository.AlertRule, orgID uuid.UUID) ([]AlertMatch, error) {
	points := getConditionInt(rule.Conditions, "points", 10)
	days := getConditionInt(rule.Conditions, "days", 7)

	customers, err := e.customers.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var matches []AlertMatch
	for _, customer := range customers {
		match, err := e.checkScoreDrop(ctx, rule, customer, points, days)
		if err != nil {
			continue
		}
		if match != nil {
			matches = append(matches, *match)
		}
	}
	return matches, nil
}

func (e *AlertEngine) evaluateScoreDropForCustomer(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer) (*AlertMatch, error) {
	points := getConditionInt(rule.Conditions, "points", 10)
	days := getConditionInt(rule.Conditions, "days", 7)
	return e.checkScoreDrop(ctx, rule, customer, points, days)
}

func (e *AlertEngine) checkScoreDrop(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer, points, days int) (*AlertMatch, error) {
	current, err := e.healthScores.GetByCustomerID(ctx, customer.ID, customer.OrgID)
	if err != nil || current == nil {
		return nil, err
	}

	historical, err := e.healthScores.GetScoreAtTime(ctx, customer.ID, customer.OrgID, time.Now().AddDate(0, 0, -days))
	if err != nil || historical == nil {
		return nil, err
	}

	delta := current.OverallScore - historical.OverallScore
	if delta >= 0 || int(math.Abs(float64(delta))) < points {
		return nil, nil
	}

	if e.isInCooldown(ctx, rule.ID, customer.ID) {
		return nil, nil
	}

	// Find biggest negative factor
	biggestFactor := ""
	if current.Factors != nil {
		minVal := 100.0
		for k, v := range current.Factors {
			if v < minVal {
				minVal = v
				biggestFactor = k
			}
		}
	}

	return &AlertMatch{
		Rule:     rule,
		Customer: customer,
		TriggerData: map[string]any{
			"customer_id":                customer.ID.String(),
			"old_score":                  historical.OverallScore,
			"new_score":                  current.OverallScore,
			"delta":                      delta,
			"days":                       days,
			"biggest_contributing_factor": biggestFactor,
			"risk_level":                 current.RiskLevel,
		},
	}, nil
}

// evaluateRiskChange checks for customers with recent risk level changes.
func (e *AlertEngine) evaluateRiskChange(ctx context.Context, rule *repository.AlertRule, orgID uuid.UUID) ([]AlertMatch, error) {
	cooldown := e.getCooldown(rule.Conditions)
	since := time.Now().Add(-cooldown)

	// Query recent risk_level.changed events
	customers, err := e.customers.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var matches []AlertMatch
	for _, customer := range customers {
		match, err := e.checkRiskChange(ctx, rule, customer, since)
		if err != nil {
			continue
		}
		if match != nil {
			matches = append(matches, *match)
		}
	}
	return matches, nil
}

func (e *AlertEngine) evaluateRiskChangeForCustomer(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer) (*AlertMatch, error) {
	cooldown := e.getCooldown(rule.Conditions)
	since := time.Now().Add(-cooldown)
	return e.checkRiskChange(ctx, rule, customer, since)
}

func (e *AlertEngine) checkRiskChange(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer, since time.Time) (*AlertMatch, error) {
	events, err := e.events.ListByCustomerAndType(ctx, customer.ID, "risk_level.changed", since)
	if err != nil || len(events) == 0 {
		return nil, err
	}

	latestEvent := events[0]
	fromLevel, _ := latestEvent.Data["previous_level"].(string)
	toLevel, _ := latestEvent.Data["new_level"].(string)

	condFrom, _ := rule.Conditions["from"].(string)
	condTo, _ := rule.Conditions["to"].(string)

	if condFrom != "" && fromLevel != condFrom {
		return nil, nil
	}
	if condTo != "" && toLevel != condTo {
		return nil, nil
	}

	if e.isInCooldown(ctx, rule.ID, customer.ID) {
		return nil, nil
	}

	score, _ := latestEvent.Data["score"]

	return &AlertMatch{
		Rule:     rule,
		Customer: customer,
		TriggerData: map[string]any{
			"customer_id":    customer.ID.String(),
			"previous_level": fromLevel,
			"new_level":      toLevel,
			"score":          score,
		},
	}, nil
}

// evaluateEventTrigger checks for recent events of a specific type.
func (e *AlertEngine) evaluateEventTrigger(ctx context.Context, rule *repository.AlertRule, orgID uuid.UUID, eventType string) ([]AlertMatch, error) {
	cooldown := e.getCooldown(rule.Conditions)
	since := time.Now().Add(-cooldown)

	customers, err := e.customers.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var matches []AlertMatch
	for _, customer := range customers {
		match, err := e.checkEventTrigger(ctx, rule, customer, eventType, since)
		if err != nil {
			continue
		}
		if match != nil {
			matches = append(matches, *match)
		}
	}
	return matches, nil
}

func (e *AlertEngine) evaluateEventTriggerForCustomer(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer, eventType string) (*AlertMatch, error) {
	cooldown := e.getCooldown(rule.Conditions)
	since := time.Now().Add(-cooldown)
	return e.checkEventTrigger(ctx, rule, customer, eventType, since)
}

func (e *AlertEngine) checkEventTrigger(ctx context.Context, rule *repository.AlertRule, customer *repository.Customer, eventType string, since time.Time) (*AlertMatch, error) {
	events, err := e.events.ListByCustomerAndType(ctx, customer.ID, eventType, since)
	if err != nil || len(events) == 0 {
		return nil, err
	}

	if e.isInCooldown(ctx, rule.ID, customer.ID) {
		return nil, nil
	}

	latestEvent := events[0]
	triggerData := map[string]any{
		"customer_id": customer.ID.String(),
		"event_type":  eventType,
		"event_id":    latestEvent.ID.String(),
		"occurred_at": latestEvent.OccurredAt.Format(time.RFC3339),
	}
	// Merge event data
	for k, v := range latestEvent.Data {
		triggerData[k] = v
	}

	return &AlertMatch{
		Rule:     rule,
		Customer: customer,
		TriggerData: triggerData,
	}, nil
}

// isInCooldown checks if an alert was recently sent for this rule+customer combo.
func (e *AlertEngine) isInCooldown(ctx context.Context, ruleID, customerID uuid.UUID) bool {
	last, err := e.alertHistory.GetLastAlertForRule(ctx, ruleID, customerID)
	if err != nil || last == nil {
		return false
	}

	cooldownEnd := last.CreatedAt.Add(e.defaultCooldown)
	return time.Now().Before(cooldownEnd)
}

func (e *AlertEngine) getCooldown(conditions map[string]any) time.Duration {
	if hours := getConditionInt(conditions, "cooldown_hours", 0); hours > 0 {
		return time.Duration(hours) * time.Hour
	}
	return e.defaultCooldown
}

func getConditionInt(conditions map[string]any, key string, defaultVal int) int {
	v, ok := conditions[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return int(i)
		}
	}
	return defaultVal
}
