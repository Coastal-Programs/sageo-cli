package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	xterm "golang.org/x/term"

	"github.com/jakeschepis/sageo-cli/internal/audit"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/crawl"
	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
	"github.com/jakeschepis/sageo-cli/internal/gsc"
	"github.com/jakeschepis/sageo-cli/internal/llm"
	_ "github.com/jakeschepis/sageo-cli/internal/llm/providers"
	"github.com/jakeschepis/sageo-cli/internal/merge"
	"github.com/jakeschepis/sageo-cli/internal/pipeline"
	"github.com/jakeschepis/sageo-cli/internal/provider"
	_ "github.com/jakeschepis/sageo-cli/internal/provider/local"
	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/serp"
	serpdforseo "github.com/jakeschepis/sageo-cli/internal/serp/dataforseo"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// Stage name constants so skip/only/resume all agree.
const (
	stageCrawl       = "crawl"
	stageAudit       = "audit"
	stageGSC         = "gsc"
	stagePSI         = "psi"
	stageLabs        = "labs"
	stageSERP        = "serp"
	stageBacklinks   = "backlinks"
	stageAEO         = "aeo"
	stageAEOMentions = "aeo-mentions"
	stageMerge       = "merge"
	stageRecommend   = "recommendations"
	stageDraft       = "draft"
	stageForecast    = "forecast"
	stageReviewGate  = "review_gate"
)

