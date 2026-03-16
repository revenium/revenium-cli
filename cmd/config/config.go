// Package config provides the config command group for the Revenium CLI.
// This is cmd/config, not to be confused with internal/config.
package config

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent config command, exported for registration in root.go.
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long: `Manage CLI configuration stored at ~/.config/revenium/config.yaml.

Valid keys:
  key           Your Revenium API key (required)
  api-url       API base URL (default https://api.revenium.ai/profitstream)
  team-id       Team ID for multi-tenant access
  tenant-id     Tenant ID
  owner-id      Owner ID

Environment variables override config file values:
  REVENIUM_API_KEY    Overrides "key"
  REVENIUM_API_URL    Overrides "api-url"
  REVENIUM_TEAM_ID    Overrides "team-id"`,
	Example: `  # Set API key
  revenium config set key your-api-key

  # Set team ID
  revenium config set team-id your-team-id

  # Set custom API URL
  revenium config set api-url https://custom.api.com/profitstream

  # Show current configuration
  revenium config show`,
}

func init() {
	Cmd.AddCommand(newSetCmd())
	Cmd.AddCommand(newShowCmd())
}
