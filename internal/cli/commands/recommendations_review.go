package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// reviewAction is the outcome of a single review prompt.
type reviewAction struct {
	// Kind is one of: approve, edit, reject, skip, quit.
	Kind   string
	Edited string // populated when Kind == "edit"
	Reason string // populated when Kind == "reject"
}

// reviewPrompter abstracts the terminal UI so the dispatch logic can be
// unit-tested without driving a real TUI.
type reviewPrompter interface {
	// Ask shows rec to the user and returns their chosen action.
	Ask(rec recommendations.Recommendation, index, total int) (reviewAction, error)
}

// newRecommendationsReviewCmd returns the `review` subcommand.
func newRecommendationsReviewCmd(format *string) *cobra.Command {
	var (
		urlFilter        string
		typeFilter       string
		autoApproveUnder int
		reviewFormat     string
		reviewer         string
	)

	c := &cobra.Command{
		Use:   "review",
		Short: "Review LLM-drafted recommendations before they ship to reports",
		Long: `Every LLM-drafted value starts out as pending_review. This command walks
the review queue so a human can approve, edit, or reject each draft before it
appears in a client-facing report.

By default the command runs an interactive TUI. Pass --format json to emit the
queue as JSON (for agent-driven review pipelines).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmtStr := reviewFormat
			if fmtStr == "" {
				fmtStr = *format
			}
			if fmtStr == "" {
				fmtStr = "interactive"
			}

			if !state.Exists(".") {
				return output.PrintCodedErrorWithHint(output.ErrNoProject,
					"No project initialized in this directory",
					"sageo init --url <site>",
					nil, nil, output.Format(*format))
			}
			st, err := state.Load(".")
			if err != nil {
				return output.PrintCodedError("STATE_LOAD_FAILED", "failed to load state", err, nil, output.Format(*format))
			}

			queue := buildReviewQueue(st, reviewFilter{
				URL:              urlFilter,
				ChangeType:       typeFilter,
				AutoApproveUnder: autoApproveUnder,
			})

			// Bulk auto-approve under threshold first. We mutate state and
			// persist before any interactive loop so partial progress is
			// always saved.
			if autoApproveUnder > 0 {
				bulk := bulkApproveUnder(st, autoApproveUnder, reviewer)
				if bulk > 0 {
					if err := st.Save("."); err != nil {
						return output.PrintCodedError("STATE_SAVE_FAILED", "failed to save state", err, nil, output.Format(*format))
					}
					// Rebuild queue so approved items drop out.
					queue = buildReviewQueue(st, reviewFilter{
						URL:              urlFilter,
						ChangeType:       typeFilter,
						AutoApproveUnder: autoApproveUnder,
					})
				}
			}

			if fmtStr == "json" || fmtStr == string(output.FormatJSON) {
				data := map[string]any{
					"queue_size": len(queue),
					"queue":      queue,
				}
				meta := map[string]any{
					"review_status_counts": ReviewStatusCounts(st),
					"fetched_at":           time.Now().UTC().Format(time.RFC3339),
				}
				return output.PrintSuccess(data, meta, output.FormatJSON)
			}

			// Interactive path.
			if len(queue) == 0 {
				fmt.Println("Nothing to review — the queue is empty.")
				return nil
			}
			prompter := &huhReviewPrompter{}
			processed, err := runReviewLoop(st, queue, prompter, reviewer, ".")
			// Persist one more time to be safe (loop persists per-decision too).
			_ = st.Save(".")
			if err != nil && !errors.Is(err, errReviewQuit) {
				return output.PrintCodedError("REVIEW_FAILED", "review failed", err, map[string]any{
					"processed": processed,
				}, output.Format(*format))
			}
			fmt.Printf("Reviewed %d of %d. Counts: %v\n", processed, len(queue), ReviewStatusCounts(st))
			printNextSteps(cmd.ErrOrStderr(), []string{
				"sageo report html --open",
			})
			return nil
		},
	}

	c.Flags().StringVar(&urlFilter, "url", "", "Only review recommendations for this URL")
	c.Flags().StringVar(&typeFilter, "type", "", "Only review recommendations of this change type")
	c.Flags().IntVar(&autoApproveUnder, "auto-approve-under-priority", 0, "Bulk-approve anything with priority strictly below N (0 = off)")
	c.Flags().StringVar(&reviewFormat, "format", "", "Output format: interactive|json (defaults to interactive)")
	c.Flags().StringVar(&reviewer, "reviewer", "cli", "Identifier recorded on each decision")
	return c
}

// reviewFilter captures the CLI filters the review queue honours.
type reviewFilter struct {
	URL              string
	ChangeType       string
	AutoApproveUnder int
}

// buildReviewQueue returns the pending-review recommendations that match the
// filters, sorted by priority desc.
func buildReviewQueue(st *state.State, f reviewFilter) []recommendations.Recommendation {
	if st == nil {
		return nil
	}
	var out []recommendations.Recommendation
	for _, r := range st.Recommendations {
		if r.EffectiveReviewStatus() != state.ReviewPending {
			continue
		}
		if f.URL != "" && r.TargetURL != f.URL {
			continue
		}
		if f.ChangeType != "" && string(r.ChangeType) != f.ChangeType {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Priority > out[j].Priority })
	return out
}

// bulkApproveUnder marks every pending recommendation with priority strictly
// below threshold as approved. Returns the number approved.
func bulkApproveUnder(st *state.State, threshold int, reviewer string) int {
	if st == nil || threshold <= 0 {
		return 0
	}
	now := time.Now().UTC()
	n := 0
	for i := range st.Recommendations {
		r := &st.Recommendations[i]
		if r.EffectiveReviewStatus() != state.ReviewPending {
			continue
		}
		if r.Priority >= threshold {
			continue
		}
		r.ReviewStatus = state.ReviewApproved
		t := now
		r.ReviewedAt = &t
		r.ReviewedBy = reviewer
		r.ReviewNotes = fmt.Sprintf("bulk auto-approved (priority %d < %d)", r.Priority, threshold)
		n++
	}
	return n
}

// ReviewStatusCounts tallies the review statuses across all recommendations.
// Exposed for `sageo analyze` metadata.
func ReviewStatusCounts(st *state.State) map[string]int {
	out := map[string]int{
		"pending":  0,
		"approved": 0,
		"edited":   0,
		"rejected": 0,
	}
	if st == nil {
		return out
	}
	for _, r := range st.Recommendations {
		switch r.EffectiveReviewStatus() {
		case state.ReviewPending:
			out["pending"]++
		case state.ReviewApproved:
			out["approved"]++
		case state.ReviewEdited:
			out["edited"]++
		case state.ReviewRejected:
			out["rejected"]++
		}
	}
	return out
}

// errReviewQuit signals the user pressed "Quit" in the TUI.
var errReviewQuit = errors.New("review: user quit")

// runReviewLoop walks queue, asks the prompter what to do, applies the
// decision to state, and persists after each decision. Returns the number
// of items processed (including skipped).
func runReviewLoop(st *state.State, queue []recommendations.Recommendation, p reviewPrompter, reviewer, workDir string) (int, error) {
	processed := 0
	for i, rec := range queue {
		act, err := p.Ask(rec, i+1, len(queue))
		if err != nil {
			return processed, err
		}
		switch act.Kind {
		case "approve":
			applyReviewDecision(st, rec.ID, state.ReviewApproved, rec.RecommendedValue, "", reviewer)
		case "edit":
			applyReviewDecision(st, rec.ID, state.ReviewEdited, act.Edited, "edited via review", reviewer)
		case "reject":
			applyReviewDecision(st, rec.ID, state.ReviewRejected, rec.RecommendedValue, act.Reason, reviewer)
		case "skip":
			// no-op
		case "quit":
			return processed, errReviewQuit
		default:
			return processed, fmt.Errorf("review: unknown action %q", act.Kind)
		}
		processed++
		// Persist per decision so ctrl-C never loses work.
		if err := st.Save(workDir); err != nil {
			return processed, fmt.Errorf("review: save state: %w", err)
		}
	}
	return processed, nil
}

// applyReviewDecision mutates the matching recommendation on st in place.
// It preserves OriginalDraft so the LLM's original output remains auditable
// even after edits.
func applyReviewDecision(st *state.State, id string, status state.ReviewStatus, newValue, notes, reviewer string) {
	now := time.Now().UTC()
	for i := range st.Recommendations {
		r := &st.Recommendations[i]
		if r.ID != id {
			continue
		}
		// Preserve the LLM's original draft the first time we touch the rec.
		if r.OriginalDraft == "" {
			r.OriginalDraft = r.RecommendedValue
		}
		if status == state.ReviewEdited {
			r.RecommendedValue = newValue
		}
		r.ReviewStatus = status
		t := now
		r.ReviewedAt = &t
		r.ReviewedBy = reviewer
		if notes != "" {
			r.ReviewNotes = notes
		}
		return
	}
}

// -----------------------------------------------------------------------------
// huh-backed prompter
// -----------------------------------------------------------------------------

type huhReviewPrompter struct{}

func (h *huhReviewPrompter) Ask(rec recommendations.Recommendation, index, total int) (reviewAction, error) {
	header := fmt.Sprintf("[%d/%d]  P%d  %s\n%s\n", index, total, rec.Priority, rec.ChangeType, rec.TargetURL)
	if rec.TargetQuery != "" {
		header += "query: " + rec.TargetQuery + "\n"
	}
	header += "\nCurrent:\n" + dashIfEmpty(rec.CurrentValue) + "\n\nProposed:\n" + dashIfEmpty(rec.RecommendedValue)

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Review draft").Description(header),
			huh.NewSelect[string]().
				Title("Decision").
				Options(
					huh.NewOption("Approve — ship as-is", "approve"),
					huh.NewOption("Edit    — open $EDITOR", "edit"),
					huh.NewOption("Reject  — exclude from report", "reject"),
					huh.NewOption("Skip    — decide later", "skip"),
					huh.NewOption("Quit    — stop reviewing", "quit"),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return reviewAction{Kind: "quit"}, nil
		}
		return reviewAction{}, err
	}
	switch choice {
	case "approve", "skip", "quit":
		return reviewAction{Kind: choice}, nil
	case "edit":
		edited, err := editInEditor(rec.RecommendedValue)
		if err != nil {
			return reviewAction{}, err
		}
		return reviewAction{Kind: "edit", Edited: edited}, nil
	case "reject":
		var reason string
		r := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Rejection reason (optional)").Value(&reason),
		))
		if err := r.Run(); err != nil && !errors.Is(err, huh.ErrUserAborted) {
			return reviewAction{}, err
		}
		return reviewAction{Kind: "reject", Reason: reason}, nil
	}
	return reviewAction{Kind: "skip"}, nil
}

func dashIfEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// editInEditor writes initial to a temp file, opens $EDITOR (fallback vi),
// and returns the saved contents.
func editInEditor(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	f, err := os.CreateTemp("", "sageo-review-*.txt")
	if err != nil {
		return "", fmt.Errorf("editor: create temp: %w", err)
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	if _, err := f.WriteString(initial); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("editor: write: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("editor: close: %w", err)
	}
	cmd := exec.Command(editor, path) // #nosec G204 -- editor comes from the user's env.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor (%s): %w", filepath.Base(editor), err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("editor: read back: %w", err)
	}
	return strings.TrimRight(string(b), "\n"), nil
}
