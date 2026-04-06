package gsc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	neturl "net/url"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/retry"
)

const baseURL = "https://www.googleapis.com/webmasters/v3"
const searchAnalyticsURL = "https://searchconsole.googleapis.com/webmasters/v3"

// Client communicates with the Google Search Console API.
type Client struct {
	httpClient  HTTPClient
	accessToken string
}

// HTTPClient is an interface for HTTP operations (supports testing).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient creates a GSC client with the given access token.
func NewClient(accessToken string) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		accessToken: accessToken,
	}
}

// NewClientWithHTTP creates a GSC client with a custom HTTP client (for testing).
func NewClientWithHTTP(accessToken string, httpClient HTTPClient) *Client {
	return &Client{
		httpClient:  httpClient,
		accessToken: accessToken,
	}
}

// Site represents a GSC property.
type Site struct {
	SiteURL         string `json:"siteUrl"`
	PermissionLevel string `json:"permissionLevel"`
}

// ListSites returns all GSC properties accessible by the authenticated user.
func (c *Client) ListSites() ([]Site, error) {
	var sites []Site

	err := retry.Do(2, func() error {
		req, err := http.NewRequest("GET", baseURL+"/sites", nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("listing sites: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			baseErr := fmt.Errorf("list sites returned status %d", resp.StatusCode)
			if resp.StatusCode == 429 || resp.StatusCode == 500 || resp.StatusCode == 502 || resp.StatusCode == 503 {
				return &retry.RetryableError{StatusCode: resp.StatusCode, Err: baseErr}
			}
			return baseErr
		}

		var result struct {
			SiteEntry []Site `json:"siteEntry"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decoding sites response: %w", err)
		}

		sites = result.SiteEntry
		return nil
	})

	if err != nil {
		return nil, err
	}
	return sites, nil
}

// QueryRequest defines the parameters for a Search Analytics query.
type QueryRequest struct {
	SiteURL    string   `json:"-"`
	StartDate  string   `json:"startDate"`
	EndDate    string   `json:"endDate"`
	Dimensions []string `json:"dimensions"`
	RowLimit   int      `json:"rowLimit,omitempty"`
	StartRow   int      `json:"startRow,omitempty"`
}

// QueryRow is a single row from a Search Analytics response.
type QueryRow struct {
	Keys        []string `json:"keys"`
	Clicks      float64  `json:"clicks"`
	Impressions float64  `json:"impressions"`
	CTR         float64  `json:"ctr"`
	Position    float64  `json:"position"`
}

// QueryResponse holds the result of a Search Analytics query.
type QueryResponse struct {
	Rows            []QueryRow `json:"rows"`
	ResponseAggType string     `json:"responseAggregationType,omitempty"`
}

// QueryPages retrieves page-level performance data.
func (c *Client) QueryPages(req QueryRequest) (*QueryResponse, error) {
	req.Dimensions = []string{"page"}
	return c.searchAnalytics(req)
}

// QueryKeywords retrieves keyword-level performance data.
func (c *Client) QueryKeywords(req QueryRequest) (*QueryResponse, error) {
	req.Dimensions = []string{"query"}
	return c.searchAnalytics(req)
}

// OpportunitySeed contains GSC data useful for opportunity detection.
type OpportunitySeed struct {
	Query       string  `json:"query"`
	Page        string  `json:"page"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
}

// QueryOpportunities retrieves page+query pairs with high impressions but low CTR or poor position.
func (c *Client) QueryOpportunities(siteURL, startDate, endDate string, rowLimit int) ([]OpportunitySeed, error) {
	req := QueryRequest{
		SiteURL:    siteURL,
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"query", "page"},
		RowLimit:   rowLimit,
	}

	resp, err := c.searchAnalytics(req)
	if err != nil {
		return nil, err
	}

	var seeds []OpportunitySeed
	for _, row := range resp.Rows {
		if len(row.Keys) < 2 {
			continue
		}
		// Filter: position > 3 (not top 3) or CTR < 3%
		if row.Position > 3 || row.CTR < 0.03 {
			seeds = append(seeds, OpportunitySeed{
				Query:       row.Keys[0],
				Page:        row.Keys[1],
				Clicks:      row.Clicks,
				Impressions: row.Impressions,
				CTR:         row.CTR,
				Position:    row.Position,
			})
		}
	}

	return seeds, nil
}

func (c *Client) searchAnalytics(req QueryRequest) (*QueryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding query: %w", err)
	}

	url := fmt.Sprintf("%s/sites/%s/searchAnalytics/query", searchAnalyticsURL, neturl.QueryEscape(req.SiteURL))

	var result QueryResponse

	err = retry.Do(2, func() error {
		httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("search analytics request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			baseErr := fmt.Errorf("search analytics returned status %d: %s", resp.StatusCode, string(respBody))
			if resp.StatusCode == 429 || resp.StatusCode == 500 || resp.StatusCode == 502 || resp.StatusCode == 503 {
				return &retry.RetryableError{StatusCode: resp.StatusCode, Err: baseErr}
			}
			return baseErr
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decoding search analytics response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Round values for clean output
	for i := range result.Rows {
		result.Rows[i].CTR = math.Round(result.Rows[i].CTR*10000) / 10000
		result.Rows[i].Position = math.Round(result.Rows[i].Position*10) / 10
	}

	return &result, nil
}
