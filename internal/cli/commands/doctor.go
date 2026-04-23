package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/auth"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/state"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// checkStatus is the outcome of a single doctor check.
type checkStatus string

const (
	checkPass checkStatus = "pass"
	checkWarn checkStatus = "warn"
	checkFail checkStatus = "fail"
)

// doctorCheck is one row in the doctor report.
type doctorCheck struct {
	Name    string      `json:"name"`
	Status  checkStatus `json:"status"`
	Message string      `json:"message"`
	Fix     string      `json:"fix,omitempty"`
}

// doctorInputs is the bundle of things each check reads. Assembled once at
// the start of the command so individual checks stay pure and testable.
type doctorInputs struct {
	ProjectExists bool
	State         *state.State
	Config        *config.Config
	GSCStatus     auth.Status
	PSIEnvPresent bool
}

// NewDoctorCmd returns the top-level `sageo doctor` command. It runs a
// health checklist across project state, config, and auth, and prints a
// structured report. Exit 1 iff any check FAILs; warnings never fail.
func NewDoctorCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostic checks on the sageo project, config, and auth",
		Long: `doctor runs a health check against the current project's state, the user's
config, and stored OAuth tokens. It reports pass / warn / fail per check
and a one-line fix for each problem. Warnings do not fail the command;
any fail causes a non-zero exit.

Use this first when sageo run produces unexpected output, or when an
agent wants to verify the environment before driving the pipeline.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			in := gatherDoctorInputs()
			checks := runDoctorChecks(in)
			summary := summariseChecks(checks)
			fmtStr := output.Format(*format)

			// When any check fails, emit an error envelope so that
			// envelope.success == false matches the non-zero exit code.
			// Consumers still get the details via metadata.checks /
			// metadata.summary.
			if summary["fail"] > 0 {
				if fmtStr != output.FormatJSON {
					renderDoctorText(cmd.OutOrStdout(), checks, summary)
				}
				meta := map[string]any{
					"checks":  checks,
					"summary": summary,
				}
				return output.PrintCodedErrorWithHint(
					"DOCTOR_CHECKS_FAILED",
					fmt.Sprintf("%d check(s) failed", summary["fail"]),
					firstFailFix(checks),
					nil, meta, fmtStr,
				)
			}

			data := map[string]any{
				"checks":  checks,
				"summary": summary,
			}
			if fmtStr == output.FormatJSON {
				return output.PrintSuccess(data, nil, fmtStr)
			}
			renderDoctorText(cmd.OutOrStdout(), checks, summary)
			return nil
		},
	}
}

func gatherDoctorInputs() doctorInputs {
	in := doctorInputs{}
	in.ProjectExists = state.Exists(".")
	if in.ProjectExists {
		if s, err := state.Load("."); err == nil {
			in.State = s
		}
	}
	if cfg, err := config.Load(); err == nil {
		in.Config = cfg
	}

	store := auth.NewFileTokenStore()
	if st, err := store.Status("gsc"); err == nil {
		in.GSCStatus = st
	}

	in.PSIEnvPresent = os.Getenv("SAGEO_PSI_API_KEY") != ""
	return in
}

// firstFailFix returns the Fix of the first failing check so the error
// envelope can surface a concrete next action via error.hint.
func firstFailFix(checks []doctorCheck) string {
	for _, c := range checks {
		if c.Status == checkFail && c.Fix != "" {
			return c.Fix
		}
	}
	return "Run `sageo doctor` again after addressing the failing checks above."
}

// runDoctorChecks is the pure core. Tests call this directly.
func runDoctorChecks(in doctorInputs) []doctorCheck {
	return []doctorCheck{
		checkProjectInitialised(in),
		checkBrandTerms(in),
		checkGSCAuth(in),
		checkGSCProperty(in),
		checkPSIAPIKey(in),
		checkLLMProvider(in),
		checkDataForSEOCreds(in),
	}
}

func checkProjectInitialised(in doctorInputs) doctorCheck {
	if in.ProjectExists {
		site := ""
		if in.State != nil {
			site = in.State.Site
		}
		return doctorCheck{
			Name:    "project_initialised",
			Status:  checkPass,
			Message: "project state found" + siteSuffix(site),
		}
	}
	return doctorCheck{
		Name:    "project_initialised",
		Status:  checkFail,
		Message: "no .sageo/state.json in current directory",
		Fix:     "sageo init --url https://example.com",
	}
}

func siteSuffix(site string) string {
	if site == "" {
		return ""
	}
	return " (" + site + ")"
}

func checkBrandTerms(in doctorInputs) doctorCheck {
	if in.State == nil {
		return doctorCheck{
			Name:    "brand_terms",
			Status:  checkWarn,
			Message: "no project state; brand terms unknown",
			Fix:     "sageo init --url <site> --brand \"Brand,alias\"",
		}
	}
	if len(in.State.BrandTerms) == 0 {
		return doctorCheck{
			Name:    "brand_terms",
			Status:  checkWarn,
			Message: "no brand terms configured (AEO mention detection will be weak)",
			Fix:     "sageo init --url <site> --brand \"Brand,alias\"",
		}
	}
	return doctorCheck{
		Name:    "brand_terms",
		Status:  checkPass,
		Message: fmt.Sprintf("%d brand term(s) configured", len(in.State.BrandTerms)),
	}
}

func checkGSCAuth(in doctorInputs) doctorCheck {
	if !in.GSCStatus.Authenticated {
		return doctorCheck{
			Name:    "gsc_auth",
			Status:  checkFail,
			Message: "not authenticated with Google Search Console",
			Fix:     "sageo auth login gsc",
		}
	}
	return doctorCheck{
		Name:    "gsc_auth",
		Status:  checkPass,
		Message: "GSC token present",
	}
}

func checkGSCProperty(in doctorInputs) doctorCheck {
	property := ""
	if in.Config != nil {
		property = in.Config.GSCProperty
	}
	if property == "" {
		return doctorCheck{
			Name:    "gsc_property",
			Status:  checkFail,
			Message: "no GSC property selected; sageo run will abort",
			Fix:     "sageo gsc sites list  &&  sageo gsc sites use <property>",
		}
	}
	return doctorCheck{
		Name:    "gsc_property",
		Status:  checkPass,
		Message: "active property: " + property,
	}
}

func checkPSIAPIKey(in doctorInputs) doctorCheck {
	hasKey := in.PSIEnvPresent
	if in.Config != nil && in.Config.PSIAPIKey != "" {
		hasKey = true
	}
	gscOK := in.GSCStatus.Authenticated
	if hasKey || gscOK {
		return doctorCheck{
			Name:    "psi_api_key",
			Status:  checkPass,
			Message: "PSI credentials available",
		}
	}
	return doctorCheck{
		Name:    "psi_api_key",
		Status:  checkWarn,
		Message: "no PSI API key and no GSC token to fall back on",
		Fix:     "sageo config set psi_api_key <key>  (or: sageo auth login gsc)",
	}
}

func checkLLMProvider(in doctorInputs) doctorCheck {
	provider := "anthropic"
	if in.Config != nil && in.Config.LLMProvider != "" {
		provider = strings.ToLower(in.Config.LLMProvider)
	}
	var key, fix string
	switch provider {
	case "openai":
		if in.Config != nil {
			key = in.Config.OpenAIAPIKey
		}
		fix = "sageo config set openai_api_key <key>"
	default:
		if in.Config != nil {
			key = in.Config.AnthropicAPIKey
		}
		fix = "sageo config set anthropic_api_key <key>"
	}
	if key == "" {
		return doctorCheck{
			Name:    "llm_provider",
			Status:  checkWarn,
			Message: fmt.Sprintf("%s API key not set; the draft stage will be skipped", provider),
			Fix:     fix,
		}
	}
	return doctorCheck{
		Name:    "llm_provider",
		Status:  checkPass,
		Message: provider + " API key present",
	}
}

func checkDataForSEOCreds(in doctorInputs) doctorCheck {
	login, pw := "", ""
	if in.Config != nil {
		login = in.Config.DataForSEOLogin
		pw = in.Config.DataForSEOPassword
	}
	if login == "" || pw == "" {
		return doctorCheck{
			Name:    "dataforseo_creds",
			Status:  checkWarn,
			Message: "DataForSEO credentials not set; SERP, Labs, backlinks, AEO stages will skip",
			Fix:     "sageo config set dataforseo_login <email>  &&  sageo config set dataforseo_password <pw>",
		}
	}
	return doctorCheck{
		Name:    "dataforseo_creds",
		Status:  checkPass,
		Message: "DataForSEO credentials present",
	}
}

func summariseChecks(checks []doctorCheck) map[string]int {
	out := map[string]int{"pass": 0, "warn": 0, "fail": 0}
	for _, c := range checks {
		out[string(c.Status)]++
	}
	return out
}

func renderDoctorText(w io.Writer, checks []doctorCheck, summary map[string]int) {
	for _, c := range checks {
		marker := "?"
		switch c.Status {
		case checkPass:
			marker = "✓"
		case checkWarn:
			marker = "!"
		case checkFail:
			marker = "✗"
		}
		_, _ = fmt.Fprintf(w, "%s %-22s %s\n", marker, c.Name, c.Message)
		if c.Fix != "" {
			_, _ = fmt.Fprintf(w, "    fix: %s\n", c.Fix)
		}
	}
	_, _ = fmt.Fprintf(w, "\nSummary: %d pass, %d warn, %d fail\n", summary["pass"], summary["warn"], summary["fail"])
}
