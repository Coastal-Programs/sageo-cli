package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/common/cost"
	"github.com/jakeschepis/sageo-cli/internal/llm"
	_ "github.com/jakeschepis/sageo-cli/internal/llm/providers" // register drivers
	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewRecommendationsCmd returns the `recommendations` command group.
func NewRecommendationsCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recommendations",
		Short: "Inspect concrete change recommendations produced by analyze",
	}
	cmd.AddCommand(newRecommendationsListCmd(format))
	cmd.AddCommand(newRecommendationsDraftCmd(format))
	cmd.AddCommand(newRecommendationsForecastCmd(format))
	cmd.AddCommand(newRecommendationsReviewCmd(format))
	return cmd
}

func newRecommendationsForecastCmd(format *string) *cobra.Command {
	c := &cobra.Command{
		Use:   "forecast",
		Short: "Estimate monthly click lift for each stored recommendation",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !state.Exists(".") {
				return output.PrintCodedError("NO_PROJECT",
					"No project initialized: run sageo init --url <site>",
					nil, nil, output.Format(*format))
			}
			st, err := state.Load(".")
			if err != nil {
				return output.PrintCodedError("STATE_LOAD_FAILED", "failed to load state", err, nil, output.Format(*format))
			}

			recs := recommendations.Load(st)
			if len(recs) == 0 {
				return output.PrintSuccess(map[string]any{
					"status":          "no_recommendations",
					"total_lift":      0,
					"recommendations": []any{},
				}, nil, output.Format(*format))
			}

			recommendations.AttachForecasts(st, recs)
			recommendations.UpsertRecommendations(st, recs)
			st.AddHistory("recommendations.forecast", fmt.Sprintf("forecasted %d recommendations", len(recs)))
			if err := st.Save("."); err != nil {
				return output.PrintCodedError("STATE_SAVE_FAILED", "failed to save state", err, nil, output.Format(*format))
			}

			type liftRow struct {
				ID           string   `json:"id"`
				TargetURL    string   `json:"target_url"`
				ChangeType   string   `json:"change_type"`
				PriorityTier string   `json:"priority_tier"`
				Point        int      `json:"point_estimate"`
				Low          int      `json:"range_low"`
				High         int      `json:"range_high"`
				RawPoint     int      `json:"raw_estimate"`
				Calibrated   bool     `json:"calibrated"`
				Confidence   string   `json:"confidence_label"`
				Caveats      []string `json:"caveats,omitempty"`
				Samples      int      `json:"calibration_samples"`
			}
			rows := make([]liftRow, 0, len(recs))
			var totalLow, totalHigh, totalPoint int
			tiers := map[string]int{}
			for _, r := range recs {
				if r.ForecastedLift == nil {
					continue
				}
				f := r.ForecastedLift
				totalPoint += f.Point()
				totalLow += f.Low()
				totalHigh += f.High()
				tier := string(f.PriorityTier)
				if tier == "" {
					tier = string(state.PriorityUnknown)
				}
				tiers[tier]++
				rows = append(rows, liftRow{
					ID:           r.ID,
					TargetURL:    r.TargetURL,
					ChangeType:   string(r.ChangeType),
					PriorityTier: tier,
					Point:        f.Point(),
					Low:          f.Low(),
					High:         f.High(),
					RawPoint:     f.RawEstimate,
					Calibrated:   f.IsCalibrated(),
					Confidence:   f.ConfidenceLabel,
					Caveats:      f.Caveats,
					Samples:      f.CalibrationSamples,
				})
			}
			sort.SliceStable(rows, func(i, j int) bool {
				if rows[i].PriorityTier != rows[j].PriorityTier {
					return tierRank(rows[i].PriorityTier) < tierRank(rows[j].PriorityTier)
				}
				return rows[i].Point > rows[j].Point
			})
			top := rows
			if len(top) > 10 {
				top = top[:10]
			}

			meta := map[string]any{
				"forecasted":           len(rows),
				"skipped":              len(recs) - len(rows),
				"tier_counts":          tiers,
				"estimated_range_low":  totalLow,
				"estimated_range_high": totalHigh,
				"fetched_at":           time.Now().UTC().Format(time.RFC3339),
				"source":               "forecast",
			}
			return output.PrintSuccess(map[string]any{
				"tier_counts":          tiers,
				"estimated_range_low":  totalLow,
				"estimated_range_high": totalHigh,
				"estimated_point":      totalPoint,
				"top":                  top,
			}, meta, output.Format(*format))
		},
	}
	return c
}

// estimatedDraftCostUSD is a conservative pre-flight estimate per draft.
// Real cost is computed from actual token usage after each call; this value
// is only used to gate the approval threshold before any network call.
const estimatedDraftCostUSD = 0.02