// NewRunCmd returns the top-level `sageo run <url>` command that orchestrates
// the full crawl → audit → data pulls → merge → recommend → draft → forecast
// pipeline in a single invocation.
func NewRunCmd(format *string, verbose *bool) *cobra.Command {
	var (
		budgetUSD      float64
		skipFlag       []string
		onlyFlag       []string
		maxPages       int
		promptsFile    string
		dryRun         bool
		approve        bool
		resume         bool
		noReview       bool
		autoApproveAll bool
		noSnapshot     bool
		retainN        int
		retainWithin   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "run <url>",
		Short: "End-to-end autonomous run: crawl → audit → GSC → PSI → SERP → Labs → backlinks → AEO → merge → recommend → draft → forecast",
		Long: `End-to-end autonomous run: crawl → audit → GSC → PSI → SERP → Labs →
backlinks → AEO → merge → recommend → draft → forecast.

BEFORE YOU RUN:
  1. sageo init --url <site>   (creates .sageo/state.json)
  2. sageo auth login gsc      (OAuth; needed for search-volume signal)
  3. sageo gsc sites use <property>   (MANDATORY — without it every
     forecast collapses to priority_tier=unknown, and sageo run will
     abort with GSC_NOT_CONFIGURED. Opt out with --skip gsc.)

TYPICAL FLOW:
  sageo run <url> --budget 10
  sageo recommendations review
  sageo report html --open

Run 'sageo doctor' to verify the environment before starting.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]
			if _, err := url.Parse(targetURL); err != nil || !strings.HasPrefix(targetURL, "http") {
				return output.PrintCodedErrorWithHint(output.ErrInvalidURL, "invalid target URL",
					"Use a full URL, for example: sageo run https://example.com",
					err, nil, output.Format(*format))
			}

			// Ensure project is initialised so every stage can persist state.
			if !state.Exists(".") {
				if _, err := state.Init(".", targetURL); err != nil {
					return output.PrintCodedError("STATE_INIT_FAILED", "failed to initialise project state", err, nil, output.Format(*format))
				}
			}

			// Load prompts for AEO stage.
			prompts, err := loadRunPrompts(promptsFile)
			if err != nil {
				return output.PrintCodedError("RUN_PROMPTS_FAILED", "failed to load prompts", err, nil, output.Format(*format))
			}

			// Pre-flight: abort up front when the GSC stage will run blind,
			// because a missing GSC property is a silent but catastrophic
			// failure (every forecast drops to priority_tier: unknown with "no
			// search-volume signal"). `--skip gsc` is the documented opt-out.
			if pfErr := preflightGSCCheck(toSet(skipFlag), toSet(onlyFlag), loadedGSCProperty()); pfErr != nil {
				return output.PrintCodedErrorWithHint(
					output.ErrGSCNotConfigured,
					"GSC stage would run without an active property — refusing to produce unknown-tier recommendations",
					pfErr.Hint(),
					pfErr, nil, output.Format(*format),
				)
			}

			cfgRun := pipeline.Config{
				WorkDir:            ".",
				BudgetUSD:          budgetUSD,
				Skip:               toSet(skipFlag),
				Only:               toSet(onlyFlag),
				Approve:            approve,
				Resume:             resume,
				DryRun:             dryRun,
				Verbose:            *verbose,
				Out:                cmd.ErrOrStderr(),
				NoSnapshot:         noSnapshot,
				SnapshotKeepLastN:  retainN,
				SnapshotKeepWithin: retainWithin,
			}

			stages := buildRunStages(targetURL, maxPages, prompts)
			stages = append(stages, stageReviewGateImpl(noReview, autoApproveAll, cmd.InOrStdin(), cmd.ErrOrStderr()))

			res, runErr := pipeline.Run(cmd.Context(), cfgRun, stages)

			// Compute summary fields even if we errored partway.
			recCount := 0
			var totalLow, totalHigh int
			tierHigh, tierMedium, tierLow, tierUnknown := 0, 0, 0, 0
			if st, err := state.Load("."); err == nil {
				recCount = len(st.Recommendations)
				for _, r := range st.Recommendations {
					if r.ForecastedLift != nil {
						totalLow += r.ForecastedLift.Low()
						totalHigh += r.ForecastedLift.High()
						switch r.ForecastedLift.PriorityTier {
						case state.PriorityHigh:
							tierHigh++
						case state.PriorityMedium:
							tierMedium++
						case state.PriorityLow:
							tierLow++
						default:
							tierUnknown++
						}
					} else {
						tierUnknown++
					}
				}
			}

			meta := map[string]any{
				"source":                "pipeline",
				"stages_run":            res.StagesRun,
				"skipped":               res.Skipped,
				"total_cost_usd":        res.TotalCostUSD,
				"recommendations_count": recCount,
				"estimated_range_low":   totalLow,
				"estimated_range_high":  totalHigh,
				"tier_high":             tierHigh,
				"tier_medium":           tierMedium,
				"tier_low":              tierLow,
				"tier_unknown":          tierUnknown,
				"outcome":               res.Outcome,
				"fetched_at":            time.Now().UTC().Format(time.RFC3339),
			}
			if res.FailedStage != "" {
				meta["failed_stage"] = res.FailedStage
			}

			if runErr != nil {
				meta["error"] = runErr.Error()
				meta["hint"] = "resume with `sageo run " + targetURL + " --resume`"
				return output.PrintCodedError("PIPELINE_FAILED", "pipeline failed: "+res.FailedStage, runErr, meta, output.Format(*format))
			}

			if !dryRun && res.Outcome != "failed" {
				printNextSteps(cmd.ErrOrStderr(), []string{
					"sageo recommendations review",
					"sageo report html --open",
				})
			}
			return output.PrintSuccess(res, meta, output.Format(*format))
		},
	}

	cmd.Flags().Float64Var(&budgetUSD, "budget", 0, "Hard ceiling for total paid API spend across the run (USD). 0 = no cap.")
	cmd.Flags().StringSliceVar(&skipFlag, "skip", nil, "Comma-separated stages to skip (e.g. --skip backlinks,aeo)")
	cmd.Flags().StringSliceVar(&onlyFlag, "only", nil, "Comma-separated stages to include exclusively")
	cmd.Flags().IntVar(&maxPages, "max-pages", 100, "Maximum pages to crawl")
	cmd.Flags().StringVar(&promptsFile, "prompts", "", "Newline-delimited file of prompts for the AEO stage. If empty, prompts are derived from GSC + Labs.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate only — do not call paid APIs")
	cmd.Flags().BoolVar(&approve, "approve", false, "Pre-approve all cost gates (no mid-run prompts)")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume from the last successful stage recorded in state")
	cmd.Flags().BoolVar(&noReview, "no-review", false, "Skip the interactive review gate — drafts remain pending_review until `sageo recommendations review` is run")
	cmd.Flags().BoolVar(&autoApproveAll, "auto-approve-all", false, "UNSAFE: mass-approve every LLM draft. Do not use for client-facing reports.")
	cmd.Flags().BoolVar(&noSnapshot, "no-snapshot", false, "Skip writing a per-run snapshot under .sageo/snapshots/")
	cmd.Flags().IntVar(&retainN, "retain", 20, "Keep the last N snapshots after this run; older ones are pruned")
	cmd.Flags().DurationVar(&retainWithin, "retain-within", 90*24*time.Hour, "Also keep snapshots within this duration (e.g. 90d via 2160h)")

	return cmd
}

// toSet converts a CLI slice into a lookup set.
func toSet(xs []string) map[string]bool {
	if len(xs) == 0 {
		return nil
	}
	out := make(map[string]bool, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x != "" {
			out[x] = true
		}
	}
	return out
}

// loadRunPrompts reads newline-delimited prompts from file (if given).
// Returns nil when file is empty; the AEO stage derives prompts from state.
func loadRunPrompts(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var prompts []string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prompts = append(prompts, line)
	}
	return prompts, scan.Err()
}

// preflightError is returned by preflightGSCCheck when the gsc stage would
// run without the config it needs. It carries a Hint() that names the three
// commands required to fix the state.
type preflightError struct {
	msg  string
	hint string
}

func (e *preflightError) Error() string { return e.msg }
func (e *preflightError) Hint() string  { return e.hint }

// loadedGSCProperty is an indirection that lets tests inject a property
// without relying on the global config path.
var loadedGSCProperty = func() string {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.GSCProperty
}

// preflightGSCCheck aborts `sageo run` when the GSC stage would run without
// an active property. A silent skip is catastrophic: every downstream
// forecast collapses to priority_tier=unknown and recommendations lose
// their search-volume and position signals. This was the root cause of the
// baysidebuilderswa.com.au debacle. Returns nil when --skip gsc or --only
// excludes gsc, or when a property is already configured.
func preflightGSCCheck(skip, only map[string]bool, gscProperty string) *preflightError {
	if skip[stageGSC] {
		return nil
	}
	if len(only) > 0 && !only[stageGSC] {
		return nil
	}
	if gscProperty != "" {
		return nil
	}
	return &preflightError{
		msg: "no GSC property configured; forecasts would collapse to priority_tier=unknown",
		hint: "Run: sageo auth login gsc && sageo gsc sites list && sageo gsc sites use <property>. " +
			"Or re-run with --skip gsc to proceed without GSC data.",
	}
}

// buildRunStages wires the concrete stage list used by `sageo run`.
func buildRunStages(targetURL string, maxPages int, prompts []string) []pipeline.Stage {
	return []pipeline.Stage{
		stageCrawlImpl(targetURL, maxPages),
		stageAuditImpl(),
		stageGSCSync(),
		stagePSIAnalyze(),
		stageLabsOverview(),
		stageSERPBatchTop(),
		stageBacklinksSummary(),
		stageAEOFanOut(prompts),
		stageAEOMentionsScan(),
		stageMergeImpl(),
		stageRecommendImpl(),
		stageDraftImpl(),
		stageForecastImpl(),
	}
}

// --- Stage implementations. Each mutates *state.State directly. ---

func stageCrawlImpl(targetURL string, maxPages int) pipeline.Stage {
	return pipeline.Stage{
		Name: stageCrawl,
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			fetcher, err := provider.NewFetcher(cfg.ActiveProvider)
			if err != nil {
				return fmt.Errorf("provider: %w", err)
			}
			svc := crawl.NewService(fetcher)
			res, err := svc.Run(ctx, crawl.Request{TargetURL: targetURL, Depth: 2, MaxPages: maxPages})
			if err != nil {
				return err
			}
			s.LastCrawl = time.Now().UTC().Format(time.RFC3339)
			s.PagesCrawled = len(res.Pages)
			s.AddHistory("pipeline.crawl", fmt.Sprintf("pages=%d errors=%d", len(res.Pages), len(res.Errors)))
			// Persist a tiny crawl fingerprint as meta; audit re-runs the crawl
			// inline since audit.Service operates on a CrawlResult. We cache the
			// fetcher by rerunning quickly; acceptable because both use local
			// caching HTTP provider. Record so audit can rebuild.
			return nil
		},
	}
}

func stageAuditImpl() pipeline.Stage {
	return pipeline.Stage{
		Name: stageAudit,
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			fetcher, err := provider.NewFetcher(cfg.ActiveProvider)
			if err != nil {
				return fmt.Errorf("provider: %w", err)
			}
			// Re-crawl (fast: most providers cache) to feed audit.
			crawler := crawl.NewService(fetcher)
			crawlRes, err := crawler.Run(ctx, crawl.Request{TargetURL: s.Site, Depth: 2, MaxPages: max(s.PagesCrawled, 50)})
			if err != nil {
				return err
			}
			auditSvc := audit.NewService()
			res, err := auditSvc.Run(ctx, audit.Request{CrawlResult: crawlRes})
			if err != nil {
				return err
			}
			findings := make([]state.Finding, 0, len(res.Issues))
			for _, i := range res.Issues {
				findings = append(findings, state.Finding{
					Rule: i.Rule, URL: i.URL, Value: i.Message,
					Verdict: string(i.Severity), Why: i.Why, Fix: i.Fix,
				})
			}
			s.UpdateAudit(res.Score, res.PageCount, findings)
			s.AddHistory("pipeline.audit", fmt.Sprintf("score=%.1f issues=%d", res.Score, len(findings)))
			return nil
		},
	}
}

func stageGSCSync() pipeline.Stage {
	return pipeline.Stage{
		Name: stageGSC,
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			if cfg.GSCProperty == "" {
				s.AddHistory("pipeline.gsc", "skipped: no gsc_property configured (run 'sageo gsc sites use <property>')")
				fmt.Fprintln(os.Stderr, "!! gsc stage skipped: no active property. Forecasts will have no search-volume signal.")
				fmt.Fprintln(os.Stderr, "!! Fix: sageo gsc sites use <property>  (use --skip gsc to suppress)")
				return nil
			}
			token, err := resolveGSCAccessToken()
			if err != nil || token == "" {
				s.AddHistory("pipeline.gsc", "skipped: not authenticated (run 'sageo auth login gsc')")
				fmt.Fprintln(os.Stderr, "!! gsc stage skipped: not authenticated. Run 'sageo auth login gsc' first (use --skip gsc to suppress).")
				return nil
			}
			client := gsc.NewClient(token)
			start := time.Now().AddDate(0, 0, -28).Format("2006-01-02")
			end := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

			pagesResp, err := client.QueryPages(gsc.QueryRequest{
				SiteURL: cfg.GSCProperty, StartDate: start, EndDate: end,
				SearchType: "web", RowLimit: 100,
			})
			if err != nil {
				return fmt.Errorf("gsc pages: %w", err)
			}
			kwResp, err := client.QueryKeywords(gsc.QueryRequest{
				SiteURL: cfg.GSCProperty, StartDate: start, EndDate: end,
				SearchType: "web", RowLimit: 100,
			})
			if err != nil {
				return fmt.Errorf("gsc keywords: %w", err)
			}
			if s.GSC == nil {
				s.GSC = &state.GSCData{}
			}
			s.GSC.Property = cfg.GSCProperty
			s.GSC.LastPull = time.Now().UTC().Format(time.RFC3339)
			s.GSC.TopPages = convertGSCRows(pagesResp.Rows)
			s.GSC.TopKeywords = convertGSCRows(kwResp.Rows)
			s.AddHistory("pipeline.gsc", fmt.Sprintf("pages=%d keywords=%d", len(s.GSC.TopPages), len(s.GSC.TopKeywords)))
			return nil
		},
	}
}

func convertGSCRows(rows []gsc.QueryRow) []state.GSCRow {
	out := make([]state.GSCRow, 0, len(rows))
	for _, r := range rows {
		key := ""
		if len(r.Keys) > 0 {
			key = r.Keys[0]
		}
		out = append(out, state.GSCRow{
			Key: key, Clicks: r.Clicks, Impressions: r.Impressions,
			CTR: r.CTR, Position: r.Position,
		})
	}
	return out
}

func stagePSIAnalyze() pipeline.Stage {
	return pipeline.Stage{
		Name: stagePSI,
		Run: func(ctx context.Context, s *state.State) error {
			// Reuse runPSIForPages from audit.go.
			var urls []string
			if s.GSC != nil {
				for _, p := range s.GSC.TopPages {
					if p.Key != "" {
						urls = append(urls, p.Key)
					}
					if len(urls) >= psiPageCap {
						break
					}
				}
			}
			if len(urls) == 0 {
				// Fall back to the site root.
				if s.Site != "" {
					urls = []string{s.Site}
				}
			}
			if len(urls) == 0 {
				s.AddHistory("pipeline.psi", "skipped: no URLs available")
				return nil
			}
			results := runPSIForPages(ctx, urls, os.Stderr)
			if len(results) == 0 {
				s.AddHistory("pipeline.psi", "no results")
				return nil
			}
			if s.PSI == nil {
				s.PSI = &state.PSIData{}
			}
			for _, r := range results {
				upsertPSIResult(s.PSI, r)
			}
			s.PSI.LastRun = time.Now().UTC().Format(time.RFC3339)
			s.AddHistory("pipeline.psi", fmt.Sprintf("pages=%d", len(results)))
			return nil
		},
	}
}

func stageLabsOverview() pipeline.Stage {
	return pipeline.Stage{
		Name: stageLabs,
		Paid: true,
		EstimateUSD: func(*state.State) float64 {
			// overview ($0.01) + ranked-keywords (~$0.01)
			return 0.02
		},
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				s.AddHistory("pipeline.labs", "skipped: no DataForSEO credentials")
				return nil
			}
			target := bareDomain(s.Site)
			if target == "" {
				s.AddHistory("pipeline.labs", "skipped: no site domain")
				return nil
			}
			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			// Ranked keywords → persist to state.
			task := map[string]any{
				"target":        target,
				"location_name": "Australia",
				"language_code": "en",
				"limit":         50,
			}
			raw, err := client.Post("/v3/dataforseo_labs/google/ranked_keywords/live", []map[string]any{task})
			if err != nil {
				return fmt.Errorf("labs ranked_keywords: %w", err)
			}
			keywords := parseLabsRankedKeywords(raw)
			if s.Labs == nil {
				s.Labs = &state.LabsData{}
			}
			s.Labs.Target = target
			s.Labs.LastRun = time.Now().UTC().Format(time.RFC3339)
			if len(keywords) > 0 {
				s.Labs.Keywords = keywords
			}
			s.AddHistory("pipeline.labs", fmt.Sprintf("keywords=%d", len(keywords)))
			return nil
		},
	}
}

// parseLabsRankedKeywords extracts the keyword items DataForSEO returns.
func parseLabsRankedKeywords(raw []byte) []state.LabsKeyword {
	var env struct {
		Tasks []struct {
			Result []struct {
				Items []struct {
					KeywordData struct {
						Keyword     string `json:"keyword"`
						KeywordInfo struct {
							SearchVolume int     `json:"search_volume"`
							CPC          float64 `json:"cpc"`
						} `json:"keyword_info"`
						KeywordProperties struct {
							Difficulty float64 `json:"keyword_difficulty"`
						} `json:"keyword_properties"`
						SearchIntentInfo struct {
							MainIntent string `json:"main_intent"`
						} `json:"search_intent_info"`
					} `json:"keyword_data"`
					RankedSerpElement struct {
						SerpItem struct {
							RankGroup int `json:"rank_group"`
						} `json:"serp_item"`
					} `json:"ranked_serp_element"`
				} `json:"items"`
			} `json:"result"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil
	}
	var out []state.LabsKeyword
	for _, t := range env.Tasks {
		for _, r := range t.Result {
			for _, it := range r.Items {
				out = append(out, state.LabsKeyword{
					Keyword:      it.KeywordData.Keyword,
					SearchVolume: it.KeywordData.KeywordInfo.SearchVolume,
					Difficulty:   it.KeywordData.KeywordProperties.Difficulty,
					CPC:          it.KeywordData.KeywordInfo.CPC,
					Intent:       it.KeywordData.SearchIntentInfo.MainIntent,
					Position:     it.RankedSerpElement.SerpItem.RankGroup,
				})
			}
		}
	}
	return out
}

