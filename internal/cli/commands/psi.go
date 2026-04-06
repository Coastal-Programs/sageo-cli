package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/psi"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewPSICmd returns the psi command group.
func NewPSICmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "psi",
		Short: "PageSpeed Insights commands",
		Long:  `Fetch Core Web Vitals from the Google PageSpeed Insights API.`,
	}

	cmd.AddCommand(newPSIRunCmd(format, verbose))

	return cmd
}

func newPSIRunCmd(format *string, verbose *bool) *cobra.Command {
	var targetURL, strategy string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a PageSpeed Insights analysis",
		Long:  `Analyse a page with Google PageSpeed Insights and return Core Web Vitals.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetURL == "" {
				return output.PrintCodedError(output.ErrPSIFailed, "url is required",
					fmt.Errorf("use --url to specify the page to analyse"), nil, output.Format(*format))
			}

			// Resolve API key: env var → config → empty (unauthenticated).
			apiKey := os.Getenv("SAGEO_PSI_API_KEY")
			if apiKey == "" {
				cfg, err := config.Load()
				if err == nil {
					apiKey = cfg.PSIAPIKey
				}
			}

			client := psi.NewClient(apiKey, nil)

			result, err := client.Run(targetURL, strategy)
			if err != nil {
				return output.PrintCodedError(output.ErrPSIFailed, "PageSpeed Insights request failed", err, nil, output.Format(*format))
			}

			// Persist to state.json if a project is initialized.
			if state.Exists(".") {
				if st, lerr := state.Load("."); lerr == nil {
					if st.PSI == nil {
						st.PSI = &state.PSIData{}
					}
					psiResult := state.PSIResult{
						URL:              result.URL,
						PerformanceScore: result.PerformanceScore,
						LCP:              result.LCP,
						CLS:              result.CLS,
						Strategy:         result.Strategy,
					}
					upsertPSIResult(st.PSI, psiResult)
					st.PSI.LastRun = time.Now().UTC().Format(time.RFC3339)
					st.AddHistory("psi.run", fmt.Sprintf("url=%s strategy=%s score=%.0f", result.URL, result.Strategy, result.PerformanceScore))
					_ = st.Save(".")
				}
			}

			return output.PrintSuccess(result, nil, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Page URL to analyse")
	cmd.Flags().StringVar(&strategy, "strategy", "mobile", "Analysis strategy: mobile or desktop")

	return cmd
}

// upsertPSIResult adds or replaces the PSI result for the given URL+strategy pair.
func upsertPSIResult(psiData *state.PSIData, r state.PSIResult) {
	for i, p := range psiData.Pages {
		if p.URL == r.URL && p.Strategy == r.Strategy {
			psiData.Pages[i] = r
			return
		}
	}
	psiData.Pages = append(psiData.Pages, r)
}
