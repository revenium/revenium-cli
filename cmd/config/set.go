package config

import (
	"fmt"

	"github.com/spf13/cobra"

	internalconfig "github.com/revenium/revenium-cli/internal/config"
)

// validKeys are the accepted configuration keys.
var validKeys = map[string]string{
	"key":     "api-key",
	"api-url": "api-url",
	"team-id": "team-id",
}

// newSetCmd creates the config set subcommand.
func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "set",
		Short:     "Set a configuration value",
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"key", "api-url", "team-id"},
		Example: `  # Set your API key
  revenium config set key your-api-key

  # Set your team ID
  revenium config set team-id your-team-id

  # Set custom API URL
  revenium config set api-url https://custom.api.com/profitstream`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			mappedKey, ok := validKeys[key]
			if !ok {
				return fmt.Errorf("unknown config key %q. Valid keys: key, api-url, team-id", key)
			}

			if err := internalconfig.Set(mappedKey, value); err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Set %s successfully.\n", key)
			return nil
		},
	}
}
