// Package llmmentions wraps the DataForSEO LLM Mentions API endpoints.
//
// This is "Layer B" of brand mention detection: DataForSEO aggregates
// real-world LLM responses across ChatGPT / Claude / Gemini / Perplexity and
// exposes volume, impressions, top pages, and top domains for a target
// keyword or domain.
//
// See: https://docs.dataforseo.com/v3/ai_optimization/llm_mentions/overview/
//
// All endpoints use the Live method (only method supported by LLM Mentions)
// and default to Australia (location_code 2036) / English per project
// convention. Pricing is pay-per-row and typically runs ~$0.10 per task;
// callers should surface cost via internal/common/cost before invoking.
package llmmentions

import (
	"encoding/json"
	"fmt"

	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
)

// Endpoint paths.
const (
	pathSearch            = "/v3/ai_optimization/llm_mentions/search/live"
	pathTopPages          = "/v3/ai_optimization/llm_mentions/top_pages/live"
	pathTopDomains        = "/v3/ai_optimization/llm_mentions/top_domains/live"
	pathAggregatedMetrics = "/v3/ai_optimization/llm_mentions/aggregated_metrics/live"
)

// Defaults follow the project-wide Australia / English convention (see
// internal/cli/commands/labs.go).
const (
	DefaultLocationCode = 2036 // Australia
	DefaultLanguageCode = "en"
	DefaultPlatform     = "google"
)

// Client wraps a DataForSEO HTTP client and exposes typed methods for each
// LLM Mentions endpoint.
type Client struct {
	dfs *dataforseo.Client
}

// NewClient builds a Client from an existing DataForSEO client.
func NewClient(dfs *dataforseo.Client) *Client {
	return &Client{dfs: dfs}
}

