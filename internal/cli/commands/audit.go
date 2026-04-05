package commands

import (
	"context"
	"errors"

	"github.com/jakeschepis/sageo-cli/internal/audit"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/crawl"
	"github.com/jakeschepis/sageo-cli/internal/provider"
	_ "github.com/jakeschepis/sageo-cli/internal/provider/local"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewAuditCmd returns the audit command group.
func NewAuditCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "SEO audit commands",
	}

	cmd.AddCommand(newAuditRunCmd(format, verbose))
	return cmd
}

func newAuditRunCmd(format *string, verbose *bool) *cobra.Command {
	var (
		targetURL string
		depth     int
		maxPages  int
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Crawl a website and run an SEO audit",
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

			auditSvc := audit.NewService()
			auditResult, err := auditSvc.Run(cmd.Context(), audit.Request{
				CrawlResult: crawlResult,
			})
			if err != nil {
				return output.PrintCodedError(output.ErrAuditFailed, "audit failed", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(auditResult, map[string]any{
				"pages_audited": auditResult.PageCount,
				"total_issues":  len(auditResult.Issues),
				"verbose":       *verbose,
			}, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Target URL to audit (required)")
	cmd.Flags().IntVar(&depth, "depth", 2, "Maximum crawl depth")
	cmd.Flags().IntVar(&maxPages, "max-pages", 50, "Maximum number of pages to crawl")

	return cmd
}
