package commands

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewSnapshotsCmd wires `sageo snapshots ...` — the management surface for
// the per-run history under .sageo/snapshots/.
func NewSnapshotsCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "Inspect and manage per-run snapshots stored under .sageo/snapshots/",
	}
	cmd.AddCommand(newSnapshotsListCmd(format))
	cmd.AddCommand(newSnapshotsShowCmd(format))
	cmd.AddCommand(newSnapshotsPathCmd(format))
	cmd.AddCommand(newSnapshotsPruneCmd(format))
	return cmd
}

func newSnapshotsListCmd(format *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List snapshots newest-first with stage, cost and outcome summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			snaps, err := state.ListSnapshots(".")
			if err != nil {
				return output.PrintCodedError("SNAPSHOTS_LIST_FAILED", "failed to list snapshots", err, nil, output.Format(*format))
			}
			rows := make([]map[string]any, 0, len(snaps))
			for _, s := range snaps {
				recCount := 0
				if st, err := s.State(); err == nil {
					recCount = len(st.Recommendations)
				}
				rows = append(rows, map[string]any{
					"timestamp":       filepath.Base(s.Dir),
					"dir":             s.Dir,
					"stages_run":      s.Meta.StagesRun,
					"total_cost_usd":  s.Meta.TotalCostUSD,
					"recommendations": recCount,
					"outcome":         s.Meta.Outcome,
					"failed_stage":    s.Meta.FailedStage,
				})
			}
			return output.PrintSuccess(rows, map[string]any{
				"count":      len(rows),
				"fetched_at": time.Now().UTC().Format(time.RFC3339),
			}, output.Format(*format))
		},
	}
}

func newSnapshotsShowCmd(format *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <ref>",
		Short: "Show metadata for a specific snapshot (ref: latest, previous, or timestamp prefix)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := state.LoadSnapshot(".", args[0])
			if err != nil {
				return output.PrintCodedError("SNAPSHOT_NOT_FOUND", "snapshot lookup failed", err, nil, output.Format(*format))
			}
			recCount := 0
			if st, err := snap.State(); err == nil {
				recCount = len(st.Recommendations)
			}
			data := map[string]any{
				"timestamp":       filepath.Base(snap.Dir),
				"dir":             snap.Dir,
				"meta":            snap.Meta,
				"recommendations": recCount,
			}
			return output.PrintSuccess(data, nil, output.Format(*format))
		},
	}
}

func newSnapshotsPathCmd(format *string) *cobra.Command {
	return &cobra.Command{
		Use:   "path <ref>",
		Short: "Print the absolute path to a snapshot directory (pipe-friendly)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := state.LoadSnapshot(".", args[0])
			if err != nil {
				return output.PrintCodedError("SNAPSHOT_NOT_FOUND", "snapshot lookup failed", err, nil, output.Format(*format))
			}
			// Intentionally raw so shell composition (e.g. `open $(...)`) works.
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), snap.Dir)
			return nil
		},
	}
}

func newSnapshotsPruneCmd(format *string) *cobra.Command {
	var (
		keepN   int
		within  time.Duration
		confirm bool
	)
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune old snapshots. Dry-run unless --confirm is passed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			snaps, err := state.ListSnapshots(".")
			if err != nil {
				return output.PrintCodedError("SNAPSHOTS_LIST_FAILED", "failed to list snapshots", err, nil, output.Format(*format))
			}
			// Compute the kill-list without touching disk.
			now := time.Now().UTC()
			keep := map[string]bool{}
			for i, s := range snaps {
				if keepN > 0 && i < keepN {
					keep[s.Dir] = true
				}
				if within > 0 && now.Sub(s.Timestamp) <= within {
					keep[s.Dir] = true
				}
			}
			var toRemove []string
			for _, s := range snaps {
				if !keep[s.Dir] {
					toRemove = append(toRemove, s.Dir)
				}
			}

			meta := map[string]any{
				"dry_run":      !confirm,
				"keep":         keepN,
				"within":       within.String(),
				"would_prune":  len(toRemove),
				"total_before": len(snaps),
			}

			if !confirm {
				return output.PrintSuccess(map[string]any{
					"to_prune": toRemove,
				}, meta, output.Format(*format))
			}

			removed, err := state.PruneSnapshots(".", keepN, within)
			if err != nil {
				return output.PrintCodedError("SNAPSHOTS_PRUNE_FAILED", "prune failed", err, meta, output.Format(*format))
			}
			meta["pruned"] = len(removed)
			return output.PrintSuccess(map[string]any{
				"removed": removed,
			}, meta, output.Format(*format))
		},
	}
	cmd.Flags().IntVar(&keepN, "keep", 20, "Retain the newest N snapshots")
	cmd.Flags().DurationVar(&within, "within", 90*24*time.Hour, "Also retain snapshots within this duration")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Actually delete; without this, prune runs as a dry-run")
	return cmd
}
