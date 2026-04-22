package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/aeo"
	"github.com/jakeschepis/sageo-cli/internal/aeo/llmmentions"
	"github.com/jakeschepis/sageo-cli/internal/aeo/mentions"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/common/cost"
	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// perQueryCostUSD is the approximate cost of a single LLM responses/live call.
const perQueryCostUSD = 0.003

// perMentionsCostUSD is the approximate per-task cost for LLM Mentions
// endpoints. Live sample responses in the DataForSEO docs report ~$0.10 per
// task (see https://docs.dataforseo.com/v3/ai_optimization/llm_mentions/overview/
// and the response `cost` field); we use a conservative upper estimate here.
const perMentionsCostUSD = 0.10

// allAEOEngines is the canonical list of engines fan-out targets.
var allAEOEngines = []string{"chatgpt", "claude", "gemini", "perplexity"}

// NewAEOCmd returns the aeo command group.
func NewAEOCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aeo",
		Short: "Answer Engine Optimization commands",
		Long:  `Query AI engines and get keyword data for Answer Engine Optimization (AEO). Powered by DataForSEO.`,
	}

	cmd.AddCommand(
		newAEOResponsesCmd(format, verbose),
		newAEOKeywordsCmd(format, verbose),
		newAEOModelsCmd(format, verbose),
		newAEOMentionsCmd(format, verbose),
	)

	return cmd
}

// aeoQuerySpec is a single (engine, model_name) pair to fan out.
type aeoQuerySpec struct {
	Engine    string
	ModelName string
}

// aeoQueryOutcome mirrors the per-row output contract.
type aeoQueryOutcome struct {
	Engine    string  `json:"engine"`
	ModelName string  `json:"model_name"`
	Response  string  `json:"response"`
	Error     string  `json:"error"`
	CostUSD   float64 `json:"cost_usd"`
}

