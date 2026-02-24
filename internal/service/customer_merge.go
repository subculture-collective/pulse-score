package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// DeduplicationResult contains stats from a deduplication run.
type DeduplicationResult struct {
	Merged  int `json:"merged"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

// CustomerMergeService handles merging HubSpot contacts with existing customers.
type CustomerMergeService struct {
	customers *repository.CustomerRepository
	contacts  *repository.HubSpotContactRepository
}

// NewCustomerMergeService creates a new CustomerMergeService.
func NewCustomerMergeService(
	customers *repository.CustomerRepository,
	contacts *repository.HubSpotContactRepository,
) *CustomerMergeService {
	return &CustomerMergeService{
		customers: customers,
		contacts:  contacts,
	}
}

// MergeOrCreateFromHubSpot finds an existing customer by email and merges HubSpot data,
// or creates a new customer if no match exists.
func (s *CustomerMergeService) MergeOrCreateFromHubSpot(ctx context.Context, orgID uuid.UUID, contact *repository.HubSpotContact) (*repository.Customer, error) {
	if contact.Email == "" {
		// No email — create as new hubspot-only customer
		return s.createFromHubSpot(ctx, orgID, contact)
	}

	existing, err := s.customers.GetByEmail(ctx, orgID, contact.Email)
	if err != nil {
		return nil, fmt.Errorf("lookup by email: %w", err)
	}

	if existing != nil {
		return s.mergeIntoExisting(ctx, existing, contact)
	}

	return s.createFromHubSpot(ctx, orgID, contact)
}

// mergeIntoExisting merges HubSpot data into an existing customer record.
func (s *CustomerMergeService) mergeIntoExisting(ctx context.Context, existing *repository.Customer, contact *repository.HubSpotContact) (*repository.Customer, error) {
	name := buildFullName(contact.FirstName, contact.LastName)

	// Name: use most recently updated (HubSpot wins if existing name is empty)
	if existing.Name == "" && name != "" {
		existing.Name = name
	}

	// CompanyName: prefer HubSpot (CRM is authoritative for company data)
	if contact.HubSpotCompanyID != "" {
		// Company name will be enriched later by EnrichCustomersWithCompanyData
	}

	// MRR: never overwrite — Stripe is source of truth for billing
	// existing.MRRCents stays as-is

	// Deep merge metadata with per-source namespacing
	if existing.Metadata == nil {
		existing.Metadata = map[string]any{}
	}

	existing.Metadata["hubspot"] = map[string]any{
		"contact_id":      contact.HubSpotContactID,
		"lifecycle_stage": contact.LifecycleStage,
		"lead_status":     contact.LeadStatus,
		"company_id":      contact.HubSpotCompanyID,
	}

	// Track sources
	sources := []string{}
	if existingSources, ok := existing.Metadata["sources"].([]any); ok {
		for _, src := range existingSources {
			if s, ok := src.(string); ok {
				sources = append(sources, s)
			}
		}
	} else if existing.Source != "" {
		sources = append(sources, existing.Source)
	}
	hasHubSpot := false
	for _, src := range sources {
		if src == "hubspot" {
			hasHubSpot = true
			break
		}
	}
	if !hasHubSpot {
		sources = append(sources, "hubspot")
	}
	existing.Metadata["sources"] = sources

	if err := s.customers.UpdateCompanyAndMetadata(ctx, existing.ID, existing.CompanyName, existing.Metadata); err != nil {
		return nil, fmt.Errorf("update merged customer: %w", err)
	}

	// Link HubSpot contact to this customer
	if err := s.contacts.LinkCustomer(ctx, contact.ID, existing.ID); err != nil {
		slog.Error("failed to link hubspot contact after merge", "error", err)
	}

	return existing, nil
}

// createFromHubSpot creates a new customer from a HubSpot contact.
func (s *CustomerMergeService) createFromHubSpot(ctx context.Context, orgID uuid.UUID, contact *repository.HubSpotContact) (*repository.Customer, error) {
	name := buildFullName(contact.FirstName, contact.LastName)

	customer := &repository.Customer{
		OrgID:      orgID,
		ExternalID: contact.HubSpotContactID,
		Source:     "hubspot",
		Email:      contact.Email,
		Name:       name,
		Metadata: map[string]any{
			"hubspot": map[string]any{
				"contact_id":      contact.HubSpotContactID,
				"lifecycle_stage": contact.LifecycleStage,
				"lead_status":     contact.LeadStatus,
				"company_id":      contact.HubSpotCompanyID,
			},
			"sources": []string{"hubspot"},
		},
	}

	if err := s.customers.UpsertByExternal(ctx, customer); err != nil {
		return nil, fmt.Errorf("create customer from hubspot: %w", err)
	}

	if err := s.contacts.LinkCustomer(ctx, contact.ID, customer.ID); err != nil {
		slog.Error("failed to link hubspot contact to new customer", "error", err)
	}

	return customer, nil
}

// DeduplicateCustomers finds and merges duplicate customers across sources.
func (s *CustomerMergeService) DeduplicateCustomers(ctx context.Context, orgID uuid.UUID) (*DeduplicationResult, error) {
	duplicates, err := s.customers.FindDuplicatesByEmail(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("find duplicates: %w", err)
	}

	result := &DeduplicationResult{}

	for _, group := range duplicates {
		if len(group) < 2 {
			continue
		}

		// Pick primary: oldest first_seen_at
		primary := group[0]
		for _, c := range group[1:] {
			if c.FirstSeenAt != nil && (primary.FirstSeenAt == nil || c.FirstSeenAt.Before(*primary.FirstSeenAt)) {
				primary = c
			}
		}

		// Merge metadata from others into primary
		if primary.Metadata == nil {
			primary.Metadata = map[string]any{}
		}

		for _, c := range group {
			if c.ID == primary.ID {
				continue
			}

			// Merge source-specific metadata
			if c.Metadata != nil {
				for k, v := range c.Metadata {
					if k == "sources" {
						continue
					}
					if _, exists := primary.Metadata[k]; !exists {
						primary.Metadata[k] = v
					}
				}
			}

			// Name: use most recently updated if primary is empty
			if primary.Name == "" && c.Name != "" {
				primary.Name = c.Name
			}

			// CompanyName: prefer HubSpot source
			if c.Source == "hubspot" && c.CompanyName != "" {
				primary.CompanyName = c.CompanyName
			}

			// MRR: prefer Stripe
			if c.Source == "stripe" && c.MRRCents > 0 {
				primary.MRRCents = c.MRRCents
				primary.Currency = c.Currency
			}
		}

		if err := s.customers.UpdateCompanyAndMetadata(ctx, primary.ID, primary.CompanyName, primary.Metadata); err != nil {
			slog.Error("failed to update merged primary customer", "id", primary.ID, "error", err)
			result.Errors++
			continue
		}

		result.Merged++
	}

	slog.Info("deduplication complete", "org_id", orgID, "merged", result.Merged, "skipped", result.Skipped, "errors", result.Errors)
	return result, nil
}