func stageSERPBatchTop() pipeline.Stage {
	return pipeline.Stage{
		Name: stageSERP,
		Paid: true,
		EstimateUSD: func(s *state.State) float64 {
			n := pickTopQueries(s, 20)
			return float64(len(n)) * 0.0006
		},
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				s.AddHistory("pipeline.serp", "skipped: no DataForSEO credentials")
				return nil
			}
			queries := pickTopQueries(s, 20)
			if len(queries) == 0 {
				s.AddHistory("pipeline.serp", "skipped: no queries to analyse")
				return nil
			}
			adapter := serpdforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)
			reqs := make([]serp.AnalyzeRequest, len(queries))
			for i, q := range queries {
				reqs[i] = serp.AnalyzeRequest{Query: q}
			}
			resps, err := adapter.AnalyzeBatch(reqs)
			if err != nil && len(resps) == 0 {
				return err
			}
			for _, r := range resps {
				if r != nil {
					persistSERPToState(s, r)
				}
			}
			s.AddHistory("pipeline.serp", fmt.Sprintf("queries=%d", len(resps)))
			return nil
		},
	}
}

func pickTopQueries(s *state.State, limit int) []string {
	if s == nil || s.GSC == nil {
		return nil
	}
	rows := append([]state.GSCRow(nil), s.GSC.TopKeywords...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Impressions > rows[j].Impressions })
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]string, 0, len(rows))
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Key == "" || seen[r.Key] {
			continue
		}
		seen[r.Key] = true
		out = append(out, r.Key)
	}
	return out
}

