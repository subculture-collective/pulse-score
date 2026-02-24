package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// HubSpotWebhookEvent represents an incoming HubSpot webhook event.
type HubSpotWebhookEvent struct {
	EventID          int64  `json:"eventId"`
	SubscriptionID   int64  `json:"subscriptionId"`
	PortalID         int64  `json:"portalId"`
	AppID            int64  `json:"appId"`
	OccurredAt       int64  `json:"occurredAt"`
	SubscriptionType string `json:"subscriptionType"`
	AttemptNumber    int    `json:"attemptNumber"`
	ObjectID         int64  `json:"objectId"`
	PropertyName     string `json:"propertyName"`
	PropertyValue    string `json:"propertyValue"`
}

// HubSpotWebhookService handles incoming HubSpot webhook events.
type HubSpotWebhookService struct {
	clientSecret string
	syncSvc      *HubSpotSyncService
	mergeSvc     *CustomerMergeService
	connRepo     *repository.IntegrationConnectionRepository
	contacts     *repository.HubSpotContactRepository
	deals        *repository.HubSpotDealRepository
	companies    *repository.HubSpotCompanyRepository
	events       *repository.CustomerEventRepository

	processedEvents map[int64]time.Time
	mu              sync.Mutex
}

// NewHubSpotWebhookService creates a new HubSpotWebhookService.
func NewHubSpotWebhookService(
	clientSecret string,
	syncSvc *HubSpotSyncService,
	mergeSvc *CustomerMergeService,
	connRepo *repository.IntegrationConnectionRepository,
	contacts *repository.HubSpotContactRepository,
	deals *repository.HubSpotDealRepository,
	companies *repository.HubSpotCompanyRepository,
	events *repository.CustomerEventRepository,
) *HubSpotWebhookService {
	return &HubSpotWebhookService{
		clientSecret:    clientSecret,
		syncSvc:         syncSvc,
		mergeSvc:        mergeSvc,
		connRepo:        connRepo,
		contacts:        contacts,
		deals:           deals,
		companies:       companies,
		events:          events,
		processedEvents: make(map[int64]time.Time),
	}
}

// VerifySignature verifies the HubSpot webhook signature (v3).
// Signature = HMAC-SHA256(clientSecret, httpMethod + requestURI + requestBody + timestamp)
func (s *HubSpotWebhookService) VerifySignature(requestBody []byte, signatureHeader, timestamp, httpMethod, requestURI string) error {
	if signatureHeader == "" {
		return &ValidationError{Field: "signature", Message: "missing signature header"}
	}
	if timestamp == "" {
		return &ValidationError{Field: "timestamp", Message: "missing timestamp header"}
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return &ValidationError{Field: "timestamp", Message: "invalid timestamp"}
	}

	// Replay protection: timestamp must be within 5 minutes
	now := time.Now().UnixMilli()
	diff := math.Abs(float64(now - ts))
	if diff > 5*60*1000 {
		return &ValidationError{Field: "timestamp", Message: "timestamp expired"}
	}

	message := httpMethod + requestURI + string(requestBody) + timestamp
	mac := hmac.New(sha256.New, []byte(s.clientSecret))
	mac.Write([]byte(message))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(signatureHeader)) {
		return &ValidationError{Field: "signature", Message: "invalid webhook signature"}
	}

	return nil
}

// ProcessEvents processes a batch of HubSpot webhook events.
func (s *HubSpotWebhookService) ProcessEvents(ctx context.Context, webhookEvents []HubSpotWebhookEvent) error {
	for _, event := range webhookEvents {
		if s.isProcessed(event.EventID) {
			slog.Debug("duplicate hubspot webhook event skipped", "event_id", event.EventID)
			continue
		}

		portalIDStr := strconv.FormatInt(event.PortalID, 10)
		conn, err := s.connRepo.GetByProviderAndExternalID(ctx, "hubspot", portalIDStr)
		if err != nil {
			slog.Error("failed to lookup hubspot connection", "portal_id", event.PortalID, "error", err)
			continue
		}
		if conn == nil {
			slog.Warn("no hubspot connection for portal", "portal_id", event.PortalID)
			continue
		}

		if err := s.processEvent(ctx, conn.OrgID, event); err != nil {
			slog.Error("failed to process hubspot event",
				"event_id", event.EventID,
				"type", event.SubscriptionType,
				"error", err,
			)
		}

		s.markProcessed(event.EventID)
	}

	return nil
}