// Request is the shared request shape for all four endpoints. Fields left at
// their zero value fall back to sensible defaults (Australia, English,
// Google).
type Request struct {
	Keyword      string `json:"-"`
	LocationCode int    `json:"location_code"`
	LanguageCode string `json:"language_code"`
	Platform     string `json:"platform,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

// buildBody renders a Request into the DataForSEO POST body. The API wraps
// keywords/domains in a `target` array; we always target a single keyword
// term here.
func buildBody(req Request) []map[string]any {
	loc := req.LocationCode
	if loc == 0 {
		loc = DefaultLocationCode
	}
	lang := req.LanguageCode
	if lang == "" {
		lang = DefaultLanguageCode
	}
	platform := req.Platform
	if platform == "" {
		platform = DefaultPlatform
	}
	body := map[string]any{
		"location_code": loc,
		"language_code": lang,
		"platform":      platform,
		"target": []map[string]any{
			{"keyword": req.Keyword},
		},
	}
	if req.Limit > 0 {
		body["limit"] = req.Limit
	}
	return []map[string]any{body}
}

// envelope is the generic DataForSEO live-endpoint response shape.
type envelope struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Tasks         []struct {
		StatusCode    int               `json:"status_code"`
		StatusMessage string            `json:"status_message"`
		Result        []json.RawMessage `json:"result"`
	} `json:"tasks"`
}

// decode pulls out the first task result and surfaces envelope / task-level
// errors with the same shape as other packages (see internal/cli/commands/aeo.go).
func decode(raw []byte, endpoint string) (json.RawMessage, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", endpoint, err)
	}
	if env.StatusCode != 20000 {
		return nil, fmt.Errorf("dataforseo %s error %d: %s", endpoint, env.StatusCode, env.StatusMessage)
	}
	if len(env.Tasks) == 0 {
		return nil, fmt.Errorf("dataforseo %s: no tasks returned", endpoint)
	}
	task := env.Tasks[0]
	if task.StatusCode != 20000 {
		return nil, fmt.Errorf("dataforseo %s task error %d: %s", endpoint, task.StatusCode, task.StatusMessage)
	}
	if len(task.Result) == 0 {
		return nil, nil
	}
	return task.Result[0], nil
}

// ---------- Search ----------

// SearchItem is a single mention row returned by the Search endpoint.
type SearchItem struct {
	Type           string   `json:"type,omitempty"`
	Question       string   `json:"question,omitempty"`
	Answer         string   `json:"answer,omitempty"`
	Platform       string   `json:"platform,omitempty"`
	MentionsCount  int      `json:"mentions_count"`
	AISearchVolume int      `json:"ai_search_volume"`
	Impressions    int64    `json:"impressions"`
	Sources        []string `json:"sources,omitempty"`
}

// SearchResult is the typed `result[0]` shape returned by the Search endpoint.
type SearchResult struct {
	Keyword    string       `json:"keyword,omitempty"`
	TotalCount int          `json:"total_count"`
	Items      []SearchItem `json:"items,omitempty"`
}

// Search queries the /search/live endpoint for raw mentions of a term.
func (c *Client) Search(req Request) (*SearchResult, error) {
	raw, err := c.dfs.Post(pathSearch, buildBody(req))
	if err != nil {
		return nil, fmt.Errorf("llm_mentions search request: %w", err)
	}
	result, err := decode(raw, "llm_mentions/search")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &SearchResult{}, nil
	}
	var out SearchResult
	if err := json.Unmarshal(result, &out); err != nil {
		return nil, fmt.Errorf("decode search result: %w", err)
	}
	return &out, nil
}

// ---------- TopPages ----------

// TopPageItem is a single cited-page row.
type TopPageItem struct {
	URL           string `json:"url"`
	Domain        string `json:"domain,omitempty"`
	MentionsCount int    `json:"mentions_count"`
	Impressions   int64  `json:"impressions"`
}

// TopPagesResult is the typed result payload for the top_pages endpoint.
type TopPagesResult struct {
	Keyword    string        `json:"keyword,omitempty"`
	TotalCount int           `json:"total_count"`
	Items      []TopPageItem `json:"items,omitempty"`
}

// TopPages queries the /top_pages/live endpoint.
func (c *Client) TopPages(req Request) (*TopPagesResult, error) {
	raw, err := c.dfs.Post(pathTopPages, buildBody(req))
	if err != nil {
		return nil, fmt.Errorf("llm_mentions top_pages request: %w", err)
	}
	result, err := decode(raw, "llm_mentions/top_pages")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &TopPagesResult{}, nil
	}
	var out TopPagesResult
	if err := json.Unmarshal(result, &out); err != nil {
		return nil, fmt.Errorf("decode top_pages result: %w", err)
	}
	return &out, nil
}

// ---------- TopDomains ----------

// TopDomainItem is a single cited-domain row.
type TopDomainItem struct {
	Domain        string `json:"domain"`
	MentionsCount int    `json:"mentions_count"`
	Impressions   int64  `json:"impressions"`
}

// TopDomainsResult is the typed result payload for the top_domains endpoint.
type TopDomainsResult struct {
	Keyword    string          `json:"keyword,omitempty"`
	TotalCount int             `json:"total_count"`
	Items      []TopDomainItem `json:"items,omitempty"`
}

// TopDomains queries the /top_domains/live endpoint.
func (c *Client) TopDomains(req Request) (*TopDomainsResult, error) {
	raw, err := c.dfs.Post(pathTopDomains, buildBody(req))
	if err != nil {
		return nil, fmt.Errorf("llm_mentions top_domains request: %w", err)
	}
	result, err := decode(raw, "llm_mentions/top_domains")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &TopDomainsResult{}, nil
	}
	var out TopDomainsResult
	if err := json.Unmarshal(result, &out); err != nil {
		return nil, fmt.Errorf("decode top_domains result: %w", err)
	}
	return &out, nil
}

// ---------- AggregatedMetrics ----------

// AggregatedMetricsResult is the typed payload for the aggregated_metrics
// endpoint. The DataForSEO response nests results under `total` with group
// arrays keyed by location / language / platform. We keep the raw payload
// accessible while exposing the most useful top-level totals.
type AggregatedMetricsResult struct {
	Keyword    string          `json:"keyword,omitempty"`
	TotalCount int             `json:"total_count"`
	Total      json.RawMessage `json:"total,omitempty"`
}

// AggregatedMetrics queries the /aggregated_metrics/live endpoint.
func (c *Client) AggregatedMetrics(req Request) (*AggregatedMetricsResult, error) {
	raw, err := c.dfs.Post(pathAggregatedMetrics, buildBody(req))
	if err != nil {
		return nil, fmt.Errorf("llm_mentions aggregated_metrics request: %w", err)
	}
	result, err := decode(raw, "llm_mentions/aggregated_metrics")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &AggregatedMetricsResult{}, nil
	}
	var out AggregatedMetricsResult
	if err := json.Unmarshal(result, &out); err != nil {
		return nil, fmt.Errorf("decode aggregated_metrics result: %w", err)
	}
	return &out, nil
}