func stageBacklinksSummary() pipeline.Stage {
	return pipeline.Stage{
		Name: stageBacklinks,
		Paid: true,
		EstimateUSD: func(*state.State) float64 {
			return 0.02
		},
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				s.AddHistory("pipeline.backlinks", "skipped: no DataForSEO credentials")
				return nil
			}
			target := bareDomain(s.Site)
			if target == "" {
				s.AddHistory("pipeline.backlinks", "skipped: no domain")
				return nil
			}
			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)
			raw, err := client.Post("/v3/backlinks/summary/live", []map[string]any{{"target": target}})
			if err != nil {
				return fmt.Errorf("backlinks summary: %w", err)
			}
			var env struct {
				Tasks []struct {
					Result []struct {
						TotalBacklinks           int64   `json:"total_backlinks"`
						TotalReferringDomains    int64   `json:"referring_domains"`
						BrokenBacklinks          int64   `json:"broken_backlinks"`
						ReferringDomainsNofollow int64   `json:"referring_domains_nofollow"`
						BacklinksSpamScore       float64 `json:"backlinks_spam_score"`
						Rank                     int     `json:"rank"`
						BacklinksNofollow        int64   `json:"backlinks_nofollow"`
					} `json:"result"`
				} `json:"tasks"`
			}
			if err := json.Unmarshal(raw, &env); err == nil && len(env.Tasks) > 0 && len(env.Tasks[0].Result) > 0 {
				r := env.Tasks[0].Result[0]
				s.Backlinks = &state.BacklinksData{
					LastRun:               time.Now().UTC().Format(time.RFC3339),
					Target:                target,
					TotalBacklinks:        r.TotalBacklinks,
					TotalReferringDomains: r.TotalReferringDomains,
					BrokenBacklinks:       r.BrokenBacklinks,
					Rank:                  r.Rank,
					DoFollow:              r.TotalBacklinks - r.BacklinksNofollow,
					NoFollow:              r.BacklinksNofollow,
					SpamScore:             r.BacklinksSpamScore,
				}
			}
			s.AddHistory("pipeline.backlinks", "summary persisted")
			return nil
		},
	}
}

