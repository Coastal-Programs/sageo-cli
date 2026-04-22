package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeschepis/sageo-cli/internal/compare"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewCompareCmd wires `sageo compare` — the feedback-loop command that
// diffs two snapshots and shows what actually moved. With no flags it
// compares the latest snapshot against the previous one.
func NewCompareCmd(format *string, verbose *bool) *cobra.Command {
	var (
		fromRef    string
		toRef      string
		fmtFlag    string
		outputHTML string
	)
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Diff two snapshots and show what actually changed",
		Long: `Compare two snapshots from .sageo/snapshots/ and emit a structured diff of
GSC ranking movement, PSI shifts, SERP / AEO citation changes, audit
findings, and recommendation outcomes.

With no flags, compares the latest snapshot against the previous one.

Changes surfaced by compare are correlational, not causal. Algorithm
updates, seasonality, and concurrent work are not controlled for.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromRef == "" {
				fromRef = "previous"
			}
			if toRef == "" {
				toRef = "latest"
			}

			fromSnap, err := state.LoadSnapshot(".", fromRef)
			if err != nil {
				return output.PrintCodedError("SNAPSHOT_NOT_FOUND", "failed to load 'from' snapshot", err,
					map[string]any{"ref": fromRef}, output.Format(*format))
			}
			toSnap, err := state.LoadSnapshot(".", toRef)
			if err != nil {
				return output.PrintCodedError("SNAPSHOT_NOT_FOUND", "failed to load 'to' snapshot", err,
					map[string]any{"ref": toRef}, output.Format(*format))
			}
			if fromSnap.Timestamp.Equal(toSnap.Timestamp) {
				return output.PrintCodedError("COMPARE_SAME_SNAPSHOT",
					"from and to resolve to the same snapshot — need two distinct snapshots to compare",
					nil,
					map[string]any{"from": filepath.Base(fromSnap.Dir), "to": filepath.Base(toSnap.Dir)},
					output.Format(*format))
			}
			if fromSnap.Timestamp.After(toSnap.Timestamp) {
				// Swap — user probably passed them in wall-clock order
				// but we want from=earlier, to=later.
				fromSnap, toSnap = toSnap, fromSnap
				fromRef, toRef = toRef, fromRef
			}

			c, err := compare.Compute(fromSnap, toSnap)
			if err != nil {
				return output.PrintCodedError("COMPARE_FAILED", "failed to compute comparison", err, nil, output.Format(*format))
			}
			c.FromRef = filepath.Base(fromSnap.Dir)
			c.ToRef = filepath.Base(toSnap.Dir)

			appended, calErr := compare.AppendCalibration(".", c)
			if calErr != nil {
				// Calibration persistence is non-fatal — surface as a
				// warning in metadata but still return the diff.
				fmt.Fprintf(os.Stderr, "warning: compare: append calibration: %v\n", calErr)
			}

			// Optional HTML sidecar.
			if outputHTML != "" {
				if err := writeCompareHTML(outputHTML, c); err != nil {
					return output.PrintCodedError("COMPARE_HTML_FAILED", "failed to write HTML report", err, nil, output.Format(*format))
				}
			}

			// Effective format: --format overrides the global -o flag
			// because compare is the one command where text output is
			// the canonical default.
			effective := fmtFlag
			if effective == "" {
				effective = *format
			}

			if effective == "text" {
				if err := compare.RenderText(os.Stdout, c); err != nil {
					return err
				}
				if appended > 0 {
					fmt.Fprintf(os.Stderr, "\ncalibration: %d data point(s) appended to .sageo/calibration.json\n", appended)
				}
				return nil
			}

			meta := map[string]any{
				"from":                 filepath.Base(fromSnap.Dir),
				"to":                   filepath.Base(toSnap.Dir),
				"calibration_appended": appended,
			}
			if outputHTML != "" {
				if abs, err := filepath.Abs(outputHTML); err == nil {
					meta["html_path"] = abs
				} else {
					meta["html_path"] = outputHTML
				}
			}
			return output.PrintSuccess(c, meta, output.Format(effective))
		},
	}

	cmd.Flags().StringVar(&fromRef, "from", "", "Earlier snapshot ref (default: previous)")
	cmd.Flags().StringVar(&toRef, "to", "", "Later snapshot ref (default: latest)")
	cmd.Flags().StringVar(&fmtFlag, "format", "text", "Output format: text or json (overrides global -o)")
	cmd.Flags().StringVar(&outputHTML, "output-html", "", "Also write a self-contained HTML comparison report to this path")
	return cmd
}

func writeCompareHTML(path string, c *compare.Comparison) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := compare.RenderHTML(&buf, c); err != nil {
		return err
	}
	return os.WriteFile(abs, buf.Bytes(), 0o644)
}
