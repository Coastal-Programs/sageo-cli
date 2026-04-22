package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/jakeschepis/sageo-cli/internal/audit"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/crawl"
	"github.com/jakeschepis/sageo-cli/internal/provider"
	_ "github.com/jakeschepis/sageo-cli/internal/provider/local"
	"github.com/jakeschepis/sageo-cli/internal/report"
	htmlreport "github.com/jakeschepis/sageo-cli/internal/report/html"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewReportCmd returns the report command group.
func NewReportCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Report generation and listing commands",
	}

	cmd.AddCommand(
		newReportGenerateCmd(format, verbose),
		newReportListCmd(format, verbose),
		newReportHTMLCmd(format, verbose),
		newReportPDFAliasCmd(format, verbose),
	)
	return cmd
}

// newReportHTMLCmd is the primary report output — a single self-contained
// HTML file. Users who want a PDF use browser print-to-PDF (Cmd/Ctrl+P).
func newReportHTMLCmd(format *string, _ *bool) *cobra.Command {
	var (
		outPath    string
		appendix   bool
		logoPath   string
		brandColor string
		openFile   bool
		title      string
	)

	cmd := &cobra.Command{
		Use:   "html",
		Short: "Render a self-contained HTML report (use browser Cmd/Ctrl+P → Save as PDF for a PDF copy)",
		Long: `Render a styled, self-contained HTML report summarising the audit, recommendations, and forecast.

The output is a single .html file with inlined CSS and minimal JS — no external
resources, works offline. To get a PDF, open the file in any modern browser and
press Cmd+P (macOS) or Ctrl+P (Linux/Windows), then choose "Save as PDF".`,
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

			abs, err := filepath.Abs(outPath)
			if err != nil {
				return output.PrintCodedError(output.ErrReportWriteFailed, "invalid output path", err, nil, output.Format(*format))
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return output.PrintCodedError(output.ErrReportWriteFailed, "failed to create output dir", err, nil, output.Format(*format))
			}

			logoURI, err := htmlreport.LoadLogoAsDataURI(logoPath)
			if err != nil {
				return output.PrintCodedError(output.ErrReportWriteFailed, "failed to load logo", err, nil, output.Format(*format))
			}

			f, err := os.Create(abs)
			if err != nil {
				return output.PrintCodedError(output.ErrReportWriteFailed, "failed to create HTML file", err, nil, output.Format(*format))
			}
			defer func() { _ = f.Close() }()

			size, err := htmlreport.RenderWithStats(s, f, htmlreport.Options{
				IncludeAppendix: appendix,
				BrandColorHex:   brandColor,
				LogoDataURI:     logoURI,
				Title:           title,
			})
			if err != nil {
				return output.PrintCodedError(output.ErrReportWriteFailed, "failed to render HTML", err, nil, output.Format(*format))
			}

			fmt.Fprintf(os.Stderr, "✓ Report written. Open with: open %s  (or Cmd+P → Save as PDF for a printed copy)\n", abs)

			if openFile {
				_ = openInBrowser(abs)
			}

			return output.PrintSuccess(map[string]any{
				"path":       abs,
				"size_bytes": size,
			}, nil, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&outPath, "output", "./sageo-report.html", "Path to write the HTML file to")
	cmd.Flags().BoolVar(&appendix, "appendix", false, "Include raw data appendix tables")
	cmd.Flags().StringVar(&logoPath, "logo", "", "Optional path to a PNG/JPG logo for the cover (embedded as base64)")
	cmd.Flags().StringVar(&brandColor, "brand-color", "", "Brand colour hex (default #1E40AF)")
	cmd.Flags().BoolVar(&openFile, "open", false, "Open the report in the default browser after generation")
	cmd.Flags().StringVar(&title, "title", "", "Report title (default \"Sageo Audit Report\")")
	return cmd
}

// newReportPDFAliasCmd preserves `sageo report pdf` as a deprecated alias that
// routes to the HTML renderer so existing scripts keep working with a warning.
func newReportPDFAliasCmd(format *string, verbose *bool) *cobra.Command {
	htmlCmd := newReportHTMLCmd(format, verbose)
	orig := htmlCmd.RunE
	cmd := &cobra.Command{
		Use:        "pdf",
		Short:      "Deprecated: alias for `sageo report html`. Use Cmd+P → Save as PDF in your browser.",
		Deprecated: "use `sageo report html` instead; open the .html file in a browser and press Cmd/Ctrl+P to save as PDF.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "warning: `sageo report pdf` is deprecated. Rendering HTML instead. Use Cmd/Ctrl+P in your browser to save as PDF.")
			return orig(cmd, args)
		},
	}
	// Copy flags so existing invocations keep working.
	cmd.Flags().AddFlagSet(htmlCmd.Flags())
	// Change default output to sageo-report.html for consistency.
	if f := cmd.Flags().Lookup("output"); f != nil {
		f.DefValue = "./sageo-report.html"
		_ = f.Value.Set("./sageo-report.html")
	}
	return cmd
}

func openInBrowser(path string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", path)
	case "windows":
		c = exec.Command("cmd", "/c", "start", "", path)
	default:
		c = exec.Command("xdg-open", path)
	}
	return c.Start()
}

func newReportGenerateCmd(format *string, verbose *bool) *cobra.Command {
	var (
		targetURL string
		depth     int
		maxPages  int
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Crawl, audit, and generate a stored report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetURL == "" {
				return output.PrintCodedError(output.ErrInvalidURL, "missing required --url flag", nil, nil, output.Format(*format))
			}

			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			fetcher, err := provider.NewFetcher(cfg.ActiveProvider)
			if err != nil {
				return output.PrintCodedError(output.ErrProviderNotFound, "failed to create provider", err, nil, output.Format(*format))
			}

			// Crawl
			crawlSvc := crawl.NewService(fetcher)
			crawlResult, err := crawlSvc.Run(cmd.Context(), crawl.Request{
				TargetURL: targetURL,
				Depth:     depth,
				MaxPages:  maxPages,
			})
			if err != nil {
				code := output.ErrCrawlFailed
				if errors.Is(err, context.DeadlineExceeded) {
					code = output.ErrFetchTimeout
				} else if errors.Is(err, context.Canceled) {
					code = output.ErrCancelled
				}
				return output.PrintCodedError(code, "crawl failed", err, nil, output.Format(*format))
			}

			// Audit
			auditSvc := audit.NewService()
			auditResult, err := auditSvc.Run(cmd.Context(), audit.Request{
				CrawlResult: crawlResult,
			})
			if err != nil {
				return output.PrintCodedError(output.ErrAuditFailed, "audit failed", err, nil, output.Format(*format))
			}

			// Report
			reportSvc := report.NewService()
			reportResult, err := reportSvc.Generate(cmd.Context(), report.Request{
				AuditResult: auditResult,
			})
			if err != nil {
				return output.PrintCodedError(output.ErrReportWriteFailed, "report generation failed", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(reportResult, map[string]any{
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Target URL to report on (required)")
	cmd.Flags().IntVar(&depth, "depth", 2, "Maximum crawl depth")
	cmd.Flags().IntVar(&maxPages, "max-pages", 50, "Maximum number of pages to crawl")

	return cmd
}

func newReportListCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := report.NewService()
			reports, err := svc.List(cmd.Context(), "")
			if err != nil {
				return output.PrintCodedError(output.ErrReportListFailed, "failed to list reports", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(reports, map[string]any{
				"count":   len(reports),
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}
}
