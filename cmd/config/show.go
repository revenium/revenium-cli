package config

import (
	"fmt"

	"github.com/spf13/cobra"

	internalconfig "github.com/revenium/revenium-cli/internal/config"
)

// newShowCmd creates the config show subcommand.
func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Example: `  # Show current config
  revenium config show`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internalconfig.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			apiKey := cfg.APIKey
			if apiKey == "" {
				apiKey = "(not set)"
			} else {
				apiKey = maskKey(apiKey)
			}

			teamID := cfg.TeamID
			if teamID == "" {
				teamID = "(not set)"
			}

			fmt.Fprintf(cmd.OutOrStdout(), "API Key:  %s\n", apiKey)
			fmt.Fprintf(cmd.OutOrStdout(), "API URL:  %s\n", cfg.APIURL)
			fmt.Fprintf(cmd.OutOrStdout(), "Team ID:  %s\n", teamID)
			return nil
		},
	}
}

// maskKey masks all but the last 4 characters of the key.
func maskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
