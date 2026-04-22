package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/common/cost"
	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

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
	)

	return cmd
}

func newAEOResponsesCmd(format *string, verbose *bool) *cobra.Command {
	var prompt, model string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "responses",
		Short: "Query an AI engine and see what it says about your brand or topic",
		Long: `Send a prompt to an AI engine (ChatGPT, Claude, Gemini, or Perplexity) and see the full response.
Useful for understanding how AI tools describe your brand, products, or keywords.

Supported models: chatgpt, claude, gemini, perplexity`,
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

			// Estimate: ~$0.003/query (LLM pass-through pricing varies by model)
			estimate, err := cost.BuildEstimate(cost.EstimateInput{
				UnitCostUSD: 0.003,
				Units:       1,
				Basis:       fmt.Sprintf("dataforseo llm responses: 1 query via %s @ ~$0.003/query", model),
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
				"model":             model,
				"source":            "dataforseo",
				"verbose":           *verbose,
			}

			if dryRun {
				return output.PrintSuccess(map[string]any{
					"prompt": prompt,
					"model":  model,
					"status": "dry_run",
				}, meta, output.Format(*format))
			}

			if approval.RequiresApproval {
				meta["reason"] = approval.Reason
				return output.PrintCodedError(output.ErrApprovalRequired, "cost exceeds approval threshold",
					fmt.Errorf("%s", approval.Reason), meta, output.Format(*format))
			}

			// Validate model before making any API calls.
			validModels := map[string]bool{"chatgpt": true, "claude": true, "gemini": true, "perplexity": true}
			if !validModels[model] {
				return output.PrintCodedError(output.ErrAEOFailed,
					fmt.Sprintf("unsupported model %q", model),
					fmt.Errorf("valid values: chatgpt, claude, gemini, perplexity"), nil, output.Format(*format))
			}

			client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)

			endpoint, endpointErr := aeoEndpointForModel(model)
			if endpointErr != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "invalid model", endpointErr, nil, output.Format(*format))
			}
			modelName, modelNameErr := defaultModelNameForEngine(model)
			if modelNameErr != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "invalid model", modelNameErr, nil, output.Format(*format))
			}
			// DataForSEO requires `user_prompt` (not `prompt`) and `model_name`.
			// See: https://docs.dataforseo.com/v3/ai_optimization-chat_gpt-llm_responses-live/
			reqBody := []map[string]any{
				{
					"user_prompt": prompt,
					"model_name":  modelName,
				},
			}

			raw, err := client.Post(endpoint, reqBody)
			if err != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "LLM responses request failed", err, nil, output.Format(*format))
			}

			// Response shape per DataForSEO docs:
			//   tasks[].result[].{ model_name, items[].sections[].{type,text} }
			// Text sections are concatenated to produce the flat `response`
			// string this command returns. Non-text sections (if any) are
			// ignored for now — extend the struct if structured output is
			// needed downstream.
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

			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				return output.PrintCodedError(output.ErrAEOFailed, "failed to decode DataForSEO response", jsonErr, nil, output.Format(*format))
			}

			if envelope.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrAEOFailed,
					fmt.Sprintf("DataForSEO error %d: %s", envelope.StatusCode, envelope.StatusMessage),
					fmt.Errorf("code %d", envelope.StatusCode), nil, output.Format(*format))
			}
			if len(envelope.Tasks) == 0 {
				return output.PrintCodedError(output.ErrAEOFailed, "no tasks returned",
					fmt.Errorf("dataforseo response contained no tasks"), nil, output.Format(*format))
			}
			// Surface per-task failures (e.g. 40503 "POST Data Is Invalid"),
			// which DataForSEO returns with a 20000 top-level envelope.
			taskResult := envelope.Tasks[0]
			if taskResult.StatusCode != 20000 {
				return output.PrintCodedError(output.ErrAEOFailed,
					fmt.Sprintf("DataForSEO task error %d: %s", taskResult.StatusCode, taskResult.StatusMessage),
					fmt.Errorf("task status %d", taskResult.StatusCode), nil, output.Format(*format))
			}
			if len(taskResult.Result) == 0 {
				return output.PrintCodedError(output.ErrAEOFailed, "no results returned",
					fmt.Errorf("dataforseo returned empty task result"), nil, output.Format(*format))
			}

			result := taskResult.Result[0]
			var responseText strings.Builder
			for _, item := range result.Items {
				for _, section := range item.Sections {
					if section.Type == "text" {
						responseText.WriteString(section.Text)
					}
				}
			}
			meta["fetched_at"] = time.Now().Format(time.RFC3339)

			return output.PrintSuccess(map[string]any{
				"prompt":   prompt,
				"response": responseText.String(),
				"model":    result.ModelName,
			}, meta, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt to send to the AI engine (required)")
	cmd.Flags().StringVar(&model, "model", "chatgpt", "AI model: chatgpt, claude, gemini, perplexity")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Estimate cost without executing")

	return cmd
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

// aeoEndpointForModel returns the DataForSEO LLM responses endpoint for the given model name.
func aeoEndpointForModel(model string) (string, error) {
	switch model {
	case "chatgpt":
		return "/v3/ai_optimization/chat_gpt/llm_responses/live", nil
	case "claude":
		return "/v3/ai_optimization/claude/llm_responses/live", nil
	case "gemini":
		return "/v3/ai_optimization/gemini/llm_responses/live", nil
	case "perplexity":
		return "/v3/ai_optimization/perplexity/llm_responses/live", nil
	default:
		return "", fmt.Errorf("unsupported model %q: valid values: chatgpt, claude, gemini, perplexity", model)
	}
}

// defaultModelNameForEngine returns the DataForSEO `model_name` default for
// each friendly engine alias. DataForSEO requires this field on every
// llm_responses/live request; without it the API rejects the task with
// 40501 "Invalid Field: 'model_name'". Defaults are picked to be fast and
// inexpensive baselines — the full model catalog per engine is available at
// /v3/ai_optimization/<engine>/llm_responses/models.
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
