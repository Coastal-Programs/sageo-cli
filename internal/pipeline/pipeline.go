// Package pipeline orchestrates the end-to-end `sageo run` workflow.
//
// A pipeline is a linear sequence of named stages. Each stage may be free or
// paid; paid stages contribute to a running budget that can cap the whole
// invocation. Stages persist their results to state.json as they go, so a
// failure part-way through can be resumed with `--resume`.
package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/report/html"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// Stage is one step of the pipeline. Each stage owns its own state mutations;
// the orchestrator only handles ordering, budget enforcement, persistence and
// observability around it.
type Stage struct {
	// Name uniquely identifies the stage (e.g. "crawl", "psi", "aeo").
	Name string
	// Run executes the stage. It may mutate *state.State; the orchestrator
	// saves state after a successful run.
	Run func(ctx context.Context, s *state.State) error
	// EstimateUSD returns a best-effort estimate of the stage's API cost.
	// Free stages should return 0 (or leave EstimateUSD nil).
	EstimateUSD func(*state.State) float64
	// Paid stages are gated by the budget ceiling and the approval flow.
	Paid bool
}

// Config tunes a pipeline invocation.
type Config struct {
	// WorkDir is the project directory containing .sageo/state.json.
	WorkDir string
	// BudgetUSD, when > 0, is the hard ceiling for total paid spend across
	// the run. Stages whose estimates would push accumulated spend over the
	// ceiling are skipped with a budget-exceeded outcome.
	BudgetUSD float64
	// Skip contains stage names to omit.
	Skip map[string]bool
	// Only, if non-empty, restricts execution to the listed stage names.
	Only map[string]bool
	// Approve pre-approves all cost gates (no interactive prompt).
	Approve bool
	// Resume starts after the last successfully completed stage recorded
	// on state.PipelineCursor.
	Resume bool
	// DryRun prints the plan (stages + estimates) and returns without
	// executing anything.
	DryRun bool
	// Verbose enables a progress line per stage with elapsed time.
	Verbose bool
	// Out is where verbose progress lines go. Defaults to os.Stderr.
	Out io.Writer
	// NoSnapshot disables per-run snapshot creation.
	NoSnapshot bool
	// SnapshotKeepLastN is the retention count for auto-prune after snapshot.
	// 0 = don't prune by count.
	SnapshotKeepLastN int
	// SnapshotKeepWithin is the retention age for auto-prune after snapshot.
	// 0 = don't prune by age.
	SnapshotKeepWithin time.Duration
}

// Result captures what the pipeline did.
type Result struct {
	StagesRun    []string           `json:"stages_run"`
	Skipped      []string           `json:"skipped,omitempty"`
	StageCosts   map[string]float64 `json:"stage_costs,omitempty"`
	TotalCostUSD float64            `json:"total_cost_usd"`
	StartedAt    time.Time          `json:"started_at"`
	CompletedAt  time.Time          `json:"completed_at"`
	Outcome      string             `json:"outcome"` // success | partial | failed | dry_run | budget_exceeded
	FailedStage  string             `json:"failed_stage,omitempty"`
	Error        string             `json:"error,omitempty"`
}