func newAEOResponsesCmd(format *string, verbose *bool) *cobra.Command {
	var prompt, engine, modelNameOverride, tier string
	var models []string
	var all, dryRun bool
	var concurrency int

	cmd := &cobra.Command{
		Use:   "responses",
		Short: "Query one or more AI engines and compare brand/topic responses",
		Long: `Send a prompt to one or more AI engines (ChatGPT, Claude, Gemini, Perplexity)
and see the full response from each. Useful for understanding how AI tools
describe your brand, products, or keywords.

Selection modes:
  --engine chatgpt                     single engine (flagship-tier model by default)
  --engine chatgpt --model-name gpt-5  single engine with explicit model
  --all                                fan out to all 4 engines
  --models gpt-5,claude-sonnet-4-6     explicit model list (engine inferred)

Supported engines: chatgpt, claude, gemini, perplexity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if prompt == "" {
				return output.PrintCodedError(output.ErrAEOFailed, "prompt is required",
					fmt.Errorf("use --prompt to specify a prompt"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				return output.PrintCodedError(output.ErrAEOFailed, "DataForSEO credentials not configured",
					fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
			}

			if tier != "flagship" && tier != "cheap" {
				return output.PrintCodedError(output.ErrAEOFailed,
					fmt.Sprintf("unsupported tier %q", tier),
					fmt.Errorf("valid values: flagship, cheap"), nil, output.Format(*format))
			}
			if concurrency < 1 {
				concurrency = 1
			}

			specs, specErr := resolveAEOQuerySpecs(engine, modelNameOverride, models, all, tier)
			if specErr != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "invalid model selection", specErr, nil, output.Format(*format))
			}
			if len(specs) == 0 {
				return output.PrintCodedError(output.ErrAEOFailed, "no engines selected",
					fmt.Errorf("provide --engine, --models, or --all"), nil, output.Format(*format))
			}

			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: perQueryCostUSD,
				Units:       len(specs),
				Basis:       fmt.Sprintf("dataforseo llm responses: %d quer(ies) @ ~$%.3f/query", len(specs), perQueryCostUSD),
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
				"concurrency":       concurrency,
				"tier":              tier,
			}
			isSingle := len(specs) == 1
			if isSingle {
				meta["model"] = specs[0].Engine
				meta["model_name"] = specs[0].ModelName
			}

			if dryRun {
				planned := make([]map[string]string, 0, len(specs))
				for _, s := range specs {
					planned = append(planned, map[string]string{"engine": s.Engine, "model_name": s.ModelName})
				}
				if isSingle {
					return output.PrintSuccess(map[string]any{
						"prompt": prompt,
						"model":  specs[0].Engine,
						"status": "dry_run",
					}, meta, output.Format(*format))
				}
				return output.PrintSuccess(map[string]any{
					"prompt":  prompt,
					"planned": planned,
					"status":  "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			outcomes := fanOutAEOQueries(client, prompt, specs, concurrency)

			meta["fetched_at"] = time.Now().Format(time.RFC3339)

			// Persist successful rows to state (best-effort).
			var persisted []state.AEOResponseResult
			for _, o := range outcomes {
				if o.Error != "" {
					continue
				}
				persisted = append(persisted, state.AEOResponseResult{
					Engine:    o.Engine,
					ModelName: o.ModelName,
					Response:  o.Response,
					FetchedAt: time.Now().UTC(),
				})
			}
			if len(persisted) > 0 {
				if st, loadErr := state.Load("."); loadErr == nil {
					st.UpsertAEOResponses(prompt, persisted)
					st.AddHistory("aeo", fmt.Sprintf("responses for %q: %d engines", prompt, len(persisted)))
					_ = st.Save(".")
				}
			}

			// Backward-compatible single-result shape.
			if isSingle {
				o := outcomes[0]
				if o.Error != "" {
					return output.PrintCodedError(output.ErrAEOFailed, "LLM responses request failed",
						fmt.Errorf("%s", o.Error), meta, output.Format(*format))
				}
				return output.PrintSuccess(map[string]any{
					"prompt":   prompt,
					"response": o.Response,
					"model":    o.ModelName,
				}, meta, output.Format(*format))
			}

			return output.PrintSuccess(map[string]any{
				"prompt":  prompt,
				"results": outcomes,
			}, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt to send to the AI engine (required)")
	cmd.Flags().StringVar(&engine, "engine", "", "AI engine: chatgpt, claude, gemini, perplexity")
	cmd.Flags().StringVar(&engine, "model", "", "(deprecated) alias for --engine")
	_ = cmd.Flags().MarkHidden("model")
	cmd.Flags().StringVar(&modelNameOverride, "model-name", "", "Override the specific model name for --engine (e.g. gpt-5)")
	cmd.Flags().StringSliceVar(&models, "models", nil, "Explicit list of models (engine inferred from name), e.g. gpt-5,claude-sonnet-4-6")
	cmd.Flags().BoolVar(&all, "all", false, "Fan out to all 4 engines using --tier model selection")
	cmd.Flags().StringVar(&tier, "tier", "flagship", "Model tier: flagship or cheap")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Max parallel requests")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

// resolveAEOQuerySpecs merges the flag surface into a concrete list of
// (engine, model_name) pairs to execute. Precedence: --models > --all > --engine.
func resolveAEOQuerySpecs(engine, modelNameOverride string, models []string, all bool, tier string) ([]aeoQuerySpec, error) {
	modelForEngine := func(e string) (string, error) {
		if tier == "cheap" {
			return defaultModelNameForEngine(e)
		}
		return flagshipModelNameForEngine(e)
	}

	if len(models) > 0 {
		specs := make([]aeoQuerySpec, 0, len(models))
		for _, m := range models {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			eng, err := inferEngineFromModelName(m)
			if err != nil {
				return nil, err
			}
			specs = append(specs, aeoQuerySpec{Engine: eng, ModelName: m})
		}
		return specs, nil
	}

	if all {
		specs := make([]aeoQuerySpec, 0, len(allAEOEngines))
		for _, e := range allAEOEngines {
			name, err := modelForEngine(e)
			if err != nil {
				return nil, err
			}
			specs = append(specs, aeoQuerySpec{Engine: e, ModelName: name})
		}
		return specs, nil
	}

	// Single engine path. Default to chatgpt if nothing supplied.
	e := engine
	if e == "" {
		e = "chatgpt"
	}
	validEngines := map[string]bool{"chatgpt": true, "claude": true, "gemini": true, "perplexity": true}
	if !validEngines[e] {
		return nil, fmt.Errorf("unsupported engine %q: valid values: chatgpt, claude, gemini, perplexity", e)
	}
	name := modelNameOverride
	if name == "" {
		resolved, err := modelForEngine(e)
		if err != nil {
			return nil, err
		}
		name = resolved
	}
	return []aeoQuerySpec{{Engine: e, ModelName: name}}, nil
}

// inferEngineFromModelName maps a DataForSEO model_name to an engine alias
// using simple prefix rules.
func inferEngineFromModelName(modelName string) (string, error) {
	lower := strings.ToLower(modelName)
	switch {
	case strings.HasPrefix(lower, "gpt"), strings.HasPrefix(lower, "o1"), strings.HasPrefix(lower, "o3"), strings.HasPrefix(lower, "o4"), strings.HasPrefix(lower, "chatgpt"):
		return "chatgpt", nil
	case strings.HasPrefix(lower, "claude"):
		return "claude", nil
	case strings.HasPrefix(lower, "gemini"):
		return "gemini", nil
	case strings.HasPrefix(lower, "sonar"), strings.HasPrefix(lower, "perplexity"):
		return "perplexity", nil
	default:
		return "", fmt.Errorf("cannot infer engine from model name %q (expected prefix gpt-, claude-, gemini-, or sonar-)", modelName)
	}
}

// fanOutAEOQueries runs each spec concurrently, bounded by a semaphore.
func fanOutAEOQueries(client *dataforseo.Client, prompt string, specs []aeoQuerySpec, concurrency int) []aeoQueryOutcome {
	// Lazily load the model catalogue so we can fall back to cached defaults
	// if a hardcoded flagship name is rejected by the API.
	catalogue, _ := aeo.LoadCached(aeoCacheDir())

	outcomes := make([]aeoQueryOutcome, len(specs))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, s := range specs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, s aeoQuerySpec) {
			defer wg.Done()
			defer func() { <-sem }()
			modelName := s.ModelName
			resp, err := queryLLMResponse(client, s.Engine, modelName, prompt)
			// Fallback chain on 40501 "Invalid Field: 'model_name'":
			// hardcoded flagship → cached default → first catalogue entry.
			if err != nil && strings.Contains(err.Error(), "40501") {
				if fallback, fbErr := aeo.DefaultModelName(catalogue, s.Engine); fbErr == nil && fallback != modelName {
					if retryResp, retryErr := queryLLMResponse(client, s.Engine, fallback, prompt); retryErr == nil {
						modelName = fallback
						resp = retryResp
						err = nil
					}
				}
			}
			out := aeoQueryOutcome{
				Engine:    s.Engine,
				ModelName: modelName,
				Response:  resp,
				CostUSD:   perQueryCostUSD,
			}
			if err != nil {
				out.Error = err.Error()
			}
			outcomes[i] = out
		}(i, s)
	}
	wg.Wait()
	return outcomes
}

// newAEOModelsCmd returns `sageo aeo models`, printing the cached DataForSEO
// AI optimization model catalogue (refreshing on demand).
func newAEOModelsCmd(format *string, verbose *bool) *cobra.Command {
	var engineFilter string
	var refresh bool
	var outFormat string

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List the DataForSEO AI optimization model catalogue",
		Long: `Print the model catalogue for each AI engine exposed by DataForSEO.

The catalogue is cached on disk for 7 days. Pass --refresh to force a refetch.
Flags:
  --engine <name>     filter to a single engine (chatgpt, claude, gemini, perplexity)
  --refresh           refetch from DataForSEO even if the cache is fresh
  --format json|text  output format (default json)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			cacheDir := aeoCacheDir()
			catalogue, err := aeo.LoadCached(cacheDir)
			if err != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "failed to load cached catalogue", err, nil, output.Format(*format))
			}

			needsFetch := refresh || len(catalogue) == 0
			source := "cache"
			fetchedAt := ""
			if needsFetch {
				if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
					return output.PrintCodedError(output.ErrAEOFailed, "DataForSEO credentials not configured",
						fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
				}
				client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)
				fresh, fetchErr := aeo.FetchAllModels(client)
				if fetchErr != nil && len(fresh) == 0 {
					return output.PrintCodedError(output.ErrAEOFailed, "failed to fetch model catalogue", fetchErr, nil, output.Format(*format))
				}
				if err := aeo.SaveCached(cacheDir, fresh); err != nil {
					return output.PrintCodedError(output.ErrAEOFailed, "failed to save catalogue cache", err, nil, output.Format(*format))
				}
				catalogue = fresh
				source = "dataforseo"
				fetchedAt = time.Now().UTC().Format(time.RFC3339)
			}

			if engineFilter != "" {
				filtered, ok := catalogue[engineFilter]
				if !ok {
					return output.PrintCodedError(output.ErrAEOFailed,
						fmt.Sprintf("no cached catalogue for engine %q", engineFilter),
						fmt.Errorf("run with --refresh to fetch"), nil, output.Format(*format))
				}
				catalogue = map[string][]aeo.Model{engineFilter: filtered}
			}

			meta := map[string]any{
				"source":  source,
				"cached":  source == "cache",
				"verbose": *verbose,
			}
			if fetchedAt != "" {
				meta["fetched_at"] = fetchedAt
			}

			formatSel := outFormat
			if formatSel == "" {
				formatSel = "json"
			}
			if formatSel == "text" {
				engines := make([]string, 0, len(catalogue))
				for e := range catalogue {
					engines = append(engines, e)
				}
				sort.Strings(engines)
				for _, e := range engines {
					fmt.Printf("%s:\n", e)
					for _, m := range catalogue[e] {
						defaultMark := ""
						if m.IsDefault {
							defaultMark = " [default]"
						}
						fmt.Printf("  - %s (%s)%s\n", m.ModelName, m.DisplayName, defaultMark)
					}
				}
				return nil
			}

			return output.PrintSuccess(map[string]any{
				"engines": catalogue,
			}, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&engineFilter, "engine", "", "Filter to a single engine: chatgpt, claude, gemini, perplexity")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refetch the catalogue from DataForSEO even if the cache is fresh")
	cmd.Flags().StringVar(&outFormat, "format", "json", "Output format: json or text")

	return cmd
}

