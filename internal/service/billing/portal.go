package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	stripeportal "github.com/stripe/stripe-go/v81/billingportal/session"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"
	stripesubscription "github.com/stripe/stripe-go/v81/subscription"

	"github.com/onnwee/pulse-score/internal/repository"
	core "github.com/onnwee/pulse-score/internal/service"
)

type orgSubscriptionWriter interface {
	GetByOrg(ctx context.Context, orgID uuid.UUID) (*repository.OrgSubscription, error)
	UpsertByOrg(ctx context.Context, sub *repository.OrgSubscription) error
}

// PortalSessionResponse returns hosted Stripe customer portal URL.
type PortalSessionResponse struct {
	URL string `json:"url"`
}

// PortalService handles Stripe customer portal and cancellation operations.
type PortalService struct {
	stripeSecretKey string
	portalReturnURL string
	frontendURL     string
	orgs            orgBillingRepository
	subscriptions   orgSubscriptionWriter
}

func NewPortalService(
	stripeSecretKey, portalReturnURL, frontendURL string,
	orgs orgBillingRepository,
	subscriptions orgSubscriptionWriter,
) *PortalService {
	return &PortalService{
		stripeSecretKey: strings.TrimSpace(stripeSecretKey),
		portalReturnURL: strings.TrimSpace(portalReturnURL),
		frontendURL:     strings.TrimRight(strings.TrimSpace(frontendURL), "/"),
		orgs:            orgs,
		subscriptions:   subscriptions,
	}
}

func (s *PortalService) CreatePortalSession(ctx context.Context, orgID uuid.UUID) (*PortalSessionResponse, error) {
	if s.stripeSecretKey == "" {
		return nil, &core.ValidationError{Field: "billing", Message: "stripe billing is not configured"}
	}

	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	if org == nil {
		return nil, &core.NotFoundError{Resource: "organization", Message: "organization not found"}
	}

	customerID := strings.TrimSpace(org.StripeCustomerID)
	if customerID == "" {
		customerID, err = s.createStripeCustomer(ctx, org)
		if err != nil {
			return nil, fmt.Errorf("create stripe customer: %w", err)
		}
		if err := s.orgs.UpdateStripeCustomerID(ctx, orgID, customerID); err != nil {
			return nil, fmt.Errorf("persist stripe customer id: %w", err)
		}
	}

	returnURL := s.portalReturnURL
	if returnURL == "" {
		returnURL = fmt.Sprintf("%s/settings/billing", s.frontendURL)
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	client := stripeportal.Client{B: stripe.GetBackend(stripe.APIBackend), Key: s.stripeSecretKey}
	portalSession, err := client.New(params)
	if err != nil {
		return nil, fmt.Errorf("create portal session: %w", err)
	}

	return &PortalSessionResponse{URL: portalSession.URL}, nil
}

func (s *PortalService) CancelAtPeriodEnd(ctx context.Context, orgID uuid.UUID) error {
	if s.stripeSecretKey == "" {
		return &core.ValidationError{Field: "billing", Message: "stripe billing is not configured"}
	}

	sub, err := s.subscriptions.GetByOrg(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}
	if sub == nil || strings.TrimSpace(sub.StripeSubscriptionID) == "" {
		return &core.NotFoundError{Resource: "subscription", Message: "no active Stripe subscription found"}
	}

	params := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(true)}
	client := stripesubscription.Client{B: stripe.GetBackend(stripe.APIBackend), Key: s.stripeSecretKey}
	stripeSub, err := client.Update(sub.StripeSubscriptionID, params)
	if err != nil {
		return fmt.Errorf("cancel subscription at period end: %w", err)
	}

	sub.CancelAtPeriodEnd = true
	sub.Status = string(stripeSub.Status)
	if stripeSub.CurrentPeriodStart > 0 {
		t := time.Unix(stripeSub.CurrentPeriodStart, 0)
		sub.CurrentPeriodStart = &t
	}
	if stripeSub.CurrentPeriodEnd > 0 {
		t := time.Unix(stripeSub.CurrentPeriodEnd, 0)
		sub.CurrentPeriodEnd = &t
	}

	if err := s.subscriptions.UpsertByOrg(ctx, sub); err != nil {
		return fmt.Errorf("persist subscription cancellation state: %w", err)
	}

	return nil
}

func (s *PortalService) createStripeCustomer(ctx context.Context, org *repository.Organization) (string, error) {
	params := &stripe.CustomerParams{
		Name: stripe.String(org.Name),
		Metadata: map[string]string{
			"org_id": org.ID.String(),
		},
	}

	client := stripecustomer.Client{B: stripe.GetBackend(stripe.APIBackend), Key: s.stripeSecretKey}
	cust, err := client.New(params)
	if err != nil {
		return "", err
	}

	return cust.ID, nil
}
