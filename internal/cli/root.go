package cli

import (
	"os"

	"github.com/jakeschepis/sageo-cli/internal/cli/commands"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	outputFormat string
	verbose      bool
)

// Execute runs the root command.
func Execute(version string) error {
	root := newRootCmd(version)
	if err := root.Execute(); err != nil {
		output.PrintError(err.Error(), nil)
		return err
	}
	return nil
}

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sageo",
		Short: "Sageo CLI for SEO crawling, auditing, and reporting",
		Long: `sageo is a command-line tool for SEO, GEO, and AEO operations.

Crawl websites, run SEO audits, generate reports, and manage providers.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format: json, text, table")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	cmd.SetErr(os.Stderr)
	cmd.SetOut(os.Stdout)

	cmd.AddCommand(commands.NewVersionCmd(version, &outputFormat, &verbose))
	cmd.AddCommand(commands.NewConfigCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewCrawlCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewAuditCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewReportCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewProviderCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewAuthCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewGSCCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewSERPCmd(&outputFormat, &verbose))
	cmd.AddCommand(commands.NewOpportunitiesCmd(&outputFormat, &verbose))

	return cmd
}
