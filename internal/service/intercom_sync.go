package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntercomSyncService handles syncing data from Intercom to local database.
type IntercomSyncService struct {
	oauthSvc      *IntercomOAuthService
	client        *IntercomClient
	contacts      *repository.IntercomContactRepository
	conversations *repository.IntercomConversationRepository
	customers     *repository.CustomerRepository
	events        *repository.CustomerEventRepository
}

type intercomContactPageFetcher func(ctx context.Context, accessToken, cursor string) (*IntercomContactListResponse, error)
type intercomConversationPageFetcher func(ctx context.Context, accessToken, cursor string) (*IntercomConversationListResponse, error)

// NewIntercomSyncService creates a new IntercomSyncService.
func NewIntercomSyncService(
	oauthSvc *IntercomOAuthService,
	client *IntercomClient,
	contacts *repository.IntercomContactRepository,
	conversations *repository.IntercomConversationRepository,
	customers *repository.CustomerRepository,
	events *repository.CustomerEventRepository,
) *IntercomSyncService {
	return &IntercomSyncService{
		oauthSvc:      oauthSvc,
		client:        client,
		contacts:      contacts,
		conversations: conversations,
		customers:     customers,
		events:        events,
	}
}

// SyncContacts fetches all contacts from Intercom and upserts them locally.
func (s *IntercomSyncService) SyncContacts(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	return s.syncContacts(ctx, orgID, "intercom_contacts", "list contacts", true, true, s.client.ListContacts)
}

// SyncConversations fetches all conversations from Intercom and upserts them locally.
func (s *IntercomSyncService) SyncConversations(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	return s.syncConversations(
		ctx,
		orgID,
		"intercom_conversations",
		"list conversations",
		true,
		true,
		true,
		s.client.ListConversations,
	)
}

// SyncContactsSince fetches contacts updated since the given time (incremental sync).
func (s *IntercomSyncService) SyncContactsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	fetchPage := func(ctx context.Context, accessToken, cursor string) (*IntercomContactListResponse, error) {
		return s.client.ListContactsUpdatedSince(ctx, accessToken, since.Unix(), cursor)
	}

	return s.syncContacts(
		ctx,
		orgID,
		"intercom_contacts_incremental",
		"list contacts since",
		false,
		false,
		fetchPage,
	)
}

func (s *IntercomSyncService) syncContacts(
	ctx context.Context,
	orgID uuid.UUID,
	step string,
	listErrorPrefix string,
	logUpsertErrors bool,
	logCompletion bool,
	fetchPage intercomContactPageFetcher,
) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: step}
	cursor := ""

	for {
		resp, err := fetchPage(ctx, accessToken, cursor)
		if err != nil {
			return progress, fmt.Errorf("%s: %w", listErrorPrefix, err)
		}

		for _, c := range resp.Data {
			progress.Total++

			if err := s.upsertContactAndCustomer(ctx, orgID, c, logUpsertErrors); err != nil {
				progress.Errors++
				continue
			}

			progress.Current++
		}

		nextCursor := nextIntercomCursor(resp.Pages)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	if logCompletion {
		slog.Info("intercom contact sync complete",
			"org_id", orgID,
			"total", progress.Total,
			"synced", progress.Current,
			"errors", progress.Errors,
		)
	}

	return progress, nil
}

func (s *IntercomSyncService) upsertContactAndCustomer(ctx context.Context, orgID uuid.UUID, c IntercomAPIContact, logUpsertErrors bool) error {
	icContact := &repository.IntercomContact{
		OrgID:             orgID,
		IntercomContactID: c.ID,
		Email:             c.Email,
		Name:              c.Name,
		Role:              c.Role,
		IntercomCompanyID: c.CompanyID,
		Metadata:          map[string]any{},
	}

	if err := s.contacts.Upsert(ctx, icContact); err != nil {
		if logUpsertErrors {
			slog.Error("failed to upsert intercom contact", "intercom_id", c.ID, "error", err)
		}
		return err
	}

	now := time.Now()
	localCustomer := &repository.Customer{
		OrgID:       orgID,
		ExternalID:  c.ID,
		Source:      "intercom",
		Email:       c.Email,
		Name:        c.Name,
		FirstSeenAt: &now,
		LastSeenAt:  &now,
		Metadata: map[string]any{
			"intercom": map[string]any{
				"role":       c.Role,
				"company_id": c.CompanyID,
			},
		},
	}

	if err := s.customers.UpsertByExternal(ctx, localCustomer); err != nil {
		if logUpsertErrors {
			slog.Error("failed to upsert customer from intercom", "intercom_id", c.ID, "error", err)
		}
		return err
	}

	if err := s.contacts.LinkCustomer(ctx, icContact.ID, localCustomer.ID); err != nil {
		slog.Error("failed to link intercom contact to customer", "error", err)
	}

	return nil
}

// SyncConversationsSince fetches conversations updated since the given time (incremental sync).
func (s *IntercomSyncService) SyncConversationsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	fetchPage := func(ctx context.Context, accessToken, cursor string) (*IntercomConversationListResponse, error) {
		return s.client.ListConversationsUpdatedSince(ctx, accessToken, since.Unix(), cursor)
	}

	return s.syncConversations(
		ctx,
		orgID,
		"intercom_conversations_incremental",
		"list conversations since",
		false,
		false,
		false,
		fetchPage,
	)
}

