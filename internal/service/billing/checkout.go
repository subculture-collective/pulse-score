package billing

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
	core "github.com/onnwee/pulse-score/internal/service"
)

type orgBillingRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Organization, error)
	UpdateStripeCustomerID(ctx context.Context, orgID uuid.UUID, stripeCustomerID string) error
}

// CreateCheckoutSessionRequest defines supported inputs for checkout creation.
type CreateCheckoutSessionRequest struct {
	PriceID string `json:"priceId"`
	Tier    string `json:"tier"`
	Cycle   string `json:"cycle"`
	Annual  bool   `json:"annual"`
}

// CreateCheckoutSessionResponse returns the Stripe hosted checkout URL.
type CreateCheckoutSessionResponse struct {
	URL string `json:"url"`
}

// CheckoutService handles Stripe checkout session creation for PulseScore billing.
type CheckoutService struct {
	stripeSecretKey string
	frontendURL     string
	orgs            orgBillingRepository
	catalog         *planmodel.Catalog
}

func NewCheckoutService(
	stripeSecretKey, frontendURL string,
	orgs orgBillingRepository,
	catalog *planmodel.Catalog,
) *CheckoutService {
	return &CheckoutService{
		stripeSecretKey: strings.TrimSpace(stripeSecretKey),
		frontendURL:     strings.TrimRight(strings.TrimSpace(frontendURL), "/"),
		orgs:            orgs,
		catalog:         catalog,
	}
}

func (s *CheckoutService) CreateCheckoutSession(ctx context.Context, orgID, userID uuid.UUID, req CreateCheckoutSessionRequest) (*CreateCheckoutSessionResponse, error) {
	if s.stripeSecretKey == "" {
		return nil, &core.ValidationError{Field: "billing", Message: "stripe billing is not configured"}
	}

	priceID, tier, cycle, err := s.resolvePriceDetails(req)
	if err != nil {
		return nil, err
	}

	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
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

	successURL := fmt.Sprintf("%s/settings/billing?checkout=success", s.frontendURL)
	cancelURL := fmt.Sprintf("%s/settings/billing?checkout=cancelled", s.frontendURL)

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		Customer:          stripe.String(customerID),
		ClientReferenceID: stripe.String(orgID.String()),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		Metadata: map[string]string{
			"org_id":  orgID.String(),
			"user_id": userID.String(),
			"tier":    string(tier),
			"cycle":   string(cycle),
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"org_id":  orgID.String(),
				"user_id": userID.String(),
				"tier":    string(tier),
				"cycle":   string(cycle),
			},
		},
	}

	client := checkoutsession.Client{B: stripe.GetBackend(stripe.APIBackend), Key: s.stripeSecretKey}
	session, err := client.New(params)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return &CreateCheckoutSessionResponse{URL: session.URL}, nil
}

func (s *CheckoutService) resolvePriceDetails(req CreateCheckoutSessionRequest) (string, planmodel.Tier, planmodel.BillingCycle, error) {
	if strings.TrimSpace(req.PriceID) != "" {
		tier, cycle, ok := s.catalog.ResolveTierAndCycleByPriceID(req.PriceID)
		if !ok {
			return "", "", "", &core.ValidationError{Field: "priceId", Message: "unsupported price id"}
		}
		return req.PriceID, tier, cycle, nil
	}

	tier := planmodel.NormalizeTier(req.Tier)
	if tier == planmodel.TierFree {
		return "", "", "", &core.ValidationError{Field: "tier", Message: "free tier does not require checkout"}
	}

	annual := req.Annual
	if strings.EqualFold(strings.TrimSpace(req.Cycle), string(planmodel.BillingCycleAnnual)) {
		annual = true
	}

	priceID, err := s.catalog.GetPriceID(string(tier), annual)
	if err != nil {
		return "", "", "", &core.ValidationError{Field: "tier", Message: err.Error()}
	}

	cycle := planmodel.BillingCycleMonthly
	if annual {
		cycle = planmodel.BillingCycleAnnual
	}

	return priceID, tier, cycle, nil
}

func (s *CheckoutService) createStripeCustomer(ctx context.Context, org *repository.Organization) (string, error) {
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