func (s *HubSpotWebhookService) processEvent(ctx context.Context, orgID uuid.UUID, event HubSpotWebhookEvent) error {
	switch event.SubscriptionType {
	case "contact.creation", "contact.propertyChange":
		return s.handleContactChange(ctx, orgID, event)
	case "deal.creation":
		return s.handleDealCreation(ctx, orgID, event)
	case "deal.propertyChange":
		return s.handleDealPropertyChange(ctx, orgID, event)
	case "company.propertyChange":
		return s.handleCompanyPropertyChange(ctx, orgID, event)
	default:
		slog.Debug("ignoring unknown hubspot event type", "type", event.SubscriptionType)
		return nil
	}
}

func (s *HubSpotWebhookService) handleContactChange(ctx context.Context, orgID uuid.UUID, event HubSpotWebhookEvent) error {
	objectIDStr := strconv.FormatInt(event.ObjectID, 10)

	hsContact, err := s.contacts.GetByHubSpotID(ctx, orgID, objectIDStr)
	if err != nil {
		return err
	}

	if hsContact == nil {
		hsContact = &repository.HubSpotContact{
			OrgID:            orgID,
			HubSpotContactID: objectIDStr,
			Metadata:         map[string]any{},
		}
	}

	switch event.PropertyName {
	case "email":
		hsContact.Email = event.PropertyValue
	case "firstname":
		hsContact.FirstName = event.PropertyValue
	case "lastname":
		hsContact.LastName = event.PropertyValue
	case "lifecyclestage":
		hsContact.LifecycleStage = event.PropertyValue
	case "hs_lead_status":
		hsContact.LeadStatus = event.PropertyValue
	}

	if err := s.contacts.Upsert(ctx, hsContact); err != nil {
		return err
	}

	if _, err := s.mergeSvc.MergeOrCreateFromHubSpot(ctx, orgID, hsContact); err != nil {
		slog.Error("failed to merge hubspot contact", "hubspot_id", objectIDStr, "error", err)
	}

	slog.Info("hubspot contact updated via webhook", "object_id", event.ObjectID, "property", event.PropertyName)
	return nil
}

func (s *HubSpotWebhookService) handleDealCreation(ctx context.Context, orgID uuid.UUID, event HubSpotWebhookEvent) error {
	objectIDStr := strconv.FormatInt(event.ObjectID, 10)

	hsDeal := &repository.HubSpotDeal{
		OrgID:         orgID,
		HubSpotDealID: objectIDStr,
		Metadata:      map[string]any{},
	}

	if err := s.deals.Upsert(ctx, hsDeal); err != nil {
		return err
	}

	if hsDeal.CustomerID != nil {
		customerEvent := &repository.CustomerEvent{
			OrgID:           orgID,
			CustomerID:      *hsDeal.CustomerID,
			EventType:       "deal_created",
			Source:          "hubspot",
			ExternalEventID: "webhook_deal_" + objectIDStr,
			OccurredAt:      time.UnixMilli(event.OccurredAt),
			Data: map[string]any{
				"deal_id": objectIDStr,
			},
		}
		if err := s.events.Upsert(ctx, customerEvent); err != nil {
			slog.Error("failed to create deal creation event", "error", err)
		}
	}

	slog.Info("hubspot deal created via webhook", "object_id", event.ObjectID)
	return nil
}

