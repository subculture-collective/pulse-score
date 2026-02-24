package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/time/rate"
)

const hubspotBaseURL = "https://api.hubapi.com"

// HubSpotClient provides rate-limited access to the HubSpot API.
type HubSpotClient struct {
	client  *http.Client
	limiter *rate.Limiter
}

// NewHubSpotClient creates a new HubSpotClient with rate limiting.
func NewHubSpotClient() *HubSpotClient {
	return &HubSpotClient{
		client:  &http.Client{},
		limiter: rate.NewLimiter(rate.Limit(10), 10), // 10 requests/second
	}
}

// HubSpotContactListResponse represents the HubSpot contacts list API response.
type HubSpotContactListResponse struct {
	Results []HubSpotAPIContact `json:"results"`
	Paging  *HubSpotPaging      `json:"paging,omitempty"`
}

// HubSpotAPIContact represents a contact from HubSpot API.
type HubSpotAPIContact struct {
	ID         string                      `json:"id"`
	Properties HubSpotContactProperties    `json:"properties"`
}

// HubSpotContactProperties holds contact properties from HubSpot.
type HubSpotContactProperties struct {
	Email              string `json:"email"`
	FirstName          string `json:"firstname"`
	LastName           string `json:"lastname"`
	Company            string `json:"company"`
	LifecycleStage     string `json:"lifecyclestage"`
	LeadStatus         string `json:"hs_lead_status"`
	AssociatedCompanyID string `json:"associatedcompanyid"`
	LastModifiedDate   string `json:"lastmodifieddate"`
}

// HubSpotDealListResponse represents the HubSpot deals list API response.
type HubSpotDealListResponse struct {
	Results []HubSpotAPIDeal `json:"results"`
	Paging  *HubSpotPaging   `json:"paging,omitempty"`
}

// HubSpotAPIDeal represents a deal from HubSpot API.
type HubSpotAPIDeal struct {
	ID           string                  `json:"id"`
	Properties   HubSpotDealProperties   `json:"properties"`
	Associations *HubSpotDealAssociations `json:"associations,omitempty"`
}

// HubSpotDealProperties holds deal properties from HubSpot.
type HubSpotDealProperties struct {
	DealName  string `json:"dealname"`
	DealStage string `json:"dealstage"`
	Amount    string `json:"amount"`
	CloseDate string `json:"closedate"`
	Pipeline  string `json:"pipeline"`
}

// HubSpotDealAssociations holds deal associations.
type HubSpotDealAssociations struct {
	Contacts *HubSpotAssociationResults `json:"contacts,omitempty"`
}

// HubSpotAssociationResults holds association results.
type HubSpotAssociationResults struct {
	Results []HubSpotAssociation `json:"results"`
}

// HubSpotAssociation represents a single association.
type HubSpotAssociation struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// HubSpotCompanyListResponse represents the HubSpot companies list API response.
type HubSpotCompanyListResponse struct {
	Results []HubSpotAPICompany `json:"results"`
	Paging  *HubSpotPaging      `json:"paging,omitempty"`
}

// HubSpotAPICompany represents a company from HubSpot API.
type HubSpotAPICompany struct {
	ID         string                    `json:"id"`
	Properties HubSpotCompanyProperties  `json:"properties"`
}

// HubSpotCompanyProperties holds company properties from HubSpot.
type HubSpotCompanyProperties struct {
	Name               string `json:"name"`
	Domain             string `json:"domain"`
	Industry           string `json:"industry"`
	NumberOfEmployees  string `json:"numberofemployees"`
	AnnualRevenue      string `json:"annualrevenue"`
}

// HubSpotPaging holds pagination info.
type HubSpotPaging struct {
	Next *HubSpotPagingNext `json:"next,omitempty"`
}

// HubSpotPagingNext holds the next page cursor.
type HubSpotPagingNext struct {
	After string `json:"after"`
	Link  string `json:"link"`
}

// HubSpotSearchRequest represents a HubSpot search API request.
type HubSpotSearchRequest struct {
	FilterGroups []HubSpotFilterGroup `json:"filterGroups"`
	Limit        int                  `json:"limit"`
	After        string               `json:"after,omitempty"`
	Properties   []string             `json:"properties"`
}

// HubSpotFilterGroup groups filters for search.
type HubSpotFilterGroup struct {
	Filters []HubSpotFilter `json:"filters"`
}

// HubSpotFilter is a single search filter.
type HubSpotFilter struct {
	PropertyName string `json:"propertyName"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
}

// ListContacts fetches contacts from HubSpot with pagination.
func (c *HubSpotClient) ListContacts(ctx context.Context, accessToken, after string) (*HubSpotContactListResponse, error) {
	url := hubspotBaseURL + "/crm/v3/objects/contacts?limit=100&properties=email,firstname,lastname,company,lifecyclestage,hs_lead_status,associatedcompanyid,lastmodifieddate"
	if after != "" {
		url += "&after=" + after
	}
	return doGet[HubSpotContactListResponse](ctx, c, url, accessToken)
}

// ListDeals fetches deals from HubSpot with pagination.
func (c *HubSpotClient) ListDeals(ctx context.Context, accessToken, after string) (*HubSpotDealListResponse, error) {
	url := hubspotBaseURL + "/crm/v3/objects/deals?limit=100&properties=dealname,dealstage,amount,closedate,pipeline&associations=contacts"
	if after != "" {
		url += "&after=" + after
	}
	return doGet[HubSpotDealListResponse](ctx, c, url, accessToken)
}

// ListCompanies fetches companies from HubSpot with pagination.
func (c *HubSpotClient) ListCompanies(ctx context.Context, accessToken, after string) (*HubSpotCompanyListResponse, error) {
	url := hubspotBaseURL + "/crm/v3/objects/companies?limit=100&properties=name,domain,industry,numberofemployees,annualrevenue"
	if after != "" {
		url += "&after=" + after
	}
	return doGet[HubSpotCompanyListResponse](ctx, c, url, accessToken)
}

// SearchContacts searches contacts using the HubSpot search API (for incremental sync).
func (c *HubSpotClient) SearchContacts(ctx context.Context, accessToken string, filterGroups []HubSpotFilterGroup, after string) (*HubSpotContactListResponse, error) {
	searchReq := HubSpotSearchRequest{
		FilterGroups: filterGroups,
		Limit:        100,
		After:        after,
		Properties:   []string{"email", "firstname", "lastname", "company", "lifecyclestage", "hs_lead_status", "associatedcompanyid", "lastmodifieddate"},
	}

	return doPost[HubSpotContactListResponse](ctx, c, hubspotBaseURL+"/crm/v3/objects/contacts/search", accessToken, searchReq)
}

// SearchDeals searches deals using the HubSpot search API (for incremental sync).
func (c *HubSpotClient) SearchDeals(ctx context.Context, accessToken string, filterGroups []HubSpotFilterGroup, after string) (*HubSpotDealListResponse, error) {
	searchReq := HubSpotSearchRequest{
		FilterGroups: filterGroups,
		Limit:        100,
		After:        after,
		Properties:   []string{"dealname", "dealstage", "amount", "closedate", "pipeline"},
	}

	return doPost[HubSpotDealListResponse](ctx, c, hubspotBaseURL+"/crm/v3/objects/deals/search", accessToken, searchReq)
}

func doGet[T any](ctx context.Context, c *HubSpotClient, url, accessToken string) (*T, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

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
		return nil, fmt.Errorf("hubspot api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}

func doPost[T any](ctx context.Context, c *HubSpotClient, url, accessToken string, payload any) (*T, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hubspot api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result T
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}
