package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/common/cost"
	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewLabsCmd returns the labs command group.
func NewLabsCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "labs",
		Short: "DataForSEO Labs commands",
		Long:  `Query DataForSEO Labs datasets for domain and keyword intelligence.`,
	}

	cmd.AddCommand(
		newLabsRankedKeywordsCmd(format, verbose),
		newLabsKeywordsCmd(format, verbose),
		newLabsOverviewCmd(format, verbose),
		newLabsCompetitorsCmd(format, verbose),
		newLabsKeywordIdeasCmd(format, verbose),
	)

	return cmd
}

func newLabsRankedKeywordsCmd(format *string, verbose *bool) *cobra.Command {
	var target, location, language string
	var limit, minVolume int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "ranked-keywords",
		Short: "Get keywords a domain or URL ranks for",
		Long:  `Retrieve ranked keywords for a domain or URL from DataForSEO Labs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if target == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "target is required",
					fmt.Errorf("use --target to specify a domain or URL"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "DataForSEO credentials not configured",
					fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
			}

			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: 0.01,
				Units:       1,
				Basis:       "dataforseo labs ranked keywords: 1 task @ $0.01/task",
			})
			if err != nil {
				return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
			}

			approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

			meta := map[string]any{
				"estimated_cost":    estimate.Amount,
				"currency":          estimate.Currency,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            "dataforseo",
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"target": target,
					"status": "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			task := map[string]any{
				"target":        target,
				"location_name": location,
				"language_code": language,
				"limit":         limit,
			}
			if minVolume > 0 {
				task["filters"] = []any{
					[]any{"keyword_data.keyword_info.search_volume", ">", minVolume},
				}
			}

			raw, err := client.Post("/v3/dataforseo_labs/google/ranked_keywords/live", []map[string]any{task})
			if err != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "ranked keywords request failed", err, nil, output.Format(*format))
			}

			var envelope struct {
				StatusCode    int    `json:"status_code"`
				StatusMessage string `json:"status_message"`
				Tasks         []struct {
					StatusCode    int               `json:"status_code"`
					StatusMessage string            `json:"status_message"`
					Result        []json.RawMessage `json:"result"`
				} `json:"tasks"`
			}

			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "failed to decode DataForSEO response", jsonErr, nil, output.Format(*format))
			}

			if envelope.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrLabsFailed,
					fmt.Sprintf("DataForSEO error %d: %s", envelope.StatusCode, envelope.StatusMessage),
					fmt.Errorf("code %d", envelope.StatusCode), nil, output.Format(*format))
			}

			var results []json.RawMessage
			if len(envelope.Tasks) > 0 {
				results = envelope.Tasks[0].Result
			}

			meta["fetched_at"] = time.Now().Format(time.RFC3339)
			meta["target"] = target

			return output.PrintSuccess(results, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Domain or URL to analyze (required)")
	cmd.Flags().StringVar(&location, "location", "United States", "Location name (e.g. 'United States')")
	cmd.Flags().StringVar(&language, "language", "en", "Language name (e.g. 'en')")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of results")
	cmd.Flags().IntVar(&minVolume, "min-volume", 0, "Minimum monthly search volume filter")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

func newLabsKeywordsCmd(format *string, verbose *bool) *cobra.Command {
	var target, location, language string
	var limit int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "keywords",
		Short: "Get keyword ideas relevant to a domain",
		Long:  `Retrieve keywords for a site from DataForSEO Labs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if target == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "target is required",
					fmt.Errorf("use --target to specify a domain"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "DataForSEO credentials not configured",
					fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
			}

			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: 0.01,
				Units:       1,
				Basis:       "dataforseo labs keywords for site: 1 task @ $0.01/task",
			})
			if err != nil {
				return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
			}

			approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

			meta := map[string]any{
				"estimated_cost":    estimate.Amount,
				"currency":          estimate.Currency,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            "dataforseo",
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"target": target,
					"status": "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			locationCode := 2840
			if location != "United States" {
				return output.PrintCodedError(output.ErrLabsFailed, "unsupported location for this endpoint",
					fmt.Errorf("--location currently supports only 'United States' for location_code mapping"), nil, output.Format(*format))
			}

			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			task := map[string]any{
				"target":        target,
				"location_code": locationCode,
				"language_code": language,
				"limit":         limit,
			}

			raw, err := client.Post("/v3/dataforseo_labs/google/keywords_for_site/live", []map[string]any{task})
			if err != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "keywords for site request failed", err, nil, output.Format(*format))
			}

			var envelope struct {
				StatusCode    int    `json:"status_code"`
				StatusMessage string `json:"status_message"`
				Tasks         []struct {
					StatusCode    int               `json:"status_code"`
					StatusMessage string            `json:"status_message"`
					Result        []json.RawMessage `json:"result"`
				} `json:"tasks"`
			}

			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "failed to decode DataForSEO response", jsonErr, nil, output.Format(*format))
			}

			if envelope.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrLabsFailed,
					fmt.Sprintf("DataForSEO error %d: %s", envelope.StatusCode, envelope.StatusMessage),
					fmt.Errorf("code %d", envelope.StatusCode), nil, output.Format(*format))
			}

			var results []json.RawMessage
			if len(envelope.Tasks) > 0 {
				results = envelope.Tasks[0].Result
			}

			meta["fetched_at"] = time.Now().Format(time.RFC3339)
			meta["target"] = target

			return output.PrintSuccess(results, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Domain to analyze (required)")
	cmd.Flags().StringVar(&location, "location", "United States", "Location name (default supports 'United States')")
	cmd.Flags().StringVar(&language, "language", "en", "Language code (e.g. 'en')")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of results")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

func newLabsOverviewCmd(format *string, verbose *bool) *cobra.Command {
	var target, location, language string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Get domain ranking overview",
		Long:  `Retrieve domain rank overview from DataForSEO Labs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if target == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "target is required",
					fmt.Errorf("use --target to specify a domain"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "DataForSEO credentials not configured",
					fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
			}

			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: 0.01,
				Units:       1,
				Basis:       "dataforseo labs domain rank overview: 1 task @ $0.01/task",
			})
			if err != nil {
				return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
			}

			approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

			meta := map[string]any{
				"estimated_cost":    estimate.Amount,
				"currency":          estimate.Currency,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            "dataforseo",
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"target": target,
					"status": "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			task := map[string]any{
				"target":        target,
				"location_name": location,
				"language_code": language,
			}

			raw, err := client.Post("/v3/dataforseo_labs/google/domain_rank_overview/live", []map[string]any{task})
			if err != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "domain rank overview request failed", err, nil, output.Format(*format))
			}

			var envelope struct {
				StatusCode    int    `json:"status_code"`
				StatusMessage string `json:"status_message"`
				Tasks         []struct {
					StatusCode    int               `json:"status_code"`
					StatusMessage string            `json:"status_message"`
					Result        []json.RawMessage `json:"result"`
				} `json:"tasks"`
			}

			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "failed to decode DataForSEO response", jsonErr, nil, output.Format(*format))
			}

			if envelope.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrLabsFailed,
					fmt.Sprintf("DataForSEO error %d: %s", envelope.StatusCode, envelope.StatusMessage),
					fmt.Errorf("code %d", envelope.StatusCode), nil, output.Format(*format))
			}

			var results []json.RawMessage
			if len(envelope.Tasks) > 0 {
				results = envelope.Tasks[0].Result
			}

			meta["fetched_at"] = time.Now().Format(time.RFC3339)
			meta["target"] = target

			return output.PrintSuccess(results, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Domain to analyze (required)")
	cmd.Flags().StringVar(&location, "location", "United States", "Location name (e.g. 'United States')")
	cmd.Flags().StringVar(&language, "language", "en", "Language name (e.g. 'en')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

func newLabsCompetitorsCmd(format *string, verbose *bool) *cobra.Command {
	var target, location, language string
	var limit int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "competitors",
		Short: "Get competing domains",
		Long:  `Retrieve competitor domains from DataForSEO Labs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if target == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "target is required",
					fmt.Errorf("use --target to specify a domain"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "DataForSEO credentials not configured",
					fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
			}

			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: 0.01,
				Units:       1,
				Basis:       "dataforseo labs competitors domain: 1 task @ $0.01/task",
			})
			if err != nil {
				return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
			}

			approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

			meta := map[string]any{
				"estimated_cost":    estimate.Amount,
				"currency":          estimate.Currency,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            "dataforseo",
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"target": target,
					"status": "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			task := map[string]any{
				"target":        target,
				"location_name": location,
				"language_code": language,
				"limit":         limit,
			}

			raw, err := client.Post("/v3/dataforseo_labs/google/competitors_domain/live", []map[string]any{task})
			if err != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "competitors domain request failed", err, nil, output.Format(*format))
			}

			var envelope struct {
				StatusCode    int    `json:"status_code"`
				StatusMessage string `json:"status_message"`
				Tasks         []struct {
					StatusCode    int               `json:"status_code"`
					StatusMessage string            `json:"status_message"`
					Result        []json.RawMessage `json:"result"`
				} `json:"tasks"`
			}

			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "failed to decode DataForSEO response", jsonErr, nil, output.Format(*format))
			}

			if envelope.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrLabsFailed,
					fmt.Sprintf("DataForSEO error %d: %s", envelope.StatusCode, envelope.StatusMessage),
					fmt.Errorf("code %d", envelope.StatusCode), nil, output.Format(*format))
			}

			var results []json.RawMessage
			if len(envelope.Tasks) > 0 {
				results = envelope.Tasks[0].Result
			}

			meta["fetched_at"] = time.Now().Format(time.RFC3339)
			meta["target"] = target

			return output.PrintSuccess(results, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Domain to analyze (required)")
	cmd.Flags().StringVar(&location, "location", "United States", "Location name (e.g. 'United States')")
	cmd.Flags().StringVar(&language, "language", "en", "Language name (e.g. 'en')")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of results")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

func newLabsKeywordIdeasCmd(format *string, verbose *bool) *cobra.Command {
	var keyword, location, language string
	var limit int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "keyword-ideas",
		Short: "Get keyword ideas from a seed keyword",
		Long:  `Retrieve keyword ideas from DataForSEO Labs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if keyword == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "keyword is required",
					fmt.Errorf("use --keyword to specify a seed keyword"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				return output.PrintCodedError(output.ErrLabsFailed, "DataForSEO credentials not configured",
					fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
			}

			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: 0.01,
				Units:       1,
				Basis:       "dataforseo labs keyword ideas: 1 task @ $0.01/task",
			})
			if err != nil {
				return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
			}

			approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

			meta := map[string]any{
				"estimated_cost":    estimate.Amount,
				"currency":          estimate.Currency,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            "dataforseo",
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"keyword": keyword,
					"status":  "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			task := map[string]any{
				"keywords":      []string{keyword},
				"location_name": location,
				"language_code": language,
				"limit":         limit,
			}

			raw, err := client.Post("/v3/dataforseo_labs/google/keyword_ideas/live", []map[string]any{task})
			if err != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "keyword ideas request failed", err, nil, output.Format(*format))
			}

			var envelope struct {
				StatusCode    int    `json:"status_code"`
				StatusMessage string `json:"status_message"`
				Tasks         []struct {
					StatusCode    int               `json:"status_code"`
					StatusMessage string            `json:"status_message"`
					Result        []json.RawMessage `json:"result"`
				} `json:"tasks"`
			}

			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				return output.PrintCodedError(output.ErrLabsFailed, "failed to decode DataForSEO response", jsonErr, nil, output.Format(*format))
			}

			if envelope.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrLabsFailed,
					fmt.Sprintf("DataForSEO error %d: %s", envelope.StatusCode, envelope.StatusMessage),
					fmt.Errorf("code %d", envelope.StatusCode), nil, output.Format(*format))
			}

			var results []json.RawMessage
			if len(envelope.Tasks) > 0 {
				results = envelope.Tasks[0].Result
			}

			meta["fetched_at"] = time.Now().Format(time.RFC3339)
			meta["keyword"] = keyword

			return output.PrintSuccess(results, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&keyword, "keyword", "", "Seed keyword to analyze (required)")
	cmd.Flags().StringVar(&location, "location", "United States", "Location name (e.g. 'United States')")
	cmd.Flags().StringVar(&language, "language", "en", "Language name (e.g. 'en')")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of results")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}
