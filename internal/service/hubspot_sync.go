package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// HubSpotSyncService handles syncing data from HubSpot to local database.
type HubSpotSyncService struct {
	oauthSvc  *HubSpotOAuthService
	client    *HubSpotClient
	contacts  *repository.HubSpotContactRepository
	deals     *repository.HubSpotDealRepository
	companies *repository.HubSpotCompanyRepository
	customers *repository.CustomerRepository
	events    *repository.CustomerEventRepository
}

type hubspotContactPageFetcher func(ctx context.Context, accessToken, after string) (*HubSpotContactListResponse, error)
type hubspotDealPageFetcher func(ctx context.Context, accessToken, after string) (*HubSpotDealListResponse, error)

// NewHubSpotSyncService creates a new HubSpotSyncService.
func NewHubSpotSyncService(
	oauthSvc *HubSpotOAuthService,
	client *HubSpotClient,
	contacts *repository.HubSpotContactRepository,
	deals *repository.HubSpotDealRepository,
	companies *repository.HubSpotCompanyRepository,
	customers *repository.CustomerRepository,
	events *repository.CustomerEventRepository,
) *HubSpotSyncService {
	return &HubSpotSyncService{
		oauthSvc:  oauthSvc,
		client:    client,
		contacts:  contacts,
		deals:     deals,
		companies: companies,
		customers: customers,
		events:    events,
	}
}

// SyncContacts fetches all contacts from HubSpot and upserts them locally.
func (s *HubSpotSyncService) SyncContacts(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	return s.syncContacts(ctx, orgID, "hubspot_contacts", "list contacts", true, true, s.client.ListContacts)
}

// SyncDeals fetches all deals from HubSpot and upserts them locally.
func (s *HubSpotSyncService) SyncDeals(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	return s.syncDeals(ctx, orgID, "hubspot_deals", "list deals", true, true, s.client.ListDeals)
}

// SyncCompanies fetches all companies from HubSpot and upserts them locally.
func (s *HubSpotSyncService) SyncCompanies(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: "hubspot_companies"}
	after := ""

	for {
		resp, err := s.client.ListCompanies(ctx, accessToken, after)
		if err != nil {
			return progress, fmt.Errorf("list companies: %w", err)
		}

		for _, c := range resp.Results {
			progress.Total++

			numEmployees := 0
			if c.Properties.NumberOfEmployees != "" {
				if n, err := strconv.Atoi(c.Properties.NumberOfEmployees); err == nil {
					numEmployees = n
				}
			}

			revenueCents := parseAmountToCents(c.Properties.AnnualRevenue)

			hsCompany := &repository.HubSpotCompany{
				OrgID:              orgID,
				HubSpotCompanyID:   c.ID,
				Name:               c.Properties.Name,
				Domain:             c.Properties.Domain,
				Industry:           c.Properties.Industry,
				NumberOfEmployees:  numEmployees,
				AnnualRevenueCents: revenueCents,
				Metadata:           map[string]any{},
			}

			if err := s.companies.Upsert(ctx, hsCompany); err != nil {
				slog.Error("failed to upsert hubspot company", "hubspot_id", c.ID, "error", err)
				progress.Errors++
				continue
			}

			progress.Current++
		}

		if resp.Paging == nil || resp.Paging.Next == nil || resp.Paging.Next.After == "" {
			break
		}
		after = resp.Paging.Next.After
	}

	slog.Info("hubspot company sync complete",
		"org_id", orgID,
		"total", progress.Total,
		"synced", progress.Current,
		"errors", progress.Errors,
	)

	return progress, nil
}

// EnrichCustomersWithCompanyData enriches customer records with company data from HubSpot.
func (s *HubSpotSyncService) EnrichCustomersWithCompanyData(ctx context.Context, orgID uuid.UUID) error {
	contacts, err := s.contacts.GetByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list hubspot contacts: %w", err)
	}

	enriched := 0
	for _, contact := range contacts {
		if contact.HubSpotCompanyID == "" || contact.CustomerID == nil {
			continue
		}

		company, err := s.companies.GetByHubSpotID(ctx, orgID, contact.HubSpotCompanyID)
		if err != nil {
			slog.Error("failed to lookup hubspot company", "company_id", contact.HubSpotCompanyID, "error", err)
			continue
		}
		if company == nil {
			continue
		}

		metadata := map[string]any{
			"hubspot_company": map[string]any{
				"industry":            company.Industry,
				"number_of_employees": company.NumberOfEmployees,
				"annual_revenue":      company.AnnualRevenueCents,
				"domain":              company.Domain,
			},
		}

		if err := s.customers.UpdateCompanyAndMetadata(ctx, *contact.CustomerID, company.Name, metadata); err != nil {
			slog.Error("failed to enrich customer with company data", "customer_id", contact.CustomerID, "error", err)
			continue
		}
		enriched++
	}

	slog.Info("customer enrichment complete", "org_id", orgID, "enriched", enriched)
	return nil
}