func stageAEOFanOut(overridePrompts []string) pipeline.Stage {
	return pipeline.Stage{
		Name: stageAEO,
		Paid: true,
		EstimateUSD: func(s *state.State) float64 {
			prompts := aeoPromptsForRun(s, overridePrompts)
			// 4 engines × per-query cost
			return float64(len(prompts)) * perQueryCostUSD * float64(len(allAEOEngines))
		},
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				s.AddHistory("pipeline.aeo", "skipped: no DataForSEO credentials")
				return nil
			}
			prompts := aeoPromptsForRun(s, overridePrompts)
			if len(prompts) == 0 {
				s.AddHistory("pipeline.aeo", "skipped: no prompts")
				return nil
			}
			specs := make([]aeoQuerySpec, 0, len(allAEOEngines))
			for _, e := range allAEOEngines {
				name, err := flagshipModelNameForEngine(e)
				if err != nil {
					continue
				}
				specs = append(specs, aeoQuerySpec{Engine: e, ModelName: name})
			}
			if len(specs) == 0 {
				s.AddHistory("pipeline.aeo", "skipped: no engines resolved")
				return nil
			}
			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)
			totalOK := 0
			for _, p := range prompts {
				outcomes := fanOutAEOQueries(client, p, specs, 4)
				var persisted []state.AEOResponseResult
				for _, o := range outcomes {
					if o.Error != "" {
						continue
					}
					persisted = append(persisted, state.AEOResponseResult{
						Engine: o.Engine, ModelName: o.ModelName,
						Response: o.Response, FetchedAt: time.Now().UTC(),
					})
				}
				if len(persisted) > 0 {
					s.UpsertAEOResponses(p, persisted)
					totalOK += len(persisted)
				}
			}
			s.AddHistory("pipeline.aeo", fmt.Sprintf("prompts=%d responses=%d", len(prompts), totalOK))
			return nil
		},
	}
}

