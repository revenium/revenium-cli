// Package cmd implements the Cobra command tree for the Revenium CLI.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd/config"
	"github.com/revenium/revenium-cli/internal/api"
	internalconfig "github.com/revenium/revenium-cli/internal/config"
)

// APIClient is the shared API client, initialized in PersistentPreRunE
// and accessible to all subcommands.
var APIClient *api.Client

// verbose controls verbose output mode.
var verbose bool

// rootCmd is the base command for the Revenium CLI.
var rootCmd = &cobra.Command{
	Use:   "revenium",
	Short: "Manage your Revenium account",
	Long:  "Manage your Revenium account from the command line. Configure API access, manage resources, and monitor usage.",
	Example: `  # Configure your API key
  revenium config set key your-api-key

  # View current configuration
  revenium config show

  # List sources (after configuration)
  revenium sources list`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for version and config commands
		if cmd.Name() == "version" || cmd.Parent() != nil && cmd.Parent().Name() == "config" || cmd.Name() == "config" {
			return nil
		}

		cfg, err := internalconfig.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.APIKey == "" {
			return fmt.Errorf("No API key configured. Run `revenium config set key <your-key>` to fix.")
		}

		APIClient = api.NewClient(cfg.APIURL, cfg.APIKey, verbose)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	rootCmd.AddGroup(
		&cobra.Group{ID: "resources", Title: "Core Resources:"},
		&cobra.Group{ID: "monitoring", Title: "Monitoring:"},
		&cobra.Group{ID: "config", Title: "Configuration:"},
	)

	// Register commands
	configCmd := config.Cmd
	configCmd.GroupID = "config"
	rootCmd.AddCommand(configCmd)

	versionCmd := newVersionCmd()
	versionCmd.GroupID = "config"
	rootCmd.AddCommand(versionCmd)

	rootCmd.SetHelpCommandGroupID("config")
	rootCmd.SetCompletionCommandGroupID("config")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