func (s *HubSpotWebhookService) handleDealPropertyChange(ctx context.Context, orgID uuid.UUID, event HubSpotWebhookEvent) error {
	objectIDStr := strconv.FormatInt(event.ObjectID, 10)

	deals, err := s.deals.GetByOrgID(ctx, orgID)
	if err != nil {
		return err
	}

	var hsDeal *repository.HubSpotDeal
	for i, d := range deals {
		if d.HubSpotDealID == objectIDStr {
			hsDeal = &deals[i]
			break
		}
	}

	if hsDeal == nil {
		hsDeal = &repository.HubSpotDeal{
			OrgID:         orgID,
			HubSpotDealID: objectIDStr,
			Metadata:      map[string]any{},
		}
	}

	switch event.PropertyName {
	case "dealname":
		hsDeal.DealName = event.PropertyValue
	case "dealstage":
		hsDeal.Stage = event.PropertyValue
	case "amount":
		hsDeal.AmountCents = parseAmountToCents(event.PropertyValue)
	case "pipeline":
		hsDeal.Pipeline = event.PropertyValue
	}

	if err := s.deals.Upsert(ctx, hsDeal); err != nil {
		return err
	}

	if event.PropertyName == "dealstage" && hsDeal.CustomerID != nil {
		customerEvent := &repository.CustomerEvent{
			OrgID:           orgID,
			CustomerID:      *hsDeal.CustomerID,
			EventType:       "deal_stage_change",
			Source:          "hubspot",
			ExternalEventID: "webhook_deal_" + objectIDStr + "_" + event.PropertyValue,
			OccurredAt:      time.UnixMilli(event.OccurredAt),
			Data: map[string]any{
				"deal_id":   objectIDStr,
				"deal_name": hsDeal.DealName,
				"stage":     event.PropertyValue,
			},
		}
		if err := s.events.Upsert(ctx, customerEvent); err != nil {
			slog.Error("failed to create deal stage event", "error", err)
		}
	}

	slog.Info("hubspot deal updated via webhook", "object_id", event.ObjectID, "property", event.PropertyName)
	return nil
}

func (s *HubSpotWebhookService) handleCompanyPropertyChange(ctx context.Context, orgID uuid.UUID, event HubSpotWebhookEvent) error {
	objectIDStr := strconv.FormatInt(event.ObjectID, 10)

	hsCompany, err := s.companies.GetByHubSpotID(ctx, orgID, objectIDStr)
	if err != nil {
		return err
	}

	if hsCompany == nil {
		hsCompany = &repository.HubSpotCompany{
			OrgID:            orgID,
			HubSpotCompanyID: objectIDStr,
			Metadata:         map[string]any{},
		}
	}

	switch event.PropertyName {
	case "name":
		hsCompany.Name = event.PropertyValue
	case "domain":
		hsCompany.Domain = event.PropertyValue
	case "industry":
		hsCompany.Industry = event.PropertyValue
	case "numberofemployees":
		if n, err := strconv.Atoi(event.PropertyValue); err == nil {
			hsCompany.NumberOfEmployees = n
		}
	case "annualrevenue":
		hsCompany.AnnualRevenueCents = parseAmountToCents(event.PropertyValue)
	}

	if err := s.companies.Upsert(ctx, hsCompany); err != nil {
		return err
	}

	if err := s.syncSvc.EnrichCustomersWithCompanyData(ctx, orgID); err != nil {
		slog.Error("failed to re-enrich customers after company update", "error", err)
	}

	slog.Info("hubspot company updated via webhook", "object_id", event.ObjectID, "property", event.PropertyName)
	return nil
}

func (s *HubSpotWebhookService) isProcessed(eventID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for id, t := range s.processedEvents {
		if t.Before(cutoff) {
			delete(s.processedEvents, id)
		}
	}

	_, exists := s.processedEvents[eventID]
	return exists
}

func (s *HubSpotWebhookService) markProcessed(eventID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processedEvents[eventID] = time.Now()
}
