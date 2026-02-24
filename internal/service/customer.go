package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/onnwee/pulse-score/internal/repository"
)

// CustomerService handles customer-related business logic.
type CustomerService struct {
	customerRepo *repository.CustomerRepository
	healthRepo   *repository.HealthScoreRepository
	subRepo      *repository.StripeSubscriptionRepository
	eventRepo    *repository.CustomerEventRepository
}

// NewCustomerService creates a new CustomerService.
func NewCustomerService(
	cr *repository.CustomerRepository,
	hr *repository.HealthScoreRepository,
	sr *repository.StripeSubscriptionRepository,
	er *repository.CustomerEventRepository,
) *CustomerService {
	return &CustomerService{
		customerRepo: cr,
		healthRepo:   hr,
		subRepo:      sr,
		eventRepo:    er,
	}
}

// CustomerListResponse is the JSON response for customer list.
type CustomerListResponse struct {
	Customers  []CustomerListItem `json:"customers"`
	Pagination PaginationMeta     `json:"pagination"`
}

// CustomerListItem is a single customer in the list response.
type CustomerListItem struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	CompanyName  string     `json:"company_name"`
	MRRCents     int        `json:"mrr_cents"`
	Source       string     `json:"source"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	OverallScore *int       `json:"overall_score"`
	RiskLevel    *string    `json:"risk_level"`
}

// PaginationMeta holds pagination metadata for list responses.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// List returns a paginated list of customers with health scores.
func (s *CustomerService) List(ctx context.Context, params repository.CustomerListParams) (*CustomerListResponse, error) {
	// Validate and default params
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 25
	}
	if params.PerPage > 100 {
		params.PerPage = 100
	}

	validSorts := map[string]bool{"name": true, "mrr": true, "score": true, "last_seen": true}
	if !validSorts[params.Sort] {
		params.Sort = "name"
	}
	if params.Order != "asc" && params.Order != "desc" {
		params.Order = "asc"
	}

	validRisks := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if params.Risk != "" && !validRisks[params.Risk] {
		return nil, &ValidationError{Field: "risk", Message: "invalid risk level"}
	}

	result, err := s.customerRepo.ListWithScores(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	items := make([]CustomerListItem, len(result.Customers))
	for i, c := range result.Customers {
		items[i] = CustomerListItem{
			ID:           c.ID,
			Name:         c.Name,
			Email:        c.Email,
			CompanyName:  c.CompanyName,
			MRRCents:     c.MRRCents,
			Source:       c.Source,
			LastSeenAt:   c.LastSeenAt,
			OverallScore: c.OverallScore,
			RiskLevel:    c.RiskLevel,
		}
	}

	return &CustomerListResponse{
		Customers: items,
		Pagination: PaginationMeta{
			Page:       result.Page,
			PerPage:    result.PerPage,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	}, nil
}

// CustomerDetail is the full detail response for a customer.
type CustomerDetail struct {
	Customer      CustomerInfo             `json:"customer"`
	HealthScore   *HealthScoreDetail       `json:"health_score"`
	Subscriptions []*SubscriptionInfo      `json:"subscriptions"`
	RecentEvents  []*EventInfo             `json:"recent_events"`
}

// CustomerInfo holds customer info fields.
type CustomerInfo struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	CompanyName string         `json:"company_name"`
	MRRCents    int            `json:"mrr_cents"`
	Currency    string         `json:"currency"`
	Source      string         `json:"source"`
	ExternalID  string         `json:"external_id"`
	FirstSeenAt *time.Time     `json:"first_seen_at"`
	LastSeenAt  *time.Time     `json:"last_seen_at"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
}

// HealthScoreDetail holds health score info with factor breakdown.
type HealthScoreDetail struct {
	OverallScore int                `json:"overall_score"`
	RiskLevel    string             `json:"risk_level"`
	Factors      map[string]float64 `json:"factors"`
	CalculatedAt time.Time          `json:"calculated_at"`
}

// SubscriptionInfo holds subscription info.
type SubscriptionInfo struct {
	ID                   uuid.UUID  `json:"id"`
	StripeSubscriptionID string     `json:"stripe_subscription_id"`
	Status               string     `json:"status"`
	PlanName             string     `json:"plan_name"`
	AmountCents          int        `json:"amount_cents"`
	Currency             string     `json:"currency"`
	Interval             string     `json:"interval"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end"`
}

