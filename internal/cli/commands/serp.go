package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/cache"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/common/cost"
	"github.com/jakeschepis/sageo-cli/internal/serp"
	serpdforseo "github.com/jakeschepis/sageo-cli/internal/serp/dataforseo"
	"github.com/jakeschepis/sageo-cli/internal/serp/serpapi"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewSERPCmd returns the serp command group.
func NewSERPCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serp",
		Short: "SERP analysis commands",
		Long:  `Analyze search engine results pages using paid SERP providers. Supports --dry-run and cost estimation.`,
	}

	cmd.AddCommand(
		newSERPAnalyzeCmd(format, verbose),
		newSERPCompareCmd(format, verbose),
	)

	return cmd
}

func newSERPAnalyzeCmd(format *string, verbose *bool) *cobra.Command {
	var query, location, language string
	var numResults int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze SERP for a query",
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return output.PrintCodedError(output.ErrSERPFailed, "query is required",
					fmt.Errorf("use --query to specify a search query"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			provider, err := serpProvider(cfg)
			if err != nil {
				return output.PrintCodedError(output.ErrSERPFailed, "failed to initialize SERP provider", err, nil, output.Format(*format))
			}

			req := serp.AnalyzeRequest{
				Query:      query,
				Location:   location,
				Language:   language,
				NumResults: numResults,
			}

			// Compute cost estimate
			estimate, err := provider.Estimate(req)
			if err != nil {
				return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
			}

			approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

			meta := map[string]any{
				"estimated_cost":    estimate.Amount,
				"currency":          estimate.Currency,
				"basis":             estimate.Basis,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            provider.Name(),
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"query":    query,
					"provider": provider.Name(),
					"status":   "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			// Check cache
			cacheStore := cache.NewFileStore()
			cacheKey := map[string]any{"provider": provider.Name(), "request": req}
			if rec, hit, cacheErr := cacheStore.Get(provider.Name(), cacheKey); hit && cacheErr == nil {
				var resp serp.AnalyzeResponse
				if jsonErr := json.Unmarshal(rec.Payload, &resp); jsonErr == nil {
					meta["cached"] = true
					meta["fetched_at"] = rec.FetchedAt
					return output.PrintSuccess(resp, meta, output.Format(*format))
				}
			}

			// Execute
			resp, err := provider.Analyze(req)
			if err != nil {
				return output.PrintCodedError(output.ErrSERPFailed, "SERP analysis failed", err, nil, output.Format(*format))
			}

			// Cache result
			fetchedAt := time.Now().Format(time.RFC3339)
			if payload, jsonErr := json.Marshal(resp); jsonErr == nil {
				_ = cacheStore.Set(provider.Name(), cacheKey, cache.Record{
					Payload:    payload,
					Source:     provider.Name(),
					FetchedAt:  fetchedAt,
					TTLSeconds: 3600,
				})
			}

			meta["cached"] = false
			meta["fetched_at"] = fetchedAt

			return output.PrintSuccess(resp, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Search query to analyze (required)")
	cmd.Flags().StringVar(&location, "location", "", "Search location")
	cmd.Flags().StringVar(&language, "language", "", "Search language code")
	cmd.Flags().IntVar(&numResults, "num", 10, "Number of results to fetch")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

func newSERPCompareCmd(format *string, verbose *bool) *cobra.Command {
	var queries []string
	var location, language string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare SERP results for multiple queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(queries) < 2 {
				return output.PrintCodedError(output.ErrSERPFailed, "at least 2 queries required",
					fmt.Errorf("use --query multiple times to specify queries"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			provider, err := serpProvider(cfg)
			if err != nil {
				return output.PrintCodedError(output.ErrSERPFailed, "failed to initialize SERP provider", err, nil, output.Format(*format))
			}

			// Estimate total cost
			totalEstimate := cost.Estimate{Currency: cost.CurrencyUSD}
			for _, q := range queries {
				est, err := provider.Estimate(serp.AnalyzeRequest{Query: q, Location: location, Language: language})
				if err != nil {
					return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
				}
				totalEstimate.Amount += est.Amount
				totalEstimate.Basis = est.Basis
			}

			approval := cost.EvaluateApproval(totalEstimate, cfg.ApprovalThresholdUSD)

			meta := map[string]any{
				"estimated_cost":    totalEstimate.Amount,
				"currency":          totalEstimate.Currency,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            provider.Name(),
				"query_count":       len(queries),
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"queries":  queries,
					"provider": provider.Name(),
					"status":   "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			results := make(map[string]*serp.AnalyzeResponse, len(queries))
			for _, q := range queries {
				resp, err := provider.Analyze(serp.AnalyzeRequest{
					Query:    q,
					Location: location,
					Language: language,
				})
				if err != nil {
					return output.PrintCodedError(output.ErrSERPFailed, fmt.Sprintf("SERP analysis failed for query %q", q), err, nil, output.Format(*format))
				}
				results[q] = resp
			}

			meta["cached"] = false
			meta["fetched_at"] = time.Now().Format(time.RFC3339)

			return output.PrintSuccess(results, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringArrayVar(&queries, "query", nil, "Search queries to compare (use multiple times)")
	cmd.Flags().StringVar(&location, "location", "", "Search location")
	cmd.Flags().StringVar(&language, "language", "", "Search language code")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

func serpProvider(cfg *config.Config) (serp.Provider, error) {
	switch cfg.SERPProvider {
	case "dataforseo":
		if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
			return nil, fmt.Errorf("dataforseo_login and dataforseo_password not configured; run 'sageo login' to set them")
		}
		return serpdforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword), nil
	case "serpapi", "":
		// Fall back to DataForSEO when login is set and provider is unset/default
		if (cfg.SERPProvider == "" || cfg.SERPProvider == "serpapi") && cfg.DataForSEOLogin != "" && cfg.DataForSEOPassword != "" {
			return serpdforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword), nil
		}
		if cfg.SERPAPIKey == "" {
			return nil, fmt.Errorf("serp_api_key not configured; set via 'sageo config set serp_api_key <key>' or run 'sageo login' to configure DataForSEO")
		}
		return serpapi.New(cfg.SERPAPIKey), nil
	default:
		return nil, fmt.Errorf("unsupported SERP provider: %s", cfg.SERPProvider)
	}
}
