package commands

import (
	"fmt"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/auth"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/gsc"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewGSCCmd returns the gsc command group.
func NewGSCCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gsc",
		Short: "Google Search Console commands",
		Long:  `Query Google Search Console data including sites, pages, keywords, and opportunity signals.`,
	}

	cmd.AddCommand(
		newGSCSitesCmd(format, verbose),
		newGSCQueryCmd(format, verbose),
		newGSCOpportunitiesCmd(format, verbose),
	)

	return cmd
}

func newGSCSitesCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sites",
		Short: "Manage GSC site properties",
	}

	cmd.AddCommand(
		newGSCSitesListCmd(format, verbose),
		newGSCSitesUseCmd(format, verbose),
	)

	return cmd
}

func newGSCSitesListCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List accessible GSC properties",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := gscClient(format)
			if err != nil {
				return err
			}

			sites, err := client.ListSites()
			if err != nil {
				return output.PrintCodedError(output.ErrGSCFailed, "failed to list sites", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(sites, map[string]any{
				"count":   len(sites),
				"source":  "gsc",
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}
}

func newGSCSitesUseCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "use <site_url>",
		Short: "Set the active GSC property",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
			}

			cfg.GSCProperty = args[0]
			if err := cfg.Save(); err != nil {
				return output.PrintCodedError(output.ErrConfigSaveFailed, "failed to save config", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(map[string]any{
				"gsc_property": args[0],
				"status":       "ok",
			}, map[string]any{
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}
}

func newGSCQueryCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query GSC search analytics data",
	}

	cmd.AddCommand(
		newGSCQueryPagesCmd(format, verbose),
		newGSCQueryKeywordsCmd(format, verbose),
	)

	return cmd
}

func newGSCQueryPagesCmd(format *string, verbose *bool) *cobra.Command {
	var startDate, endDate string
	var rowLimit int

	cmd := &cobra.Command{
		Use:   "pages",
		Short: "Query page-level performance data",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, property, err := gscClientAndProperty(format)
			if err != nil {
				return err
			}

			if startDate == "" {
				startDate = time.Now().AddDate(0, 0, -28).Format("2006-01-02")
			}
			if endDate == "" {
				endDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
			}

			resp, err := client.QueryPages(gsc.QueryRequest{
				SiteURL:   property,
				StartDate: startDate,
				EndDate:   endDate,
				RowLimit:  rowLimit,
			})
			if err != nil {
				return output.PrintCodedError(output.ErrGSCFailed, "failed to query pages", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(resp.Rows, map[string]any{
				"count":      len(resp.Rows),
				"source":     "gsc",
				"start_date": startDate,
				"end_date":   endDate,
				"verbose":    *verbose,
			}, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD, default: 28 days ago)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD, default: yesterday)")
	cmd.Flags().IntVar(&rowLimit, "limit", 100, "Maximum rows to return")

	return cmd
}

func newGSCQueryKeywordsCmd(format *string, verbose *bool) *cobra.Command {
	var startDate, endDate string
	var rowLimit int

	cmd := &cobra.Command{
		Use:   "keywords",
		Short: "Query keyword-level performance data",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, property, err := gscClientAndProperty(format)
			if err != nil {
				return err
			}

			if startDate == "" {
				startDate = time.Now().AddDate(0, 0, -28).Format("2006-01-02")
			}
			if endDate == "" {
				endDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
			}

			resp, err := client.QueryKeywords(gsc.QueryRequest{
				SiteURL:   property,
				StartDate: startDate,
				EndDate:   endDate,
				RowLimit:  rowLimit,
			})
			if err != nil {
				return output.PrintCodedError(output.ErrGSCFailed, "failed to query keywords", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(resp.Rows, map[string]any{
				"count":      len(resp.Rows),
				"source":     "gsc",
				"start_date": startDate,
				"end_date":   endDate,
				"verbose":    *verbose,
			}, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD, default: 28 days ago)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD, default: yesterday)")
	cmd.Flags().IntVar(&rowLimit, "limit", 100, "Maximum rows to return")

	return cmd
}

func newGSCOpportunitiesCmd(format *string, verbose *bool) *cobra.Command {
	var startDate, endDate string
	var rowLimit int

	cmd := &cobra.Command{
		Use:   "opportunities",
		Short: "Find SEO opportunity signals from GSC data",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, property, err := gscClientAndProperty(format)
			if err != nil {
				return err
			}

			if startDate == "" {
				startDate = time.Now().AddDate(0, 0, -28).Format("2006-01-02")
			}
			if endDate == "" {
				endDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
			}

			seeds, err := client.QueryOpportunities(property, startDate, endDate, rowLimit)
			if err != nil {
				return output.PrintCodedError(output.ErrGSCFailed, "failed to query opportunities", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(seeds, map[string]any{
				"count":      len(seeds),
				"source":     "gsc",
				"start_date": startDate,
				"end_date":   endDate,
				"verbose":    *verbose,
			}, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD, default: 28 days ago)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD, default: yesterday)")
	cmd.Flags().IntVar(&rowLimit, "limit", 1000, "Maximum rows to return")

	return cmd
}

// gscClient creates an authenticated GSC client.
func gscClient(format *string) (*gsc.Client, error) {
	store := auth.NewFileTokenStore()

	st, err := store.Status("gsc")
	if err != nil {
		return nil, output.PrintCodedError(output.ErrAuthFailed, "failed to check auth status", err, nil, output.Format(*format))
	}
	if !st.Authenticated {
		return nil, output.PrintCodedError(output.ErrAuthRequired, "not authenticated with GSC",
			fmt.Errorf("run 'sageo auth login gsc' first (token may be missing or expired)"), nil, output.Format(*format))
	}

	token, err := store.Load("gsc")
	if err != nil {
		return nil, output.PrintCodedError(output.ErrAuthRequired, "not authenticated with GSC", fmt.Errorf("run 'sageo auth login gsc' first"), nil, output.Format(*format))
	}

	return gsc.NewClient(token.AccessToken), nil
}

// gscClientAndProperty creates an authenticated GSC client and resolves the active property.
func gscClientAndProperty(format *string) (*gsc.Client, string, error) {
	client, err := gscClient(format)
	if err != nil {
		return nil, "", err
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, "", output.PrintCodedError(output.ErrConfigLoadFailed, "failed to load config", err, nil, output.Format(*format))
	}

	if cfg.GSCProperty == "" {
		return nil, "", output.PrintCodedError(output.ErrGSCFailed, "no GSC property configured",
			fmt.Errorf("run 'sageo gsc sites use <url>' or set gsc_property in config"), nil, output.Format(*format))
	}

	return client, cfg.GSCProperty, nil
}
