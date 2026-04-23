package commands

import (
	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/pkg/output"
	"github.com/spf13/cobra"
)

// NewConfigCmd returns the config command group.
func NewConfigCmd(format *string, verbose *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set Sageo CLI configuration values",
		Long: `Manage sageo CLI configuration.

Config path defaults to ~/.config/sageo/config.json and can be overridden with SAGEO_CONFIG.`,
	}

	cmd.AddCommand(
		newConfigShowCmd(format, verbose),
		newConfigGetCmd(format, verbose),
		newConfigSetCmd(format, verbose),
		newConfigPathCmd(format, verbose),
	)

	return cmd
}

func newConfigShowCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show full config with sensitive fields redacted",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedErrorWithHint(output.ErrConfigLoadFailed, "failed to load config", "Run `sageo config list` to inspect your config, or re-run `sageo init --url <site>` if the project is new.", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(cfg.Redacted(), map[string]any{
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}
}

func newConfigGetCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a single config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedErrorWithHint(output.ErrConfigLoadFailed, "failed to load config", "Run `sageo config list` to inspect your config, or re-run `sageo init --url <site>` if the project is new.", err, nil, output.Format(*format))
			}

			value, err := cfg.Get(args[0])
			if err != nil {
				return output.PrintCodedError(output.ErrConfigGetFailed, "unknown config key", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(map[string]any{
				"key":   args[0],
				"value": value,
			}, map[string]any{
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}
}

func newConfigSetCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a single config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return output.PrintCodedErrorWithHint(output.ErrConfigLoadFailed, "failed to load config", "Run `sageo config list` to inspect your config, or re-run `sageo init --url <site>` if the project is new.", err, nil, output.Format(*format))
			}

			if err := cfg.Set(args[0], args[1]); err != nil {
				return output.PrintCodedError(output.ErrConfigGetFailed, "unknown config key", err, nil, output.Format(*format))
			}

			if err := cfg.Save(); err != nil {
				return output.PrintCodedError(output.ErrConfigSaveFailed, "failed to save config", err, nil, output.Format(*format))
			}

			return output.PrintSuccess(map[string]any{
				"status": "ok",
				"key":    args[0],
			}, map[string]any{
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}
}

func newConfigPathCmd(format *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show active config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.PrintSuccess(map[string]any{
				"path": config.Path(),
			}, map[string]any{
				"verbose": *verbose,
			}, output.Format(*format))
		},
	}
}
