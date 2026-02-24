package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/time/rate"
)

const intercomBaseURL = "https://api.intercom.io"

// IntercomClient provides rate-limited access to the Intercom API.
type IntercomClient struct {
	client  *http.Client
	limiter *rate.Limiter
}

// NewIntercomClient creates a new IntercomClient with rate limiting.
func NewIntercomClient() *IntercomClient {
	return &IntercomClient{
		client:  &http.Client{},
		limiter: rate.NewLimiter(rate.Limit(15), 15), // ~1000 req/min
	}
}

// IntercomContactListResponse represents the Intercom contacts list API response.
type IntercomContactListResponse struct {
	Type       string              `json:"type"`
	Data       []IntercomAPIContact `json:"data"`
	TotalCount int                 `json:"total_count"`
	Pages      *IntercomPages      `json:"pages,omitempty"`
}

// IntercomAPIContact represents a contact from Intercom API.
type IntercomAPIContact struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Role              string `json:"role"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	ExternalID        string `json:"external_id"`
	CompanyID         string `json:"company_id,omitempty"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
	LastSeenAt        int64  `json:"last_seen_at"`
	SignedUpAt        int64  `json:"signed_up_at"`
	UnsubscribedEmail bool   `json:"unsubscribed_from_emails"`
}

// IntercomConversationListResponse represents the Intercom conversations list API response.
type IntercomConversationListResponse struct {
	Type          string                      `json:"type"`
	Conversations []IntercomAPIConversation   `json:"conversations"`
	TotalCount    int                         `json:"total_count"`
	Pages         *IntercomPages              `json:"pages,omitempty"`
}

// IntercomAPIConversation represents a conversation from Intercom API.
type IntercomAPIConversation struct {
	ID                  string                    `json:"id"`
	Type                string                    `json:"type"`
	Title               string                    `json:"title"`
	State               string                    `json:"state"`
	Open                bool                      `json:"open"`
	Read                bool                      `json:"read"`
	Priority            string                    `json:"priority"`
	CreatedAt           int64                     `json:"created_at"`
	UpdatedAt           int64                     `json:"updated_at"`
	WaitingSince        int64                     `json:"waiting_since"`
	SnoozedUntil        int64                     `json:"snoozed_until"`
	Contacts            *IntercomConvContacts     `json:"contacts"`
	Statistics          *IntercomConvStatistics   `json:"statistics"`
	ConversationRating  *IntercomConvRating       `json:"conversation_rating"`
}

// IntercomConvContacts holds conversation contacts.
type IntercomConvContacts struct {
	Type     string                    `json:"type"`
	Contacts []IntercomConvContact     `json:"contacts"`
}

// IntercomConvContact is a contact in a conversation.
type IntercomConvContact struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// IntercomConvStatistics holds conversation statistics.
type IntercomConvStatistics struct {
	TimeToAssignment        int64  `json:"time_to_assignment"`
	TimeToAdminReply        int64  `json:"time_to_admin_reply"`
	TimeToFirstClose        int64  `json:"time_to_first_close"`
	TimeToLastClose         int64  `json:"time_to_last_close"`
	FirstContactReplyAt     int64  `json:"first_contact_reply_at"`
	FirstAssignmentAt       int64  `json:"first_assignment_at"`
	FirstAdminReplyAt       int64  `json:"first_admin_reply_at"`
	FirstCloseAt            int64  `json:"first_close_at"`
	LastAssignmentAt        int64  `json:"last_assignment_at"`
	LastAssignmentAdminReplyAt int64 `json:"last_assignment_admin_reply_at"`
	LastContactReplyAt      int64  `json:"last_contact_reply_at"`
	LastAdminReplyAt        int64  `json:"last_admin_reply_at"`
	LastCloseAt             int64  `json:"last_close_at"`
	LastClosedByID          string `json:"last_closed_by_id"`
	CountReopens            int    `json:"count_reopens"`
	CountAssignments        int    `json:"count_assignments"`
	CountConversationParts  int    `json:"count_conversation_parts"`
}

// IntercomConvRating holds conversation rating data.
type IntercomConvRating struct {
	Rating int    `json:"rating"`
	Remark string `json:"remark"`
}

// IntercomPages holds pagination info from Intercom API.
type IntercomPages struct {
	Type       string           `json:"type"`
	Next       *IntercomPageRef `json:"next,omitempty"`
	Page       int              `json:"page"`
	PerPage    int              `json:"per_page"`
	TotalPages int              `json:"total_pages"`
}

// IntercomPageRef holds a reference to a page with a starting_after cursor.
type IntercomPageRef struct {
	Page         int    `json:"page"`
	StartingAfter string `json:"starting_after"`
}

// ListContacts fetches contacts from Intercom with cursor-based pagination.
func (c *IntercomClient) ListContacts(ctx context.Context, accessToken, startingAfter string) (*IntercomContactListResponse, error) {
	url := intercomBaseURL + "/contacts?per_page=50"
	if startingAfter != "" {
		url += "&starting_after=" + startingAfter
	}
	return intercomGet[IntercomContactListResponse](ctx, c, url, accessToken)
}

// ListContactsUpdatedSince fetches contacts updated since a timestamp (for incremental sync).
func (c *IntercomClient) ListContactsUpdatedSince(ctx context.Context, accessToken string, since int64, startingAfter string) (*IntercomContactListResponse, error) {
	url := fmt.Sprintf("%s/contacts?per_page=50&updated_since=%d", intercomBaseURL, since)
	if startingAfter != "" {
		url += "&starting_after=" + startingAfter
	}
	return intercomGet[IntercomContactListResponse](ctx, c, url, accessToken)
}

// ListConversations fetches conversations from Intercom with cursor-based pagination.
func (c *IntercomClient) ListConversations(ctx context.Context, accessToken, startingAfter string) (*IntercomConversationListResponse, error) {
	url := intercomBaseURL + "/conversations?per_page=50&display_as=plaintext"
	if startingAfter != "" {
		url += "&starting_after=" + startingAfter
	}
	return intercomGet[IntercomConversationListResponse](ctx, c, url, accessToken)
}

// ListConversationsUpdatedSince fetches conversations updated since a timestamp.
func (c *IntercomClient) ListConversationsUpdatedSince(ctx context.Context, accessToken string, since int64, startingAfter string) (*IntercomConversationListResponse, error) {
	url := fmt.Sprintf("%s/conversations?per_page=50&display_as=plaintext&updated_since=%d", intercomBaseURL, since)
	if startingAfter != "" {
		url += "&starting_after=" + startingAfter
	}
	return intercomGet[IntercomConversationListResponse](ctx, c, url, accessToken)
}

func intercomGet[T any](ctx context.Context, c *IntercomClient, url, accessToken string) (*T, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Intercom-Version", "2.11")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("intercom api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}
