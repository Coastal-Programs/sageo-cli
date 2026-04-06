package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
	"github.com/jakeschepis/sageo-cli/internal/serp/serpapi"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type loginAction string

const (
	loginActionGSC        loginAction = "gsc"
	loginActionDataForSEO loginAction = "dataforseo"
	loginActionSerpAPI    loginAction = "serpapi"
	loginActionAll        loginAction = "all"
	loginActionFinish     loginAction = "finish"
)

var (
	loginHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#1D4ED8")).Bold(true)
	loginSubtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#2563EB"))
	loginSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#0F766E")).Bold(true)
	loginInfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#1E40AF"))
	loginErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#B91C1C")).Bold(true)
)

var errBackToMenu = errors.New("back to menu")

const (
	selectControlsHint = "Controls: ↑/↓ move • Enter select • Esc back"
	inputControlsHint  = "Controls: Enter continue • Esc back"
)

// NewLoginCmd returns the top-level interactive login command.
func NewLoginCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Interactive setup for service credentials",
		Long:  `Interactively configure credentials for Google Search Console, SerpAPI, and other services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("sageo login requires an interactive terminal")
			}
			return runLogin(format, verbose)
		},
	}
}

func runLogin(format *string, verbose *bool) error {
	printLoginHeader()

	for {
		action, err := selectLoginAction()
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println()
				printLoginSummary()
				return nil
			}
			return err
		}

		switch action {
		case loginActionGSC:
			if err := runGSCLoginForm(format, verbose); err != nil {
				if errors.Is(err, errBackToMenu) {
					fmt.Printf("%s\n\n", loginInfoStyle.Render("• Back to service menu"))
					continue
				}
				fmt.Printf("%s\n\n", loginErrorStyle.Render("✗ Google Search Console: "+err.Error()))
			}
		case loginActionDataForSEO:
			if err := runDataForSEOLoginForm(); err != nil {
				if errors.Is(err, errBackToMenu) {
					fmt.Printf("%s\n\n", loginInfoStyle.Render("• Back to service menu"))
					continue
				}
				fmt.Printf("%s\n\n", loginErrorStyle.Render("✗ DataForSEO: "+err.Error()))
			}
		case loginActionSerpAPI:
			if err := runSerpAPILoginForm(); err != nil {
				if errors.Is(err, errBackToMenu) {
					fmt.Printf("%s\n\n", loginInfoStyle.Render("• Back to service menu"))
					continue
				}
				fmt.Printf("%s\n\n", loginErrorStyle.Render("✗ SerpAPI: "+err.Error()))
			}
		case loginActionAll:
			if err := runGSCLoginForm(format, verbose); err != nil {
				if errors.Is(err, errBackToMenu) {
					fmt.Printf("%s\n\n", loginInfoStyle.Render("• Back to service menu"))
					continue
				}
				fmt.Printf("%s\n\n", loginErrorStyle.Render("✗ Google Search Console: "+err.Error()))
			}
			if err := runDataForSEOLoginForm(); err != nil {
				if errors.Is(err, errBackToMenu) {
					fmt.Printf("%s\n\n", loginInfoStyle.Render("• Back to service menu"))
					continue
				}
				fmt.Printf("%s\n\n", loginErrorStyle.Render("✗ DataForSEO: "+err.Error()))
			}
			if err := runSerpAPILoginForm(); err != nil {
				if errors.Is(err, errBackToMenu) {
					fmt.Printf("%s\n\n", loginInfoStyle.Render("• Back to service menu"))
					continue
				}
				fmt.Printf("%s\n\n", loginErrorStyle.Render("✗ SerpAPI: "+err.Error()))
			}
		case loginActionFinish:
			fmt.Println()
			printLoginSummary()
			return nil
		}
	}
}

func printLoginHeader() {
	fmt.Println()
	fmt.Println(loginHeaderStyle.Render("Sageo CLI by Coastal Programs"))
	fmt.Println(loginSubtleStyle.Render("Credential setup"))
	fmt.Println()
}

func selectLoginAction() (loginAction, error) {
	choice := loginActionFinish

	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	gscLabel := "Google Search Console (OAuth)"
	if isGSCConfigured(cfg) {
		gscLabel += " (configured)"
	}

	dataForSEOLabel := "DataForSEO (SERP + AEO/GEO)"
	if isDataForSEOConfigured(cfg) {
		dataForSEOLabel += " (configured)"
	}

	serpAPILabel := "SerpAPI (API key)"
	if isSerpAPIConfigured(cfg) {
		serpAPILabel += " (configured)"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[loginAction]().
				Title("Select a setup action").
				Description(selectControlsHint).
				Options(
					huh.NewOption(gscLabel, loginActionGSC),
					huh.NewOption(dataForSEOLabel, loginActionDataForSEO),
					huh.NewOption(serpAPILabel, loginActionSerpAPI),
					huh.NewOption("Set up all services", loginActionAll),
					huh.NewOption("Finish", loginActionFinish),
				).
				Value(&choice),
		),
	).WithTheme(loginTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return loginActionFinish, nil
		}
		return "", err
	}

	return choice, nil
}

func runGSCLoginForm(format *string, verbose *bool) error {
	var clientID string
	var clientSecret string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("GSC Client ID").
				Description(inputControlsHint).
				Value(&clientID).
				Validate(validateRequired("client ID")),
			huh.NewInput().
				Title("GSC Client Secret").
				Description(inputControlsHint).
				EchoMode(huh.EchoModePassword).
				Value(&clientSecret).
				Validate(validateRequired("client secret")),
		),
	).WithTheme(loginTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errBackToMenu
		}
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Set("gsc_client_id", strings.TrimSpace(clientID)); err != nil {
		return fmt.Errorf("failed to set client ID: %w", err)
	}
	if err := cfg.Set("gsc_client_secret", strings.TrimSpace(clientSecret)); err != nil {
		return fmt.Errorf("failed to set client secret: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(loginInfoStyle.Render("• Opening browser for authorization..."))
	if err := loginGSC(format, verbose); err != nil {
		return err
	}

	fmt.Println(loginSuccessStyle.Render("✓ Google Search Console authenticated"))
	fmt.Println()
	return nil
}

func runDataForSEOLoginForm() error {
	var login string
	var password string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("DataForSEO Login (email)").
				Description(inputControlsHint).
				Value(&login).
				Validate(validateRequired("login")),
			huh.NewInput().
				Title("DataForSEO Password").
				Description(inputControlsHint).
				EchoMode(huh.EchoModePassword).
				Value(&password).
				Validate(validateRequired("password")),
		),
	).WithTheme(loginTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errBackToMenu
		}
		return err
	}

	trimmedLogin := strings.TrimSpace(login)
	trimmedPassword := strings.TrimSpace(password)

	fmt.Println(loginInfoStyle.Render("• Verifying DataForSEO credentials..."))

	client := dataforseo.New(trimmedLogin, trimmedPassword)
	if err := client.VerifyCredentials(); err != nil {
		detail := sanitizeVerifyError(err, trimmedLogin, trimmedPassword)
		fmt.Println(loginErrorStyle.Render("✗ DataForSEO credential verification failed: " + detail))
		fmt.Println()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Set("dataforseo_login", trimmedLogin); err != nil {
		return fmt.Errorf("failed to set DataForSEO login: %w", err)
	}
	if err := cfg.Set("dataforseo_password", trimmedPassword); err != nil {
		return fmt.Errorf("failed to set DataForSEO password: %w", err)
	}
	if err := cfg.Set("serp_provider", "dataforseo"); err != nil {
		return fmt.Errorf("failed to set serp provider: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(loginSuccessStyle.Render("✓ DataForSEO configured (verified)"))
	fmt.Println()
	return nil
}

func runSerpAPILoginForm() error {
	var apiKey string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("SerpAPI Key").
				Description(inputControlsHint).
				EchoMode(huh.EchoModePassword).
				Value(&apiKey).
				Validate(validateRequired("API key")),
		),
	).WithTheme(loginTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errBackToMenu
		}
		return err
	}

	trimmedKey := strings.TrimSpace(apiKey)

	fmt.Println(loginInfoStyle.Render("• Verifying SerpAPI key..."))

	adapter := serpapi.New(trimmedKey)
	if err := adapter.VerifyKey(); err != nil {
		detail := sanitizeVerifyError(err, trimmedKey)
		fmt.Println(loginErrorStyle.Render("✗ SerpAPI key verification failed: " + detail))
		fmt.Println()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Set("serp_api_key", trimmedKey); err != nil {
		return fmt.Errorf("failed to set API key: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(loginSuccessStyle.Render("✓ SerpAPI configured (verified)"))
	fmt.Println()
	return nil
}

func printLoginSummary() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(loginErrorStyle.Render("✗ Failed to load config for summary: " + err.Error()))
		return
	}

	fmt.Println(loginSuccessStyle.Render("✓ Setup complete"))
	fmt.Println(loginInfoStyle.Render("• Configured services:"))

	for _, line := range buildLoginSummaryLines(cfg) {
		fmt.Println(line)
	}

	fmt.Println()
}

func buildLoginSummaryLines(cfg *config.Config) []string {
	lines := []string{
		fmt.Sprintf("  • Google Search Console: %s", serviceSummaryStatus(isGSCConfigured(cfg), redactValue(cfg.GSCClientID))),
		fmt.Sprintf("  • DataForSEO: %s", serviceSummaryStatus(isDataForSEOConfigured(cfg), cfg.DataForSEOLogin)),
		fmt.Sprintf("  • SerpAPI: %s", serviceSummaryStatus(isSerpAPIConfigured(cfg), redactValue(cfg.SERPAPIKey))),
	}

	if cfg.SERPProvider != "" {
		lines = append(lines, fmt.Sprintf("  • SERP provider: %s", cfg.SERPProvider))
	}

	return lines
}

func serviceSummaryStatus(configured bool, value string) string {
	if !configured {
		return "not configured"
	}
	if strings.TrimSpace(value) == "" {
		return "configured"
	}
	return "configured (" + strings.TrimSpace(value) + ")"
}

func validateRequired(name string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s cannot be empty", name)
		}
		return nil
	}
}

func isDataForSEOConfigured(cfg *config.Config) bool {
	return strings.TrimSpace(cfg.DataForSEOLogin) != "" && strings.TrimSpace(cfg.DataForSEOPassword) != ""
}

func isSerpAPIConfigured(cfg *config.Config) bool {
	return strings.TrimSpace(cfg.SERPAPIKey) != ""
}

func isGSCConfigured(cfg *config.Config) bool {
	return strings.TrimSpace(cfg.GSCClientID) != "" && strings.TrimSpace(cfg.GSCClientSecret) != ""
}

func loginTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		styles := huh.ThemeCharm(isDark)

		styles.Focused.Title = styles.Focused.Title.Foreground(lipgloss.Color("#1D4ED8")).Bold(true)
		styles.Focused.Description = styles.Focused.Description.Foreground(lipgloss.Color("#2563EB"))
		styles.Focused.SelectSelector = styles.Focused.SelectSelector.Foreground(lipgloss.Color("#2563EB")).Bold(true)
		styles.Focused.Option = styles.Focused.Option.Foreground(lipgloss.Color("#0F172A"))
		styles.Focused.SelectedOption = styles.Focused.SelectedOption.Foreground(lipgloss.Color("#1D4ED8")).Bold(true)
		styles.Focused.TextInput.Cursor = styles.Focused.TextInput.Cursor.Foreground(lipgloss.Color("#2563EB"))
		styles.Focused.NextIndicator = styles.Focused.NextIndicator.Foreground(lipgloss.Color("#1D4ED8"))
		styles.Focused.PrevIndicator = styles.Focused.PrevIndicator.Foreground(lipgloss.Color("#1D4ED8"))
		styles.Focused.FocusedButton = styles.Focused.FocusedButton.Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#1D4ED8")).Bold(true)
		styles.Focused.BlurredButton = styles.Focused.BlurredButton.Foreground(lipgloss.Color("#1E3A8A"))
		styles.Focused.ErrorIndicator = styles.Focused.ErrorIndicator.Foreground(lipgloss.Color("#B91C1C")).Bold(true)
		styles.Focused.ErrorMessage = styles.Focused.ErrorMessage.Foreground(lipgloss.Color("#B91C1C"))

		return styles
	})
}

func redactValue(v string) string {
	if len(v) <= 8 {
		return "****"
	}
	return v[:4] + "****"
}

// sanitizeVerifyError returns a user-safe error detail string with any
// secret-like tokens (long hex/base64 strings, emails, passwords) scrubbed.
func sanitizeVerifyError(err error, secrets ...string) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	// Remove any literal secret values that were passed in.
	for _, s := range secrets {
		s = strings.TrimSpace(s)
		if s != "" {
			msg = strings.ReplaceAll(msg, s, "****")
		}
	}

	return msg
}
