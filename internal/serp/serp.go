package serp

import "github.com/jakeschepis/sageo-cli/internal/common/cost"

// AnalyzeRequest defines the input for a SERP analysis.
type AnalyzeRequest struct {
	Query      string `json:"query"`
	Location   string `json:"location,omitempty"`
	Language   string `json:"language,omitempty"`
	NumResults int    `json:"num_results,omitempty"`
}

// OrganicResult represents a single organic search result.
type OrganicResult struct {
	Position int    `json:"position"`
	Title    string `json:"title"`
	Link     string `json:"link"`
	Snippet  string `json:"snippet"`
	Domain   string `json:"domain,omitempty"`
}

// AnalyzeResponse holds the result of a SERP analysis.
type AnalyzeResponse struct {
	Query          string          `json:"query"`
	OrganicResults []OrganicResult `json:"organic_results"`
	TotalResults   int64           `json:"total_results,omitempty"`
	SearchTime     float64         `json:"search_time,omitempty"`
}

// Provider defines the interface for SERP data providers.
type Provider interface {
	// Name returns the provider identifier.
	Name() string
	// Estimate returns a cost estimate for the given request without executing.
	Estimate(req AnalyzeRequest) (cost.Estimate, error)
	// Analyze executes a SERP query and returns results.
	Analyze(req AnalyzeRequest) (*AnalyzeResponse, error)
}