// SyncContactsSince fetches contacts modified since the given time (incremental sync).
func (s *HubSpotSyncService) SyncContactsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	filterGroups := []HubSpotFilterGroup{{
		Filters: []HubSpotFilter{{
			PropertyName: "lastmodifieddate",
			Operator:     "GTE",
			Value:        fmt.Sprintf("%d", since.UnixMilli()),
		}},
	}}
	fetchPage := func(ctx context.Context, accessToken, after string) (*HubSpotContactListResponse, error) {
		return s.client.SearchContacts(ctx, accessToken, filterGroups, after)
	}

	return s.syncContacts(
		ctx,
		orgID,
		"hubspot_contacts_incremental",
		"search contacts",
		false,
		false,
		fetchPage,
	)
}

func (s *HubSpotSyncService) syncContacts(
	ctx context.Context,
	orgID uuid.UUID,
	step string,
	listErrorPrefix string,
	logUpsertErrors bool,
	logCompletion bool,
	fetchPage hubspotContactPageFetcher,
) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: step}
	after := ""

	for {
		resp, err := fetchPage(ctx, accessToken, after)
		if err != nil {
			return progress, fmt.Errorf("%s: %w", listErrorPrefix, err)
		}

		for _, c := range resp.Results {
			progress.Total++

			if err := s.upsertContactAndCustomer(ctx, orgID, c, logUpsertErrors); err != nil {
				progress.Errors++
				continue
			}

			progress.Current++
		}

		if resp.Paging == nil || resp.Paging.Next == nil || resp.Paging.Next.After == "" {
			break
		}
		after = resp.Paging.Next.After
	}

	if logCompletion {
		slog.Info("hubspot contact sync complete",
			"org_id", orgID,
			"total", progress.Total,
			"synced", progress.Current,
			"errors", progress.Errors,
		)
	}

	return progress, nil
}

// SyncDealsSince fetches deals modified since the given time (incremental sync).
func (s *HubSpotSyncService) SyncDealsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	filterGroups := []HubSpotFilterGroup{{
		Filters: []HubSpotFilter{{
			PropertyName: "hs_lastmodifieddate",
			Operator:     "GTE",
			Value:        fmt.Sprintf("%d", since.UnixMilli()),
		}},
	}}
	fetchPage := func(ctx context.Context, accessToken, after string) (*HubSpotDealListResponse, error) {
		return s.client.SearchDeals(ctx, accessToken, filterGroups, after)
	}

	return s.syncDeals(
		ctx,
		orgID,
		"hubspot_deals_incremental",
		"search deals",
		false,
		false,
		fetchPage,
	)
}

func (s *HubSpotSyncService) syncDeals(
	ctx context.Context,
	orgID uuid.UUID,
	step string,
	listErrorPrefix string,
	logUpsertErrors bool,
	logCompletion bool,
	fetchPage hubspotDealPageFetcher,
) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: step}
	after := ""

	for {
		resp, err := fetchPage(ctx, accessToken, after)
		if err != nil {
			return progress, fmt.Errorf("%s: %w", listErrorPrefix, err)
		}

		for _, d := range resp.Results {
			progress.Total++

			if err := s.upsertDeal(ctx, orgID, d, logUpsertErrors); err != nil {
				progress.Errors++
				continue
			}

			progress.Current++
		}

		if resp.Paging == nil || resp.Paging.Next == nil || resp.Paging.Next.After == "" {
			break
		}
		after = resp.Paging.Next.After
	}

	if logCompletion {
		slog.Info("hubspot deal sync complete",
			"org_id", orgID,
			"total", progress.Total,
			"synced", progress.Current,
			"errors", progress.Errors,
		)
	}

	return progress, nil
}

