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
	Example: `  # Set API key
  revenium config set key your-api-key

  # Show current configuration
  revenium config show`,
}

func init() {
	Cmd.AddCommand(newSetCmd())
	Cmd.AddCommand(newShowCmd())
}
