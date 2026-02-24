package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntercomMetrics holds computed Intercom metrics for a customer.
type IntercomMetrics struct {
	CustomerID uuid.UUID `json:"customer_id"`

	// Ticket counts
	OpenTickets   int `json:"open_tickets"`
	ClosedTickets int `json:"closed_tickets"`
	TotalTickets  int `json:"total_tickets"`

	// Response time
	AvgResolutionHours float64 `json:"avg_resolution_hours"`

	// Trends (total conversations in time windows)
	Trend7d  int `json:"trend_7d"`
	Trend30d int `json:"trend_30d"`
	Trend90d int `json:"trend_90d"`
}

// IntercomOrgMetrics holds aggregate Intercom metrics for an organization.
type IntercomOrgMetrics struct {
	OrgID             uuid.UUID `json:"org_id"`
	TotalConversations int      `json:"total_conversations"`
	TotalContacts      int      `json:"total_contacts"`
	OpenConversations  int      `json:"open_conversations"`
	AvgResolutionHours float64  `json:"avg_resolution_hours"`
}

// IntercomMetricsService computes Intercom-based health metrics.
type IntercomMetricsService struct {
	contacts      *repository.IntercomContactRepository
	conversations *repository.IntercomConversationRepository
}

// NewIntercomMetricsService creates a new IntercomMetricsService.
func NewIntercomMetricsService(
	contacts *repository.IntercomContactRepository,
	conversations *repository.IntercomConversationRepository,
) *IntercomMetricsService {
	return &IntercomMetricsService{
		contacts:      contacts,
		conversations: conversations,
	}
}

// GetCustomerMetrics calculates Intercom metrics for a specific customer.
func (s *IntercomMetricsService) GetCustomerMetrics(ctx context.Context, orgID, customerID uuid.UUID) (*IntercomMetrics, error) {
	metrics := &IntercomMetrics{CustomerID: customerID}
	now := time.Now()

	// Get open and closed counts since the beginning (use zero time for all-time counts)
	open, closed, err := s.conversations.CountByCustomerAndState(ctx, customerID, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("count tickets by state: %w", err)
	}
	metrics.OpenTickets = open
	metrics.ClosedTickets = closed
	metrics.TotalTickets = open + closed

	// Average resolution hours (all time)
	avgHours, err := s.conversations.AvgResolutionHours(ctx, customerID, time.Time{})
	if err != nil {
		slog.Error("failed to get avg resolution hours", "customer_id", customerID, "error", err)
	} else {
		metrics.AvgResolutionHours = avgHours
	}

	// Trends: total conversations (open + closed) created in time windows
	open7, closed7, err := s.conversations.CountByCustomerAndState(ctx, customerID, now.AddDate(0, 0, -7))
	if err != nil {
		slog.Error("failed to count 7d conversations", "error", err)
	} else {
		metrics.Trend7d = open7 + closed7
	}

	open30, closed30, err := s.conversations.CountByCustomerAndState(ctx, customerID, now.AddDate(0, 0, -30))
	if err != nil {
		slog.Error("failed to count 30d conversations", "error", err)
	} else {
		metrics.Trend30d = open30 + closed30
	}

	open90, closed90, err := s.conversations.CountByCustomerAndState(ctx, customerID, now.AddDate(0, 0, -90))
	if err != nil {
		slog.Error("failed to count 90d conversations", "error", err)
	} else {
		metrics.Trend90d = open90 + closed90
	}

	return metrics, nil
}

// GetOrgMetrics calculates aggregate Intercom metrics for an organization.
func (s *IntercomMetricsService) GetOrgMetrics(ctx context.Context, orgID uuid.UUID) (*IntercomOrgMetrics, error) {
	metrics := &IntercomOrgMetrics{OrgID: orgID}

	convCount, err := s.conversations.CountByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count conversations: %w", err)
	}
	metrics.TotalConversations = convCount

	contactCount, err := s.contacts.CountByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count contacts: %w", err)
	}
	metrics.TotalContacts = contactCount

	return metrics, nil
}