func newRecommendationsDraftCmd(format *string) *cobra.Command {
	var (
		urlFilter    string
		typeFilter   string
		limit        int
		providerName string
		dryRun       bool
	)

	c := &cobra.Command{
		Use:   "draft",
		Short: "Use an LLM to draft concrete copy for recommendations with empty RecommendedValue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !state.Exists(".") {
				return output.PrintCodedError("NO_PROJECT",
					"No project initialized: run sageo init --url <site>",
					nil, nil, output.Format(*format))
			}
			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}
			st, err := state.Load(".")
			if err != nil {
				return output.PrintCodedError("STATE_LOAD_FAILED", "failed to load state", err, nil, output.Format(*format))
			}

			all := recommendations.LoadTop(st, 0)
			var targets []recommendations.Recommendation
			for _, r := range all {
				if r.RecommendedValue != "" {
					continue
				}
				if urlFilter != "" && r.TargetURL != urlFilter {
					continue
				}
				if typeFilter != "" && string(r.ChangeType) != typeFilter {
					continue
				}
				targets = append(targets, r)
				if limit > 0 && len(targets) >= limit {
					break
				}
			}

			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: estimatedDraftCostUSD,
				Units:       len(targets),
				Basis:       fmt.Sprintf("llm draft: %d drafts @ ~$%.3f each (pre-flight estimate)", len(targets), estimatedDraftCostUSD),
			})
			if err != nil {
				return output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
			}
			approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

			resolvedProvider := providerName
			if resolvedProvider == "" {
				resolvedProvider = cfg.LLMProvider
			}
			if resolvedProvider == "" {
				resolvedProvider = "anthropic"
			}

			meta := map[string]any{
				"estimated_cost":    estimate.Amount,
				"currency":          estimate.Currency,
				"requires_approval": approval.RequiresApproval,
				"dry_run":           dryRun,
				"source":            resolvedProvider,
				"candidates":        len(targets),
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"status":     "dry_run",
					"candidates": summariseTargets(targets),
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			if len(targets) == 0 {
				return output.PrintSuccess(map[string]any{
					"status":  "no_work",
					"drafted": 0,
				}, meta, output.Format(*format))
			}

			provider, err := llm.New(resolvedProvider, cfg)
			if err != nil {
				return output.PrintCodedError(output.ErrProviderNotFound, "failed to initialise LLM provider", err, meta, output.Format(*format))
			}

			ctx := context.Background()
			accountingProvider := &countingProvider{inner: provider}
			var drafted, failed int
			var results []map[string]any
			for i := range targets {
				rec := &targets[i]
				page := recommendations.ContextForURL(st, rec.TargetURL, rec.TargetQuery)
				if err := recommendations.Draft(ctx, accountingProvider, rec, page); err != nil {
					failed++
					results = append(results, map[string]any{
						"id":          rec.ID,
						"target_url":  rec.TargetURL,
						"change_type": string(rec.ChangeType),
						"error":       err.Error(),
					})
					continue
				}
				if rec.RecommendedValue == "" {
					continue // unsupported change type; nothing to persist
				}
				drafted++
				results = append(results, map[string]any{
					"id":                rec.ID,
					"target_url":        rec.TargetURL,
					"change_type":       string(rec.ChangeType),
					"recommended_value": rec.RecommendedValue,
				})
			}

			if drafted > 0 {
				st.UpsertRecommendations(targets)
				st.AddHistory("recommendations.draft", fmt.Sprintf("drafted %d via %s", drafted, provider.Name()))
				if err := st.Save("."); err != nil {
					return output.PrintCodedError("STATE_SAVE_FAILED", "failed to save state", err, meta, output.Format(*format))
				}
			}

			meta["actual_cost_usd"] = accountingProvider.totalCostUSD
			meta["input_tokens"] = accountingProvider.totalInputTokens
			meta["output_tokens"] = accountingProvider.totalOutputTokens
			meta["fetched_at"] = time.Now().Format(time.RFC3339)

			return output.PrintSuccess(map[string]any{
				"drafted": drafted,
				"failed":  failed,
				"results": results,
			}, meta, output.Format(*format))
		},
	}

	c.Flags().StringVar(&urlFilter, "url", "", "Filter recommendations by target URL")
	c.Flags().StringVar(&typeFilter, "type", "", "Filter by change type (title, meta_description, h1, h2_add, schema_add, body_expand, internal_link_add)")
	c.Flags().IntVar(&limit, "limit", 20, "Maximum number of recommendations to draft (0 = all)")
	c.Flags().StringVar(&providerName, "provider", "", "LLM provider: anthropic|openai (defaults to config.llm_provider)")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without calling the LLM")

	return c
}

// countingProvider wraps an llm.Provider and totals usage across calls.
type countingProvider struct {
	inner             llm.Provider
	totalInputTokens  int
	totalOutputTokens int
	totalCostUSD      float64
}

