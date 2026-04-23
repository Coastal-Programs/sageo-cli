package commands

import (
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the init command.
func NewInitCmd(format *string, verbose *bool) *cobra.Command {
	var siteURL, brand string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a .sageo project for a site",
		RunE: func(cmd *cobra.Command, args []string) error {
			if siteURL == "" {
				return output.PrintCodedErrorWithHint(output.ErrInvalidURL, "--url is required",
					"Use a full URL, for example: sageo init --url https://example.com",
					nil, nil, output.Format(*format))
			}

			s, err := state.Init(".", siteURL)
			if err != nil {
				return output.PrintErrorResponse(err.Error(), err, nil, output.Format(*format))
			}

			if brand != "" {
				terms := splitBrandTerms(brand)
				if len(terms) > 0 {
					s.BrandTerms = terms
					if saveErr := s.Save("."); saveErr != nil {
						return output.PrintErrorResponse(saveErr.Error(), saveErr, nil, output.Format(*format))
					}
				}
			}

			printNextSteps(cmd.ErrOrStderr(), []string{
				"sageo auth login gsc",
				"sageo gsc sites use " + s.Site,
				"sageo run " + s.Site + " --budget 10",
			})

			return output.PrintSuccess(map[string]interface{}{
				"site":        s.Site,
				"initialized": s.Initialized,
				"brand_terms": s.BrandTerms,
				"state_file":  state.Path("."),
			}, nil, output.Format(*format))
		},
	}

	cmd.Flags().StringVar(&siteURL, "url", "", "Site URL to track")
	cmd.Flags().StringVar(&brand, "brand", "", "Comma-separated brand names/aliases to track in AI responses (e.g. \"Sageo,sageo.io\")")
	return cmd
}

// splitBrandTerms splits a comma-separated brand flag into trimmed,
// non-empty terms.
func splitBrandTerms(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
