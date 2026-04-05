package dataforseo

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/jakeschepis/sageo-cli/internal/common/cost"
	dfs "github.com/jakeschepis/sageo-cli/internal/dataforseo"
	"github.com/jakeschepis/sageo-cli/internal/serp"
)

const (
	serpEndpoint    = "/v3/serp/google/organic/live/regular"
	costPerQueryUSD = 0.002 // live mode
	costBasis       = "dataforseo: $0.002/query (live mode)"
)

// Adapter implements serp.Provider using the DataForSEO SERP API.
type Adapter struct {
	client *dfs.Client
}

// New creates a DataForSEO SERP adapter.
func New(login, password string, opts ...dfs.Option) *Adapter {
	return &Adapter{client: dfs.New(login, password, opts...)}
}

// Name returns the provider identifier.
func (a *Adapter) Name() string { return "dataforseo" }

// Estimate returns a cost estimate without executing the search.
func (a *Adapter) Estimate(req serp.AnalyzeRequest) (cost.Estimate, error) {
	return cost.BuildEstimate(cost.EstimateInput{
		UnitCostUSD: costPerQueryUSD,
		Units:       1,
		Basis:       costBasis,
	})
}

// Analyze executes a SERP query against the DataForSEO live endpoint.
func (a *Adapter) Analyze(req serp.AnalyzeRequest) (*serp.AnalyzeResponse, error) {
	task := map[string]any{
		"keyword": req.Query,
	}
	if req.Location != "" {
		task["location_name"] = req.Location
	}
	if req.Language != "" {
		task["language_code"] = req.Language
	}
	if req.NumResults > 0 {
		task["depth"] = req.NumResults
	}

	raw, err := a.client.Post(serpEndpoint, []map[string]any{task})
	if err != nil {
		return nil, fmt.Errorf("dataforseo serp request: %w", err)
	}

	var envelope struct {
		StatusCode    int    `json:"status_code"`
		StatusMessage string `json:"status_message"`
		Tasks         []struct {
			StatusCode    int    `json:"status_code"`
			StatusMessage string `json:"status_message"`
			Result        []struct {
				Keyword    string `json:"keyword"`
				TotalCount int64  `json:"se_results_count"`
				Items      []struct {
					Type        string `json:"type"`
					RankGroup   int    `json:"rank_group"`
					Title       string `json:"title"`
					URL         string `json:"url"`
					Description string `json:"description"`
				} `json:"items"`
			} `json:"result"`
		} `json:"tasks"`
	}

	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decoding dataforseo response: %w", err)
	}

	if envelope.StatusCode != 20000 {
		return nil, fmt.Errorf("dataforseo error %d: %s", envelope.StatusCode, envelope.StatusMessage)
	}
	if len(envelope.Tasks) == 0 {
		return nil, fmt.Errorf("dataforseo returned no tasks")
	}
	task0 := envelope.Tasks[0]
	if task0.StatusCode != 20000 {
		return nil, fmt.Errorf("dataforseo task error %d: %s", task0.StatusCode, task0.StatusMessage)
	}
	if len(task0.Result) == 0 {
		return &serp.AnalyzeResponse{Query: req.Query}, nil
	}

	result := task0.Result[0]
	organic := make([]serp.OrganicResult, 0)
	for _, item := range result.Items {
		if item.Type != "organic" {
			continue
		}
		parsed, _ := url.Parse(item.URL)
		domain := ""
		if parsed != nil {
			domain = parsed.Hostname()
		}
		organic = append(organic, serp.OrganicResult{
			Position: item.RankGroup,
			Title:    item.Title,
			Link:     item.URL,
			Snippet:  item.Description,
			Domain:   domain,
		})
	}

	return &serp.AnalyzeResponse{
		Query:          req.Query,
		OrganicResults: organic,
		TotalResults:   result.TotalCount,
	}, nil
}