func (s *HubSpotSyncService) upsertContactAndCustomer(ctx context.Context, orgID uuid.UUID, c HubSpotAPIContact, logUpsertErrors bool) error {
	name := buildFullName(c.Properties.FirstName, c.Properties.LastName)

	hsContact := &repository.HubSpotContact{
		OrgID:            orgID,
		HubSpotContactID: c.ID,
		Email:            c.Properties.Email,
		FirstName:        c.Properties.FirstName,
		LastName:         c.Properties.LastName,
		HubSpotCompanyID: c.Properties.AssociatedCompanyID,
		LifecycleStage:   c.Properties.LifecycleStage,
		LeadStatus:       c.Properties.LeadStatus,
		Metadata:         map[string]any{},
	}

	if err := s.contacts.Upsert(ctx, hsContact); err != nil {
		if logUpsertErrors {
			slog.Error("failed to upsert hubspot contact", "hubspot_id", c.ID, "error", err)
		}
		return err
	}

	now := time.Now()
	localCustomer := &repository.Customer{
		OrgID:       orgID,
		ExternalID:  c.ID,
		Source:      "hubspot",
		Email:       c.Properties.Email,
		Name:        name,
		CompanyName: c.Properties.Company,
		FirstSeenAt: &now,
		LastSeenAt:  &now,
		Metadata: map[string]any{
			"hubspot": map[string]any{
				"lifecycle_stage": c.Properties.LifecycleStage,
				"lead_status":     c.Properties.LeadStatus,
			},
		},
	}

	if err := s.customers.UpsertByExternal(ctx, localCustomer); err != nil {
		if logUpsertErrors {
			slog.Error("failed to upsert customer from hubspot", "hubspot_id", c.ID, "error", err)
		}
		return err
	}

	if err := s.contacts.LinkCustomer(ctx, hsContact.ID, localCustomer.ID); err != nil {
		slog.Error("failed to link hubspot contact to customer", "error", err)
	}

	return nil
}

func (s *HubSpotSyncService) upsertDeal(ctx context.Context, orgID uuid.UUID, d HubSpotAPIDeal, logUpsertErrors bool) error {
	amountCents := parseAmountToCents(d.Properties.Amount)
	closeDate := parseHubSpotDate(d.Properties.CloseDate)

	contactID := hubSpotDealContactID(d)
	customerID := s.resolveHubSpotDealCustomerID(ctx, orgID, contactID)

	hsDeal := &repository.HubSpotDeal{
		OrgID:            orgID,
		CustomerID:       customerID,
		HubSpotDealID:    d.ID,
		HubSpotContactID: contactID,
		DealName:         d.Properties.DealName,
		Stage:            d.Properties.DealStage,
		AmountCents:      amountCents,
		Currency:         "USD",
		CloseDate:        closeDate,
		Pipeline:         d.Properties.Pipeline,
		Metadata:         map[string]any{},
	}

	if err := s.deals.Upsert(ctx, hsDeal); err != nil {
		if logUpsertErrors {
			slog.Error("failed to upsert hubspot deal", "hubspot_id", d.ID, "error", err)
		}
		return err
	}

	s.emitDealStageEvent(ctx, orgID, customerID, d, amountCents)
	return nil
}

func hubSpotDealContactID(d HubSpotAPIDeal) string {
	if d.Associations == nil || d.Associations.Contacts == nil || len(d.Associations.Contacts.Results) == 0 {
		return ""
	}

	return d.Associations.Contacts.Results[0].ID
}

func (s *HubSpotSyncService) resolveHubSpotDealCustomerID(ctx context.Context, orgID uuid.UUID, contactID string) *uuid.UUID {
	if contactID == "" {
		return nil
	}

	hsContact, err := s.contacts.GetByHubSpotID(ctx, orgID, contactID)
	if err != nil || hsContact == nil || hsContact.CustomerID == nil {
		return nil
	}

	return hsContact.CustomerID
}

func (s *HubSpotSyncService) emitDealStageEvent(ctx context.Context, orgID uuid.UUID, customerID *uuid.UUID, d HubSpotAPIDeal, amountCents int64) {
	if customerID == nil {
		return
	}

	event := &repository.CustomerEvent{
		OrgID:           orgID,
		CustomerID:      *customerID,
		EventType:       "deal_stage_change",
		Source:          "hubspot",
		ExternalEventID: "deal_" + d.ID + "_" + d.Properties.DealStage,
		OccurredAt:      time.Now(),
		Data: map[string]any{
			"deal_name":    d.Properties.DealName,
			"stage":        d.Properties.DealStage,
			"amount_cents": amountCents,
			"pipeline":     d.Properties.Pipeline,
		},
	}

	if err := s.events.Upsert(ctx, event); err != nil {
		slog.Error("failed to create deal event", "error", err)
	}
}

func buildFullName(first, last string) string {
	name := first
	if last != "" {
		if name != "" {
			name += " "
		}
		name += last
	}
	return name
}

func parseAmountToCents(amount string) int64 {
	if amount == "" {
		return 0
	}
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0
	}
	return int64(f * 100)
}

func parseHubSpotDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	// HubSpot dates can be in ISO 8601 format
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		// Try date-only format
		t, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil
		}
	}
	return &t
}