func (c *countingProvider) Name() string { return c.inner.Name() }

func (c *countingProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	resp, err := c.inner.Complete(ctx, req)
	c.totalInputTokens += resp.InputTokens
	c.totalOutputTokens += resp.OutputTokens
	c.totalCostUSD += resp.CostUSD
	return resp, err
}

func summariseTargets(recs []recommendations.Recommendation) []map[string]string {
	out := make([]map[string]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, map[string]string{
			"id":          r.ID,
			"target_url":  r.TargetURL,
			"change_type": string(r.ChangeType),
		})
	}
	return out
}

// tierRank orders priority tiers for display: high < medium < low < unknown.
func tierRank(t string) int {
	switch t {
	case string(state.PriorityHigh):
		return 0
	case string(state.PriorityMedium):
		return 1
	case string(state.PriorityLow):
		return 2
	default:
		return 3
	}
}

func newRecommendationsListCmd(format *string) *cobra.Command {
	var (
		urlFilter  string
		typeFilter string
		top        int
		listFormat string
	)

	c := &cobra.Command{
		Use:   "list",
		Short: "List stored recommendations, sorted by priority",
		RunE: func(cmd *cobra.Command, args []string) error {
			// --format flag on this command overrides the global --output
			// for text rendering; when unset, fall through to the global.
			fmtStr := listFormat
			if fmtStr == "" {
				fmtStr = *format
			}

			if !state.Exists(".") {
				return output.PrintCodedError(
					"NO_PROJECT",
					"No project initialized: run sageo init --url <site>",
					nil, nil, output.Format(fmtStr),
				)
			}
			st, err := state.Load(".")
			if err != nil {
				return output.PrintCodedError("STATE_LOAD_FAILED", "failed to load state", err, nil, output.Format(fmtStr))
			}

			all := recommendations.LoadTop(st, 0) // sorted by priority desc

			// Apply filters.
			filtered := all[:0:0]
			for _, r := range all {
				if urlFilter != "" && r.TargetURL != urlFilter {
					continue
				}
				if typeFilter != "" && string(r.ChangeType) != typeFilter {
					continue
				}
				filtered = append(filtered, r)
			}

			if top > 0 && len(filtered) > top {
				filtered = filtered[:top]
			}

			if fmtStr == string(output.FormatText) {
				if len(filtered) == 0 {
					fmt.Println("No recommendations found.")
					return nil
				}
				for _, r := range filtered {
					tier := "UNKNOWN"
					var rangeLine string
					if r.ForecastedLift != nil {
						if r.ForecastedLift.PriorityTier != "" {
							tier = strings.ToUpper(string(r.ForecastedLift.PriorityTier))
						}
						if r.ForecastedLift.ConfidenceLabel == "insufficient_data" {
							rangeLine = fmt.Sprintf("~%d to %d clicks/mo (unverified)", r.ForecastedLift.Low(), r.ForecastedLift.High())
						} else {
							rangeLine = fmt.Sprintf("%d to %d clicks/mo", r.ForecastedLift.Low(), r.ForecastedLift.High())
						}
					}
					fmt.Printf("Priority: %s  %s  %s\n", tier, r.ChangeType, r.TargetURL)
					if rangeLine != "" {
						fmt.Printf("    est:   %s\n", rangeLine)
					}
					if r.TargetQuery != "" {
						fmt.Printf("    query: %s\n", r.TargetQuery)
					}
					if r.Rationale != "" {
						fmt.Printf("    why:   %s\n", r.Rationale)
					}
					if r.RecommendedValue != "" {
						fmt.Printf("    fix:   %s\n", r.RecommendedValue)
					}
					if r.ForecastedLift != nil && len(r.ForecastedLift.Caveats) > 0 {
						fmt.Printf("    note:  %s\n", strings.Join(r.ForecastedLift.Caveats, "; "))
					}
					fmt.Printf("    effort: %d min\n", r.EffortMinutes)
					fmt.Println(strings.Repeat("-", 60))
				}
				return nil
			}

			data := map[string]any{
				"count":           len(filtered),
				"recommendations": filtered,
			}
			return output.PrintSuccess(data, nil, output.Format(fmtStr))
		},
	}

	c.Flags().StringVar(&urlFilter, "url", "", "Filter by target URL (exact match)")
	c.Flags().StringVar(&typeFilter, "type", "", "Filter by change type (title, meta_description, h1, h2_add, schema_add, body_expand, speed_fix, backlink_outreach, indexability_fix, internal_link_add)")
	c.Flags().IntVar(&top, "top", 20, "Maximum number of recommendations to return (0 = all)")
	c.Flags().StringVar(&listFormat, "format", "", "Output format: json|text (defaults to global --output)")

	return c
}