// queryLLMResponse issues a single LLM responses/live request for one
// (engine, model_name) pair and returns the concatenated text response.
// See https://docs.dataforseo.com/v3/ai_optimization-chat_gpt-llm_responses-live/
func queryLLMResponse(client *dataforseo.Client, engine, modelName, prompt string) (string, error) {
	endpoint, err := aeoEndpointForModel(engine)
	if err != nil {
		return "", err
	}
	reqBody := []map[string]any{{
		"user_prompt": prompt,
		"model_name":  modelName,
	}}
	raw, err := client.Post(endpoint, reqBody)
	if err != nil {
		return "", err
	}

	var envelope struct {
		StatusCode    int    `json:"status_code"`
		StatusMessage string `json:"status_message"`
		Tasks         []struct {
			StatusCode    int    `json:"status_code"`
			StatusMessage string `json:"status_message"`
			Result        []struct {
				ModelName string `json:"model_name"`
				Items     []struct {
					Type     string `json:"type"`
					Sections []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"sections"`
				} `json:"items"`
			} `json:"result"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", fmt.Errorf("decode dataforseo response: %w", err)
	}
	if envelope.StatusCode != 20000 {
		return "", fmt.Errorf("dataforseo error %d: %s", envelope.StatusCode, envelope.StatusMessage)
	}
	if len(envelope.Tasks) == 0 {
		return "", fmt.Errorf("no tasks returned")
	}
	task := envelope.Tasks[0]
	if task.StatusCode != 20000 {
		return "", fmt.Errorf("dataforseo task error %d: %s", task.StatusCode, task.StatusMessage)
	}
	if len(task.Result) == 0 {
		return "", fmt.Errorf("no results returned")
	}
	result := task.Result[0]
	var b strings.Builder
	for _, item := range result.Items {
		for _, section := range item.Sections {
			if section.Type == "text" {
				b.WriteString(section.Text)
			}
		}
	}
	return b.String(), nil
}

func newAEOKeywordsCmd(format *string, verbose *bool) *cobra.Command {
	var keyword, location, language string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "keywords",
		Short: "Get AI search volume for keywords",
		Long:  `Retrieve AI search volume data showing how often keywords are used in AI tools like ChatGPT and Gemini.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if keyword == "" {
				return output.PrintCodedError(output.ErrAEOFailed, "keyword is required",
					fmt.Errorf("use --keyword to specify a keyword"), nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
				return output.PrintCodedError(output.ErrAEOFailed, "DataForSEO credentials not configured",
					fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
			}

			// Estimate: $0.01/task
			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: 0.01,
				Units:       1,
				Basis:       "dataforseo ai keyword data: 1 task @ $0.01/task",
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

			// DataForSEO expects `keywords` (plural, array) and requires
			// `location_name` or `location_code`. Default to United States /
			// English when not provided so the command works out of the box.
			// See: https://docs.dataforseo.com/v3/ai_optimization-ai_keyword_data-keywords_search_volume-live/
			if location == "" {
				location = "United States"
			}
			if language == "" {
				language = "en"
			}
			task := map[string]any{
				"keywords":      []string{keyword},
				"location_name": location,
				"language_code": language,
			}

			raw, err := client.Post("/v3/ai_optimization/ai_keyword_data/keywords_search_volume/live", []map[string]any{task})
			if err != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "AI keyword data request failed", err, nil, output.Format(*format))
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
				return output.PrintCodedError(output.ErrAEOFailed, "failed to decode DataForSEO response", jsonErr, nil, output.Format(*format))
			}

			if envelope.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrAEOFailed,
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

	cmd.Flags().StringVar(&keyword, "keyword", "", "Keyword to get AI search volume for (required)")
	cmd.Flags().StringVar(&location, "location", "", "Location name (e.g. 'United States')")
	cmd.Flags().StringVar(&language, "language", "", "Language code (e.g. 'en')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
}

// aeoEndpointForModel returns the DataForSEO LLM responses endpoint for the given engine.
func aeoEndpointForModel(engine string) (string, error) {
	seg := aeo.EngineToPath(engine)
	if seg == "" {
		return "", fmt.Errorf("unsupported model %q: valid values: chatgpt, claude, gemini, perplexity", engine)
	}
	return "/v3/ai_optimization/" + seg + "/llm_responses/live", nil
}

// aeoCacheDir returns the on-disk directory used for the AEO model catalogue.
func aeoCacheDir() string {
	return filepath.Join(filepath.Dir(config.Path()), "cache")
}

// defaultModelNameForEngine returns the "cheap" tier DataForSEO `model_name`
// for each engine. DataForSEO requires this field on every
// llm_responses/live request; without it the API rejects the task with
// 40501 "Invalid Field: 'model_name'". The full model catalog per engine is
// available at /v3/ai_optimization/<engine>/llm_responses/models.
func defaultModelNameForEngine(engine string) (string, error) {
	switch engine {
	case "chatgpt":
		return "gpt-4o-mini", nil
	case "claude":
		return "claude-haiku-4-5", nil
	case "gemini":
		return "gemini-2.5-flash-lite", nil
	case "perplexity":
		return "sonar", nil
	default:
		return "", fmt.Errorf("unsupported engine %q: valid values: chatgpt, claude, gemini, perplexity", engine)
	}
}

// flagshipModelNameForEngine returns the "flagship" tier DataForSEO
// `model_name` for each engine. These are the most capable models currently
// exposed by each provider via DataForSEO. If a specific flagship name is not
// (yet) available in the DataForSEO catalogue, DataForSEO will reject the
// request with 40501 — callers should then fall back to the cheap tier or
// consult /v3/ai_optimization/<engine>/llm_responses/models.
func flagshipModelNameForEngine(engine string) (string, error) {
	switch engine {
	case "chatgpt":
		return "gpt-5", nil
	case "claude":
		return "claude-sonnet-4-6", nil
	case "gemini":
		return "gemini-3-pro", nil
	case "perplexity":
		return "sonar-pro", nil
	default:
		return "", fmt.Errorf("unsupported engine %q: valid values: chatgpt, claude, gemini, perplexity", engine)
	}
}

// newAEOMentionsCmd returns the `sageo aeo mentions` command group: three
// DataForSEO LLM Mentions API wrappers (search, top-pages, top-domains) plus
// a local scan over state.AEOData.Responses.
func newAEOMentionsCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mentions",
		Short: "Detect brand mentions in AI responses (local scan + DataForSEO LLM Mentions API)",
		Long: `Answer: when users ask AI about our topic, does our brand show up, and how often?

Two layers:
  scan         Layer A: grep brand terms in AI responses already stored by 'sageo aeo responses'.
  search       Layer B: DataForSEO LLM Mentions search (detailed mentions, ai_search_volume).
  top-pages    Layer B: pages AI engines cite for a brand/keyword.
  top-domains  Layer B: domains AI engines cite for a brand/keyword.`,
	}
	cmd.AddCommand(
		newAEOMentionsSearchCmd(format, verbose),
		newAEOMentionsTopPagesCmd(format, verbose),
		newAEOMentionsTopDomainsCmd(format, verbose),
		newAEOMentionsScanCmd(format, verbose),
	)
	return cmd
}

// mentionsRunContext captures resolved config + cost state used by every
// Layer B mentions subcommand.
type mentionsRunContext struct {
	client *llmmentions.Client
	meta   map[string]any
	dryRun bool
}

// setupMentionsRun centralises the config load, credential check, cost
// estimate, approval gate, and dry-run metadata shared by search/top-pages/
// top-domains. Callers should check ctx==nil to know the command already
// emitted a response (dry-run or error) and should return nil.
func setupMentionsRun(format *string, verbose *bool, endpoint, term string, dryRun bool) (*mentionsRunContext, error) {
	if term == "" {
		return nil, output.PrintCodedError(output.ErrAEOFailed, "--term is required",
			fmt.Errorf("pass --term to specify the brand or keyword"), nil, output.Format(*format))
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
	}
	if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
		return nil, output.PrintCodedError(output.ErrAEOFailed, "DataForSEO credentials not configured",
			fmt.Errorf("run 'sageo login' and select DataForSEO to configure credentials"), nil, output.Format(*format))
	}

	estimate, err := cost.BuildEstimate(cost.EstimateInput{
		UnitCostUSD: perMentionsCostUSD,
		Units:       1,
		Basis:       fmt.Sprintf("dataforseo llm_mentions %s: 1 task @ ~$%.2f/task", endpoint, perMentionsCostUSD),
	})
	if err != nil {
		return nil, output.PrintCodedError(output.ErrEstimateFailed, "failed to estimate cost", err, nil, output.Format(*format))
	}
	approval := cost.EvaluateApproval(estimate, cfg.ApprovalThresholdUSD)

	meta := map[string]any{
		"estimated_cost":    estimate.Amount,
		"currency":          estimate.Currency,
		"requires_approval": approval.RequiresApproval,
		"dry_run":           dryRun,
		"source":            "dataforseo",
		"verbose":           *verbose,
		"endpoint":          endpoint,
		"term":              term,
	}

	if dryRun {
		return nil, output.PrintSuccess(map[string]any{
			"term":   term,
			"status": "dry_run",
		}, meta, output.Format(*format))
	}
	if approval.RequiresApproval {
		meta["reason"] = approval.Reason
		return nil, output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
			fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
	}

	client := llmmentions.NewClient(dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword))
	return &mentionsRunContext{client: client, meta: meta, dryRun: dryRun}, nil
}

func newAEOMentionsSearchCmd(format *string, verbose *bool) *cobra.Command {
	var term string
	var dryRun bool
	var limit int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search LLM mentions of a brand/keyword (DataForSEO LLM Mentions API)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, resp := setupMentionsRun(format, verbose, "search", term, dryRun)
			if ctx == nil {
				return resp
			}
			result, err := ctx.client.Search(llmmentions.Request{Keyword: term, Limit: limit})
			if err != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "llm_mentions search failed", err, ctx.meta, output.Format(*format))
			}
			ctx.meta["fetched_at"] = time.Now().Format(time.RFC3339)
			return output.PrintSuccess(result, ctx.meta, output.Format(*format))
		},
	}
	cmd.Flags().StringVar(&term, "term", "", "Brand term or keyword to search (required)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of rows to return (0 = API default)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")
	return cmd
}

func newAEOMentionsTopPagesCmd(format *string, verbose *bool) *cobra.Command {
	var term string
	var dryRun bool
	var limit int

	cmd := &cobra.Command{
		Use:   "top-pages",
		Short: "Show which pages AI engines cite for a brand/keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, resp := setupMentionsRun(format, verbose, "top_pages", term, dryRun)
			if ctx == nil {
				return resp
			}
			result, err := ctx.client.TopPages(llmmentions.Request{Keyword: term, Limit: limit})
			if err != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "llm_mentions top_pages failed", err, ctx.meta, output.Format(*format))
			}
			ctx.meta["fetched_at"] = time.Now().Format(time.RFC3339)

			// Persist top pages to state (best-effort, keyed by term).
			if state.Exists(".") && result != nil && len(result.Items) > 0 {
				pages := make([]state.TopPageEntry, 0, len(result.Items))
				for _, it := range result.Items {
					pages = append(pages, state.TopPageEntry{
						URL:      it.URL,
						Domain:   it.Domain,
						Mentions: it.MentionsCount,
					})
				}
				if st, loadErr := state.Load("."); loadErr == nil {
					existing := findMentionsData(st, term)
					existing.Term = term
					existing.TopPages = pages
					st.UpsertMentions(existing)
					st.AddHistory("aeo", fmt.Sprintf("mentions top-pages %q: %d pages", term, len(pages)))
					_ = st.Save(".")
				}
			}

			return output.PrintSuccess(result, ctx.meta, output.Format(*format))
		},
	}
	cmd.Flags().StringVar(&term, "term", "", "Brand term or keyword (required)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of rows to return (0 = API default)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")
	return cmd
}

func newAEOMentionsTopDomainsCmd(format *string, verbose *bool) *cobra.Command {
	var term string
	var dryRun bool
	var limit int

	cmd := &cobra.Command{
		Use:   "top-domains",
		Short: "Show which domains AI engines cite for a brand/keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, resp := setupMentionsRun(format, verbose, "top_domains", term, dryRun)
			if ctx == nil {
				return resp
			}
			result, err := ctx.client.TopDomains(llmmentions.Request{Keyword: term, Limit: limit})
			if err != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "llm_mentions top_domains failed", err, ctx.meta, output.Format(*format))
			}
			ctx.meta["fetched_at"] = time.Now().Format(time.RFC3339)

			// Persist domain share to state (best-effort, keyed by term).
			if state.Exists(".") && result != nil && len(result.Items) > 0 {
				var total int64
				for _, it := range result.Items {
					total += int64(it.MentionsCount)
				}
				share := make(map[string]float64, len(result.Items))
				for _, it := range result.Items {
					if total > 0 {
						share[it.Domain] = float64(it.MentionsCount) / float64(total)
					}
				}
				if st, loadErr := state.Load("."); loadErr == nil {
					existing := findMentionsData(st, term)
					existing.Term = term
					existing.DomainShare = share
					st.UpsertMentions(existing)
					st.AddHistory("aeo", fmt.Sprintf("mentions top-domains %q: %d domains", term, len(share)))
					_ = st.Save(".")
				}
			}

			return output.PrintSuccess(result, ctx.meta, output.Format(*format))
		},
	}
	cmd.Flags().StringVar(&term, "term", "", "Brand term or keyword (required)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of rows to return (0 = API default)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")
	return cmd
}

// newAEOMentionsScanCmd implements Layer A: scan every AI response already in
// state for brand terms. Free, offline, no DataForSEO calls.
func newAEOMentionsScanCmd(format *string, verbose *bool) *cobra.Command {
	var extraTerms []string

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan stored AEO responses for brand mentions (local, free)",
		Long: `Grep every AI response currently saved in .sageo/state.json (populated by
'sageo aeo responses') for the project's brand terms. Terms are drawn from:

  - the site's bare domain (from state.site)
  - state.brand_terms (set via 'sageo init --brand ...')
  - --term flags on this command (repeatable, merged)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := state.Load(".")
			if err != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "failed to load state",
					err, nil, output.Format(*format))
			}
			if st.AEO == nil || len(st.AEO.Responses) == 0 {
				return output.PrintCodedError(output.ErrAEOFailed, "no AEO responses in state",
					fmt.Errorf("run 'sageo aeo responses --prompt ... --all' first"), nil, output.Format(*format))
			}

			terms := collectBrandTerms(st, extraTerms)
			if len(terms) == 0 {
				return output.PrintCodedError(output.ErrAEOFailed, "no brand terms to scan",
					fmt.Errorf("set via 'sageo init --brand ...' or pass --term"), nil, output.Format(*format))
			}

			matches := mentions.DetectInResponses(st.AEO.Responses, terms)

			// Persist per-term under MentionsData.LocalMatches.
			byTerm := map[string][]state.LocalMentionMatch{}
			for _, m := range mentions.ToStateMatches(matches) {
				byTerm[m.Term] = append(byTerm[m.Term], m)
			}
			for term, list := range byTerm {
				existing := findMentionsData(st, term)
				existing.Term = term
				existing.LocalMatches = list
				st.UpsertMentions(existing)
			}
			st.AddHistory("aeo", fmt.Sprintf("mentions scan: %d matches across %d terms", len(matches), len(byTerm)))
			_ = st.Save(".")

			meta := map[string]any{
				"source":      "local",
				"terms":       terms,
				"responses":   countResponses(st.AEO.Responses),
				"match_count": len(matches),
				"verbose":     *verbose,
				"fetched_at":  time.Now().Format(time.RFC3339),
			}
			return output.PrintSuccess(map[string]any{
				"matches": matches,
			}, meta, output.Format(*format))
		},
	}
	cmd.Flags().StringSliceVar(&extraTerms, "term", nil, "Additional brand term(s) to scan (repeatable)")
	return cmd
}

// findMentionsData returns the existing MentionsData for term, or an empty
// value ready to be populated and upserted.
func findMentionsData(st *state.State, term string) state.MentionsData {
	for _, m := range st.Mentions {
		if m.Term == term {
			return m
		}
	}
	return state.MentionsData{Term: term}
}

// collectBrandTerms merges the project's site domain, configured BrandTerms,
// and any --term flags into a deduplicated slice.
func collectBrandTerms(st *state.State, extra []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		key := strings.ToLower(t)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, t)
	}
	if host := bareDomain(st.Site); host != "" {
		add(host)
	}
	for _, t := range st.BrandTerms {
		add(t)
	}
	for _, t := range extra {
		add(t)
	}
	return out
}

// bareDomain extracts the hostname (without port or www. prefix) from a URL
// string. Returns empty if the input is not parseable as a URL.
func bareDomain(siteURL string) string {
	if siteURL == "" {
		return ""
	}
	s := siteURL
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	return strings.TrimPrefix(host, "www.")
}

// countResponses totals the engine rows across every prompt entry.
func countResponses(prompts []state.AEOPromptResult) int {
	n := 0
	for _, p := range prompts {
		n += len(p.Results)
	}
	return n
}
