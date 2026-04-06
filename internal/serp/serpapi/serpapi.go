package serpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/cost"
	"github.com/jakeschepis/sageo-cli/internal/serp"
)

const (
	defaultBaseURL    = "https://serpapi.com/search"
	defaultAccountURL = "https://serpapi.com/account.json"
	costPerSearchUSD  = 0.01 // estimated cost per search
	costBasis         = "serpapi: $0.01/search (estimate based on 100 searches/month plan)"
)

// HTTPClient is an interface for HTTP operations (supports testing).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter implements serp.Provider for SerpAPI.
type Adapter struct {
	apiKey     string
	baseURL    string
	accountURL string
	httpClient HTTPClient
}

// Option configures the SerpAPI adapter.
type Option func(*Adapter)

// WithBaseURL overrides the default SerpAPI endpoint (useful for testing).
func WithBaseURL(url string) Option {
	return func(a *Adapter) { a.baseURL = url }
}

// WithAccountURL overrides the default SerpAPI account endpoint (useful for testing).
func WithAccountURL(url string) Option {
	return func(a *Adapter) { a.accountURL = url }
}

// WithHTTPClient overrides the default HTTP client (useful for testing).
func WithHTTPClient(c HTTPClient) Option {
	return func(a *Adapter) { a.httpClient = c }
}

// New creates a SerpAPI adapter.
func New(apiKey string, opts ...Option) *Adapter {
	a := &Adapter{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		accountURL: defaultAccountURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns the provider identifier.
func (a *Adapter) Name() string { return "serpapi" }

// Estimate returns a cost estimate without executing the search.
func (a *Adapter) Estimate(req serp.AnalyzeRequest) (cost.Estimate, error) {
	units := 1
	if req.NumResults > 100 {
		units = (req.NumResults + 99) / 100
	}
	return cost.BuildEstimate(cost.EstimateInput{
		UnitCostUSD: costPerSearchUSD,
		Units:       units,
		Basis:       costBasis,
	})
}

// Analyze executes a SERP query against SerpAPI.
func (a *Adapter) Analyze(req serp.AnalyzeRequest) (*serp.AnalyzeResponse, error) {
	params := url.Values{
		"api_key": {a.apiKey},
		"q":       {req.Query},
		"engine":  {"google"},
	}
	if req.Location != "" {
		params.Set("location", req.Location)
	}
	if req.Language != "" {
		params.Set("hl", req.Language)
	}
	if req.NumResults > 0 {
		params.Set("num", strconv.Itoa(req.NumResults))
	}

	httpReq, err := http.NewRequest("GET", a.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating serpapi request: %w", err)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("serpapi request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serpapi returned status %d", resp.StatusCode)
	}

	var raw struct {
		OrganicResults []struct {
			Position int    `json:"position"`
			Title    string `json:"title"`
			Link     string `json:"link"`
			Snippet  string `json:"snippet"`
		} `json:"organic_results"`
		SearchInformation struct {
			TotalResults     string  `json:"total_results"`
			TimeTakenDisplay float64 `json:"time_taken_displayed"`
		} `json:"search_information"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding serpapi response: %w", err)
	}

	results := make([]serp.OrganicResult, 0, len(raw.OrganicResults))
	for _, r := range raw.OrganicResults {
		parsed, _ := url.Parse(r.Link)
		domain := ""
		if parsed != nil {
			domain = parsed.Hostname()
		}
		results = append(results, serp.OrganicResult{
			Position: r.Position,
			Title:    r.Title,
			Link:     r.Link,
			Snippet:  r.Snippet,
			Domain:   domain,
		})
	}

	totalResults, _ := strconv.ParseInt(raw.SearchInformation.TotalResults, 10, 64)

	return &serp.AnalyzeResponse{
		Query:          req.Query,
		OrganicResults: results,
		TotalResults:   totalResults,
		SearchTime:     raw.SearchInformation.TimeTakenDisplay,
	}, nil
}

// VerifyKey checks whether the configured API key is valid by calling the
// SerpAPI account endpoint and inspecting the response.
func (a *Adapter) VerifyKey() error {
	params := url.Values{"api_key": {a.apiKey}}
	httpReq, err := http.NewRequest("GET", a.accountURL+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("creating serpapi account request: %w", err)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("serpapi account request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("serpapi: invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("serpapi account endpoint returned status %d", resp.StatusCode)
	}

	var acct struct {
		AccountEmail string `json:"account_email"`
		PlanName     string `json:"plan_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&acct); err != nil {
		return fmt.Errorf("serpapi: malformed account response: %w", err)
	}

	if acct.AccountEmail == "" {
		return fmt.Errorf("serpapi: invalid API key (no account email in response)")
	}

	return nil
}