// aeoPromptsForRun returns the list of prompts to send through the AEO fan-out.
// If override is non-empty it wins. Otherwise, derive up to 10 prompts from top
// GSC queries and Labs keywords (commercial/informational intent preferred).
func aeoPromptsForRun(s *state.State, override []string) []string {
	if len(override) > 0 {
		return override
	}
	var prompts []string
	seen := map[string]bool{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		prompts = append(prompts, p)
	}
	if s.GSC != nil {
		rows := append([]state.GSCRow(nil), s.GSC.TopKeywords...)
		sort.Slice(rows, func(i, j int) bool { return rows[i].Clicks > rows[j].Clicks })
		for _, r := range rows {
			add(r.Key)
			if len(prompts) >= 10 {
				return prompts
			}
		}
	}
	if s.Labs != nil {
		for _, k := range s.Labs.Keywords {
			add(k.Keyword)
			if len(prompts) >= 10 {
				return prompts
			}
		}
	}
	return prompts
}

func stageAEOMentionsScan() pipeline.Stage {
	return pipeline.Stage{
		Name: stageAEOMentions,
		Run: func(ctx context.Context, s *state.State) error {
			if s.AEO == nil || len(s.AEO.Responses) == 0 {
				s.AddHistory("pipeline.aeo-mentions", "skipped: no AEO responses")
				return nil
			}
			// We cannot directly call mentions.DetectInResponses without the
			// import cycle; use the existing helper via collectBrandTerms.
			terms := collectBrandTerms(s, nil)
			if len(terms) == 0 {
				s.AddHistory("pipeline.aeo-mentions", "skipped: no brand terms")
				return nil
			}
			// Defer to the existing mentions package via the CLI-local helper.
			// We re-implement inline here: a simple substring scan per term.
			byTerm := map[string][]state.LocalMentionMatch{}
			for _, prompt := range s.AEO.Responses {
				for _, r := range prompt.Results {
					lower := strings.ToLower(r.Response)
					for _, term := range terms {
						t := strings.ToLower(term)
						if t == "" {
							continue
						}
						count := strings.Count(lower, t)
						if count == 0 {
							continue
						}
						byTerm[term] = append(byTerm[term], state.LocalMentionMatch{
							Engine: r.Engine, ModelName: r.ModelName,
							Prompt: prompt.Prompt, Term: term, Count: count,
						})
					}
				}
			}
			for term, list := range byTerm {
				existing := findMentionsData(s, term)
				existing.Term = term
				existing.LocalMatches = list
				s.UpsertMentions(existing)
			}
			s.AddHistory("pipeline.aeo-mentions", fmt.Sprintf("terms=%d", len(byTerm)))
			return nil
		},
	}
}