// Run executes the given stages in order, honouring budget/skip/only/resume.
//
// Each stage is called in sequence. On success, state.PipelineCursor is
// updated and state is persisted. On error, state is still persisted
// (progress so far) and the error is propagated to the caller.
func Run(ctx context.Context, cfg Config, stages []Stage) (*Result, error) {
	if cfg.Out == nil {
		cfg.Out = os.Stderr
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}

	startedAt := time.Now().UTC()
	res := &Result{
		StageCosts: map[string]float64{},
		StartedAt:  startedAt,
		Outcome:    "success",
	}

	// Determine which stages are in scope (skip/only + resume).
	var active []Stage
	var resumeHit bool
	for _, stg := range stages {
		if len(cfg.Only) > 0 && !cfg.Only[stg.Name] {
			continue
		}
		if cfg.Skip[stg.Name] {
			continue
		}
		active = append(active, stg)
	}

	if cfg.Resume {
		st, err := state.Load(cfg.WorkDir)
		if err == nil && st.PipelineCursor != "" {
			trimmed := make([]Stage, 0, len(active))
			for _, stg := range active {
				if !resumeHit {
					if stg.Name == st.PipelineCursor {
						resumeHit = true
					}
					continue
				}
				trimmed = append(trimmed, stg)
			}
			if resumeHit {
				active = trimmed
			}
		}
	}

	if cfg.DryRun {
		total := 0.0
		st, _ := state.Load(cfg.WorkDir)
		for _, stg := range active {
			cost := 0.0
			if stg.EstimateUSD != nil && st != nil {
				cost = stg.EstimateUSD(st)
			}
			total += cost
			res.StagesRun = append(res.StagesRun, stg.Name)
			res.StageCosts[stg.Name] = cost
			_, _ = fmt.Fprintf(cfg.Out, "[dry-run] stage=%s paid=%v estimate=$%.4f\n", stg.Name, stg.Paid, cost)
		}
		res.TotalCostUSD = total
		res.CompletedAt = time.Now().UTC()
		res.Outcome = "dry_run"
		return res, nil
	}

	// Execute.
	for _, stg := range active {
		if err := ctx.Err(); err != nil {
			res.Outcome = "failed"
			res.Error = err.Error()
			res.CompletedAt = time.Now().UTC()
			return res, err
		}

		// Load fresh state for each stage so stages see each other's writes.
		st, err := state.Load(cfg.WorkDir)
		if err != nil {
			res.Outcome = "failed"
			res.FailedStage = stg.Name
			res.Error = err.Error()
			res.CompletedAt = time.Now().UTC()
			return res, fmt.Errorf("stage %s: load state: %w", stg.Name, err)
		}

		// Budget check for paid stages.
		cost := 0.0
		if stg.EstimateUSD != nil {
			cost = stg.EstimateUSD(st)
		}
		if stg.Paid && cfg.BudgetUSD > 0 {
			projected := res.TotalCostUSD + cost
			if projected > cfg.BudgetUSD {
				_, _ = fmt.Fprintf(cfg.Out, "[pipeline] skipping stage=%s — would exceed budget ($%.4f + $%.4f > $%.4f)\n",
					stg.Name, res.TotalCostUSD, cost, cfg.BudgetUSD)
				res.Skipped = append(res.Skipped, stg.Name)
				res.Outcome = "budget_exceeded"
				continue
			}
		}

		start := time.Now()
		if cfg.Verbose {
			_, _ = fmt.Fprintf(cfg.Out, "[pipeline] ▶ stage=%s paid=%v estimate=$%.4f\n", stg.Name, stg.Paid, cost)
		}

		runErr := stg.Run(ctx, st)

		// Persist state even on error so caller can resume.
		st.PipelineCursor = stg.Name
		_ = st.Save(cfg.WorkDir)

		elapsed := time.Since(start)
		if runErr != nil {
			_, _ = fmt.Fprintf(cfg.Out, "[pipeline] ✗ stage=%s failed after %s: %v\n", stg.Name, elapsed.Round(time.Millisecond), runErr)
			res.FailedStage = stg.Name
			res.Error = runErr.Error()
			res.Outcome = "failed"
			res.CompletedAt = time.Now().UTC()
			writeSnapshot(cfg, res, runErr)
			return res, fmt.Errorf("stage %s failed: %w (resume with --resume)", stg.Name, runErr)
		}

		res.StagesRun = append(res.StagesRun, stg.Name)
		res.StageCosts[stg.Name] = cost
		res.TotalCostUSD += cost

		if cfg.Verbose {
			_, _ = fmt.Fprintf(cfg.Out, "[pipeline] ✓ stage=%s elapsed=%s cost=$%.4f\n",
				stg.Name, elapsed.Round(time.Millisecond), cost)
		}
	}

	res.CompletedAt = time.Now().UTC()

	writeSnapshot(cfg, res, nil)

	// Record to PipelineRuns audit log.
	if st, err := state.Load(cfg.WorkDir); err == nil {
		st.PipelineRuns = append(st.PipelineRuns, state.PipelineRun{
			StartedAt:    res.StartedAt,
			CompletedAt:  res.CompletedAt,
			StagesRun:    append([]string(nil), res.StagesRun...),
			TotalCostUSD: res.TotalCostUSD,
			Outcome:      res.Outcome,
			FailedStage:  res.FailedStage,
			Error:        res.Error,
		})
		// Keep the log bounded.
		const maxRuns = 50
		if len(st.PipelineRuns) > maxRuns {
			st.PipelineRuns = st.PipelineRuns[len(st.PipelineRuns)-maxRuns:]
		}
		// Clear cursor on full success so next run starts clean.
		if res.Outcome == "success" {
			st.PipelineCursor = ""
		}
		_ = st.Save(cfg.WorkDir)
	}

	return res, nil
}