// EventInfo holds event info.
type EventInfo struct {
	ID         uuid.UUID      `json:"id"`
	EventType  string         `json:"event_type"`
	Source     string         `json:"source"`
	OccurredAt time.Time      `json:"occurred_at"`
	Data       map[string]any `json:"data"`
}

// GetDetail returns the full detail for a customer.
func (s *CustomerService) GetDetail(ctx context.Context, customerID, orgID uuid.UUID) (*CustomerDetail, error) {
	customer, err := s.customerRepo.GetByIDAndOrg(ctx, customerID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, &NotFoundError{Resource: "customer", Message: "customer not found"}
	}

	var (
		healthScore   *repository.HealthScore
		subscriptions []*repository.StripeSubscription
		events        []*repository.CustomerEvent
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		healthScore, err = s.healthRepo.GetByCustomerID(gctx, customerID, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		subscriptions, err = s.subRepo.ListActiveByCustomer(gctx, customerID)
		return err
	})

	g.Go(func() error {
		var err error
		events, err = s.eventRepo.ListByCustomer(gctx, customerID, 10)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("load customer detail: %w", err)
	}

	detail := &CustomerDetail{
		Customer: CustomerInfo{
			ID:          customer.ID,
			Name:        customer.Name,
			Email:       customer.Email,
			CompanyName: customer.CompanyName,
			MRRCents:    customer.MRRCents,
			Currency:    customer.Currency,
			Source:      customer.Source,
			ExternalID:  customer.ExternalID,
			FirstSeenAt: customer.FirstSeenAt,
			LastSeenAt:  customer.LastSeenAt,
			Metadata:    customer.Metadata,
			CreatedAt:   customer.CreatedAt,
		},
	}

	if healthScore != nil {
		detail.HealthScore = &HealthScoreDetail{
			OverallScore: healthScore.OverallScore,
			RiskLevel:    healthScore.RiskLevel,
			Factors:      healthScore.Factors,
			CalculatedAt: healthScore.CalculatedAt,
		}
	}

	subInfos := make([]*SubscriptionInfo, len(subscriptions))
	for i, sub := range subscriptions {
		subInfos[i] = &SubscriptionInfo{
			ID:                   sub.ID,
			StripeSubscriptionID: sub.StripeSubscriptionID,
			Status:               sub.Status,
			PlanName:             sub.PlanName,
			AmountCents:          sub.AmountCents,
			Currency:             sub.Currency,
			Interval:             sub.Interval,
			CurrentPeriodEnd:     sub.CurrentPeriodEnd,
		}
	}
	detail.Subscriptions = subInfos

	eventInfos := make([]*EventInfo, len(events))
	for i, e := range events {
		eventInfos[i] = &EventInfo{
			ID:         e.ID,
			EventType:  e.EventType,
			Source:     e.Source,
			OccurredAt: e.OccurredAt,
			Data:       e.Data,
		}
	}
	detail.RecentEvents = eventInfos

	return detail, nil
}

// EventListResponse is the JSON response for event listing.
type EventListResponse struct {
	Events     []*EventInfo   `json:"events"`
	Pagination PaginationMeta `json:"pagination"`
}

// ListEvents returns a paginated list of events for a customer.
func (s *CustomerService) ListEvents(ctx context.Context, params repository.EventListParams) (*EventListResponse, error) {
	// Validate params
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 25
	}
	if params.PerPage > 100 {
		params.PerPage = 100
	}
	if !params.From.IsZero() && !params.To.IsZero() && params.From.After(params.To) {
		return nil, &ValidationError{Field: "from", Message: "from must be before to"}
	}

	// Verify customer exists and belongs to org
	customer, err := s.customerRepo.GetByIDAndOrg(ctx, params.CustomerID, params.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, &NotFoundError{Resource: "customer", Message: "customer not found"}
	}

	result, err := s.eventRepo.ListPaginated(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	eventInfos := make([]*EventInfo, len(result.Events))
	for i, e := range result.Events {
		eventInfos[i] = &EventInfo{
			ID:         e.ID,
			EventType:  e.EventType,
			Source:     e.Source,
			OccurredAt: e.OccurredAt,
			Data:       e.Data,
		}
	}

	return &EventListResponse{
		Events: eventInfos,
		Pagination: PaginationMeta{
			Page:       result.Page,
			PerPage:    result.PerPage,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	}, nil
}