func stageMergeImpl() pipeline.Stage {
	return pipeline.Stage{
		Name: stageMerge,
		Run: func(ctx context.Context, s *state.State) error {
			findings := merge.Run(s)
			if findings == nil {
				findings = []merge.MergedFinding{}
			}
			raw, err := json.Marshal(findings)
			if err != nil {
				return err
			}
			s.MergedFindings = raw
			s.LastAnalysis = time.Now().UTC().Format(time.RFC3339)
			s.AddHistory("pipeline.merge", fmt.Sprintf("merged=%d", len(findings)))
			return nil
		},
	}
}

func stageRecommendImpl() pipeline.Stage {
	return pipeline.Stage{
		Name: stageRecommend,
		Run: func(ctx context.Context, s *state.State) error {
			var findings []merge.MergedFinding
			if len(s.MergedFindings) > 0 {
				_ = json.Unmarshal(s.MergedFindings, &findings)
			}
			if len(findings) == 0 {
				findings = merge.Run(s)
			}
			recs := merge.GenerateRecommendations(s, findings)
			recommendations.UpsertRecommendations(s, recs)
			s.AddHistory("pipeline.recommendations", fmt.Sprintf("recs=%d", len(recs)))
			return nil
		},
	}
}

func stageDraftImpl() pipeline.Stage {
	return pipeline.Stage{
		Name: stageDraft,
		Paid: true,
		EstimateUSD: func(s *state.State) float64 {
			n := 0
			for _, r := range s.Recommendations {
				if r.RecommendedValue == "" {
					n++
				}
			}
			return float64(n) * 0.02
		},
		Run: func(ctx context.Context, s *state.State) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			providerName := cfg.LLMProvider
			if providerName == "" {
				providerName = "anthropic"
			}
			p, err := llm.New(providerName, cfg)
			if err != nil {
				s.AddHistory("pipeline.draft", "skipped: "+err.Error())
				return nil
			}
			drafted := 0
			recs := append([]recommendations.Recommendation(nil), s.Recommendations...)
			for i := range recs {
				r := &recs[i]
				if r.RecommendedValue != "" {
					continue
				}
				page := recommendations.ContextForURL(s, r.TargetURL, r.TargetQuery)
				if err := recommendations.Draft(ctx, p, r, page); err != nil {
					continue
				}
				if r.RecommendedValue == "" {
					continue
				}
				drafted++
				recommendations.UpsertRecommendations(s, []recommendations.Recommendation{*r})
			}
			s.AddHistory("pipeline.draft", fmt.Sprintf("drafted=%d", drafted))
			return nil
		},
	}
}