// writeSnapshot is the end-of-run hook that freezes state.json,
// recommendations.json, report.html, and metadata.json into a new
// timestamped directory. Snapshot failures are logged but do not fail the
// pipeline — a successful run whose snapshot fails is still a successful
// run as far as the user's data is concerned.
func writeSnapshot(cfg Config, res *Result, runErr error) {
	if cfg.NoSnapshot {
		return
	}
	st, err := state.Load(cfg.WorkDir)
	if err != nil {
		_, _ = fmt.Fprintf(cfg.Out, "[snapshot] skipped: cannot load state: %v\n", err)
		return
	}

	meta := state.SnapshotMeta{
		StartedAt:    res.StartedAt,
		CompletedAt:  res.CompletedAt,
		StagesRun:    append([]string(nil), res.StagesRun...),
		TotalCostUSD: res.TotalCostUSD,
		Outcome:      res.Outcome,
		FailedStage:  res.FailedStage,
	}
	if runErr != nil && meta.Error == "" {
		meta.Error = runErr.Error()
	}

	// Render the HTML report. Best-effort — a render failure should not
	// block snapshot creation.
	var reportHTML []byte
	var buf bytesBuffer
	if err := html.Render(st, &buf, html.Options{}); err == nil {
		reportHTML = buf.Bytes()
	} else {
		_, _ = fmt.Fprintf(cfg.Out, "[snapshot] report render failed: %v\n", err)
	}

	snap, err := state.CreateSnapshot(cfg.WorkDir, st, meta, reportHTML)
	if err != nil {
		_, _ = fmt.Fprintf(cfg.Out, "[snapshot] create failed: %v\n", err)
		return
	}
	if cfg.Verbose {
		_, _ = fmt.Fprintf(cfg.Out, "[snapshot] wrote %s\n", snap.Dir)
	}

	// Auto-prune after successful creation.
	if cfg.SnapshotKeepLastN > 0 || cfg.SnapshotKeepWithin > 0 {
		removed, err := state.PruneSnapshots(cfg.WorkDir, cfg.SnapshotKeepLastN, cfg.SnapshotKeepWithin)
		if err != nil {
			_, _ = fmt.Fprintf(cfg.Out, "[snapshot] prune failed: %v\n", err)
		} else if len(removed) > 0 && cfg.Verbose {
			_, _ = fmt.Fprintf(cfg.Out, "[snapshot] pruned %d old snapshot(s)\n", len(removed))
		}
	}
}

// bytesBuffer is a tiny shim around bytes.Buffer so we avoid another import
// at the top of this file purely for snapshot rendering.
type bytesBuffer struct {
	b []byte
}

func (w *bytesBuffer) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

func (w *bytesBuffer) Bytes() []byte { return w.b }
