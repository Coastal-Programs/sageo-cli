package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("  Sageo CLI — Login")
	fmt.Println()
	fmt.Println("  Services:")
	fmt.Println("    1. Google Search Console (OAuth)")
	fmt.Println("    2. DataForSEO (SERP + AEO/GEO)")
	fmt.Println("    3. SerpAPI (API key)")
	fmt.Println()

	for {
		fmt.Print("  Select a service (1-3, or 'all', 'done' to finish): ")
		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			if err := loginGSCInteractive(scanner, format, verbose); err != nil {
				fmt.Printf("  ✗ Google Search Console: %v\n\n", err)
			}
		case "2":
			if err := loginDataForSEOInteractive(scanner); err != nil {
				fmt.Printf("  ✗ DataForSEO: %v\n\n", err)
			}
		case "3":
			if err := loginSerpAPIInteractive(scanner); err != nil {
				fmt.Printf("  ✗ SerpAPI: %v\n\n", err)
			}
		case "all":
			if err := loginGSCInteractive(scanner, format, verbose); err != nil {
				fmt.Printf("  ✗ Google Search Console: %v\n\n", err)
			}
			if err := loginDataForSEOInteractive(scanner); err != nil {
				fmt.Printf("  ✗ DataForSEO: %v\n\n", err)
			}
			if err := loginSerpAPIInteractive(scanner); err != nil {
				fmt.Printf("  ✗ SerpAPI: %v\n\n", err)
			}
		case "done", "q", "quit", "exit":
			// fall through to summary
		default:
			fmt.Println("  Invalid choice. Enter 1, 2, 3, 'all', or 'done'.")
			fmt.Println()
			continue
		}

		if choice == "done" || choice == "q" || choice == "quit" || choice == "exit" || choice == "all" {
			break
		}
	}

	fmt.Println()
	fmt.Println("  ✓ Setup complete")
	fmt.Println()
	return nil
}

func loginGSCInteractive(scanner *bufio.Scanner, format *string, verbose *bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println()

	// Prompt for client ID
	fmt.Print("  GSC Client ID: ")
	if !scanner.Scan() {
		return fmt.Errorf("interrupted")
	}
	clientID := strings.TrimSpace(scanner.Text())
	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}
	fmt.Printf("  GSC Client ID: %s\n", redactValue(clientID))

	// Prompt for client secret
	fmt.Print("  GSC Client Secret: ")
	if !scanner.Scan() {
		return fmt.Errorf("interrupted")
	}
	clientSecret := strings.TrimSpace(scanner.Text())
	if clientSecret == "" {
		return fmt.Errorf("client secret cannot be empty")
	}
	fmt.Printf("  GSC Client Secret: %s\n", redactValue(clientSecret))

	// Save credentials to config
	if err := cfg.Set("gsc_client_id", clientID); err != nil {
		return fmt.Errorf("failed to set client ID: %w", err)
	}
	if err := cfg.Set("gsc_client_secret", clientSecret); err != nil {
		return fmt.Errorf("failed to set client secret: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Run the existing OAuth flow
	fmt.Println("  Opening browser for authorization...")
	if err := loginGSC(format, verbose); err != nil {
		return err
	}

	fmt.Println("  ✓ Google Search Console authenticated")
	fmt.Println()
	return nil
}

func loginDataForSEOInteractive(scanner *bufio.Scanner) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println()

	fmt.Print("  DataForSEO Login (email): ")
	if !scanner.Scan() {
		return fmt.Errorf("interrupted")
	}
	login := strings.TrimSpace(scanner.Text())
	if login == "" {
		return fmt.Errorf("login cannot be empty")
	}
	fmt.Printf("  DataForSEO Login: %s\n", login)

	fmt.Print("  DataForSEO Password: ")
	if !scanner.Scan() {
		return fmt.Errorf("interrupted")
	}
	password := strings.TrimSpace(scanner.Text())
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	fmt.Printf("  DataForSEO Password: %s\n", redactValue(password))

	if err := cfg.Set("dataforseo_login", login); err != nil {
		return fmt.Errorf("failed to set DataForSEO login: %w", err)
	}
	if err := cfg.Set("dataforseo_password", password); err != nil {
		return fmt.Errorf("failed to set DataForSEO password: %w", err)
	}
	if err := cfg.Set("serp_provider", "dataforseo"); err != nil {
		return fmt.Errorf("failed to set serp provider: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("  ✓ DataForSEO configured")
	fmt.Println()
	return nil
}

func loginSerpAPIInteractive(scanner *bufio.Scanner) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println()

	fmt.Print("  SerpAPI Key: ")
	if !scanner.Scan() {
		return fmt.Errorf("interrupted")
	}
	apiKey := strings.TrimSpace(scanner.Text())
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	fmt.Printf("  SerpAPI Key: %s\n", redactValue(apiKey))

	if err := cfg.Set("serp_api_key", apiKey); err != nil {
		return fmt.Errorf("failed to set API key: %w", err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("  ✓ SerpAPI configured")
	fmt.Println()
	return nil
}

func redactValue(v string) string {
	if len(v) <= 8 {
		return "****"
	}
	return v[:4] + "****"
}