func stageForecastImpl() pipeline.Stage {
	return pipeline.Stage{
		Name: stageForecast,
		Run: func(ctx context.Context, s *state.State) error {
			recs := recommendations.Load(s)
			if len(recs) == 0 {
				return nil
			}
			recommendations.AttachForecasts(s, recs)
			recommendations.UpsertRecommendations(s, recs)
			s.AddHistory("pipeline.forecast", fmt.Sprintf("forecasted=%d", len(recs)))
			return nil
		},
	}
}

// stageReviewGateImpl is the terminal stage that enforces the LLM review
// gate. It runs after draft/forecast have produced pending drafts.
//
//   - --auto-approve-all: bulk-approves every pending draft. Documented as
//     unsafe for client-facing reports.
//   - --no-review: leaves everything pending; the user must run
//     `sageo recommendations review` separately.
//   - default: in an interactive terminal, prompts "review now?" and defers
//     to the interactive review flow. Non-interactive (piped) runs fall
//     back to the --no-review path with a warning.
func stageReviewGateImpl(noReview, autoApproveAll bool, stdin io.Reader, stderr io.Writer) pipeline.Stage {
	return pipeline.Stage{
		Name: stageReviewGate,
		Run: func(ctx context.Context, s *state.State) error {
			pending := countPendingReview(s)
			if pending == 0 {
				s.AddHistory("pipeline.review_gate", "no pending drafts")
				return nil
			}
			if autoApproveAll {
				approveAllPending(s, "run --auto-approve-all")
				s.AddHistory("pipeline.review_gate", fmt.Sprintf("auto-approved=%d (unsafe flag)", pending))
				_, _ = fmt.Fprintf(stderr, "[review] auto-approved %d draft(s) via --auto-approve-all (unsafe for client reports)\n", pending)
				return nil
			}
			if noReview {
				s.AddHistory("pipeline.review_gate", fmt.Sprintf("deferred=%d (--no-review)", pending))
				_, _ = fmt.Fprintf(stderr, "[review] %d draft(s) left pending_review — run `sageo recommendations review` before shipping reports\n", pending)
				return nil
			}
			// Interactive default. If stdin is not a terminal, defer.
			if !isTerminal(stdin) {
				s.AddHistory("pipeline.review_gate", fmt.Sprintf("deferred=%d (non-interactive)", pending))
				_, _ = fmt.Fprintf(stderr, "[review] %d draft(s) pending; non-interactive session — run `sageo recommendations review` later\n", pending)
				return nil
			}
			_, _ = fmt.Fprintf(stderr, "[review] %d draft(s) pending review. Review now? [Y/n] ", pending)
			var ans string
			br := bufio.NewReader(stdin)
			line, _ := br.ReadString('\n')
			ans = strings.ToLower(strings.TrimSpace(line))
			if ans == "n" || ans == "no" {
				s.AddHistory("pipeline.review_gate", fmt.Sprintf("deferred=%d (user skipped)", pending))
				return nil
			}
			queue := buildReviewQueue(s, reviewFilter{})
			prompter := &huhReviewPrompter{}
			processed, err := runReviewLoop(s, queue, prompter, "cli", ".")
			if err != nil && !errors.Is(err, errReviewQuit) {
				return err
			}
			s.AddHistory("pipeline.review_gate", fmt.Sprintf("reviewed=%d of %d", processed, pending))
			return nil
		},
	}
}

// countPendingReview is a small helper to total pending-review recs.
func countPendingReview(s *state.State) int {
	if s == nil {
		return 0
	}
	n := 0
	for _, r := range s.Recommendations {
		if r.EffectiveReviewStatus() == state.ReviewPending {
			n++
		}
	}
	return n
}

// approveAllPending marks every pending-review rec as approved.
func approveAllPending(s *state.State, note string) {
	if s == nil {
		return
	}
	now := time.Now().UTC()
	for i := range s.Recommendations {
		r := &s.Recommendations[i]
		if r.EffectiveReviewStatus() != state.ReviewPending {
			continue
		}
		r.ReviewStatus = state.ReviewApproved
		t := now
		r.ReviewedAt = &t
		r.ReviewedBy = "cli"
		if note != "" {
			r.ReviewNotes = note
		}
	}
}

// isTerminal reports whether r is an interactive terminal. We deliberately
// avoid importing a new dep — golang.org/x/term is already in the module.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return xterm.IsTerminal(int(f.Fd()))
}

// max avoids importing "math" for a trivial helper.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