func (s *IntercomSyncService) syncConversations(
	ctx context.Context,
	orgID uuid.UUID,
	step string,
	listErrorPrefix string,
	emitEvents bool,
	logUpsertErrors bool,
	logCompletion bool,
	fetchPage intercomConversationPageFetcher,
) (*SyncProgress, error) {
	accessToken, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	progress := &SyncProgress{Step: step}
	cursor := ""

	for {
		resp, err := fetchPage(ctx, accessToken, cursor)
		if err != nil {
			return progress, fmt.Errorf("%s: %w", listErrorPrefix, err)
		}

		for _, conv := range resp.Conversations {
			progress.Total++

			if err := s.upsertConversation(ctx, orgID, conv, emitEvents); err != nil {
				if logUpsertErrors {
					slog.Error("failed to upsert intercom conversation", "intercom_id", conv.ID, "error", err)
				}
				progress.Errors++
				continue
			}

			progress.Current++
		}

		nextCursor := nextIntercomCursor(resp.Pages)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	if logCompletion {
		slog.Info("intercom conversation sync complete",
			"org_id", orgID,
			"total", progress.Total,
			"synced", progress.Current,
			"errors", progress.Errors,
		)
	}

	return progress, nil
}

func (s *IntercomSyncService) upsertConversation(ctx context.Context, orgID uuid.UUID, conv IntercomAPIConversation, emitEvent bool) error {
	contactID := intercomConversationContactID(conv)
	customerID := s.resolveConversationCustomerID(ctx, orgID, contactID)

	icConv := mapIntercomConversation(orgID, conv, contactID, customerID)
	if err := s.conversations.Upsert(ctx, icConv); err != nil {
		return err
	}

	if emitEvent {
		s.emitConversationEvent(ctx, orgID, customerID, conv)
	}

	return nil
}

func (s *IntercomSyncService) resolveConversationCustomerID(ctx context.Context, orgID uuid.UUID, contactID string) *uuid.UUID {
	if contactID == "" {
		return nil
	}

	icContact, err := s.contacts.GetByIntercomID(ctx, orgID, contactID)
	if err != nil || icContact == nil || icContact.CustomerID == nil {
		return nil
	}

	return icContact.CustomerID
}

func mapIntercomConversation(orgID uuid.UUID, conv IntercomAPIConversation, contactID string, customerID *uuid.UUID) *repository.IntercomConversation {
	closedAt, firstResponseAt := intercomConversationTiming(conv.Statistics)
	rating, ratingRemark := intercomConversationRating(conv.ConversationRating)

	return &repository.IntercomConversation{
		OrgID:                  orgID,
		CustomerID:             customerID,
		IntercomConversationID: conv.ID,
		IntercomContactID:      contactID,
		State:                  conv.State,
		Rating:                 rating,
		RatingRemark:           ratingRemark,
		Open:                   conv.Open,
		Read:                   conv.Read,
		Priority:               conv.Priority,
		Subject:                conv.Title,
		CreatedAtRemote:        timeFromUnix(conv.CreatedAt),
		UpdatedAtRemote:        timeFromUnix(conv.UpdatedAt),
		ClosedAt:               closedAt,
		FirstResponseAt:        firstResponseAt,
		Metadata:               map[string]any{},
	}
}

func intercomConversationContactID(conv IntercomAPIConversation) string {
	if conv.Contacts == nil || len(conv.Contacts.Contacts) == 0 {
		return ""
	}

	return conv.Contacts.Contacts[0].ID
}

func intercomConversationTiming(stats *IntercomConvStatistics) (*time.Time, *time.Time) {
	if stats == nil {
		return nil, nil
	}

	var closedAt *time.Time
	if stats.FirstCloseAt > 0 {
		t := time.Unix(stats.FirstCloseAt, 0)
		closedAt = &t
	}

	var firstResponseAt *time.Time
	if stats.FirstAdminReplyAt > 0 {
		t := time.Unix(stats.FirstAdminReplyAt, 0)
		firstResponseAt = &t
	}

	return closedAt, firstResponseAt
}

func intercomConversationRating(rating *IntercomConvRating) (int, string) {
	if rating == nil {
		return 0, ""
	}

	return rating.Rating, rating.Remark
}

func (s *IntercomSyncService) emitConversationEvent(ctx context.Context, orgID uuid.UUID, customerID *uuid.UUID, conv IntercomAPIConversation) {
	if customerID == nil {
		return
	}

	event := &repository.CustomerEvent{
		OrgID:           orgID,
		CustomerID:      *customerID,
		EventType:       "conversation_" + conv.State,
		Source:          "intercom",
		ExternalEventID: "conv_" + conv.ID,
		OccurredAt:      time.Now(),
		Data: map[string]any{
			"conversation_id": conv.ID,
			"state":           conv.State,
			"priority":        conv.Priority,
			"subject":         conv.Title,
		},
	}

	if err := s.events.Upsert(ctx, event); err != nil {
		slog.Error("failed to create conversation event", "error", err)
	}
}

func nextIntercomCursor(pages *IntercomPages) string {
	if pages == nil || pages.Next == nil {
		return ""
	}

	return pages.Next.StartingAfter
}

func timeFromUnix(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}
