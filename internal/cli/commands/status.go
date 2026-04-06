package commands

import (
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewStatusCmd returns the status command.
func NewStatusCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current project state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !state.Exists(".") {
				return output.PrintCodedError(
					"NO_PROJECT",
					"No project initialized — run sageo init --url <site>",
					nil, nil,
					output.Format(*format),
				)
			}

			s, err := state.Load(".")
			if err != nil {
				return output.PrintCodedError("STATE_LOAD_FAILED", "failed to load state", err, nil, output.Format(*format))
			}

			used, missing := s.Sources()

			data := map[string]any{
				"site":            s.Site,
				"initialized":     s.Initialized,
				"last_crawl":      s.LastCrawl,
				"score":           s.Score,
				"pages_crawled":   s.PagesCrawled,
				"findings_count":  len(s.Findings),
				"history_count":   len(s.History),
				"sources_used":    used,
				"sources_missing": missing,
			}
			metadata := map[string]any{
				"generated_at": time.Now().UTC().Format(time.RFC3339),
			}
			return output.PrintSuccess(data, metadata, output.Format(*format))
		},
	}
}
