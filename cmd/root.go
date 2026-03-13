// Package cmd implements the Cobra command tree for the Revenium CLI.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd/config"
	"github.com/revenium/revenium-cli/internal/api"
	internalconfig "github.com/revenium/revenium-cli/internal/config"
	"github.com/revenium/revenium-cli/internal/output"
)

// APIClient is the shared API client, initialized in PersistentPreRunE
// and accessible to all subcommands.
var APIClient *api.Client

// Output is the shared output formatter, initialized in PersistentPreRunE
// and accessible to all subcommands.
var Output *output.Formatter

// verbose controls verbose output mode.
var verbose bool

// jsonMode controls JSON output mode.
var jsonMode bool

// quiet controls quiet output mode (suppress non-error output).
var quiet bool

// yesMode controls whether to skip confirmation prompts.
var yesMode bool

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
		// Always initialize the output formatter so all commands can use it
		Output = output.New(jsonMode, quiet)

		// Skip config loading for version, config, and completion commands
		if cmd.Name() == "version" || cmd.Parent() != nil && cmd.Parent().Name() == "config" || cmd.Name() == "config" {
			return nil
		}
		// Completion commands (bash, zsh, fish, powershell) are children of
		// the Cobra-generated "completion" command and don't need API access.
		for p := cmd; p != nil; p = p.Parent() {
			if p.Name() == "completion" {
				return nil
			}
		}

		cfg, err := internalconfig.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if cfg.APIKey == "" {
			return fmt.Errorf("No API key configured. Run `revenium config set key <your-key>` to fix.")
		}

		APIClient = api.NewClient(cfg.APIURL, cfg.APIKey, cfg.TeamID, cfg.TenantID, cfg.OwnerID, verbose)
		return nil
	},
}

// JSONMode returns true if --json flag is active. Used by main.go to
// decide error rendering format without importing the output package.
func JSONMode() bool {
	return jsonMode
}

// YesMode returns true if --yes flag is active. Used by subcommands
// to skip confirmation prompts.
func YesMode() bool {
	return yesMode
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVarP(&yesMode, "yes", "y", false, "Skip confirmation prompts")

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

// RegisterCommand adds a command to the root command with the given group ID.
// This is used by main.go to register resource commands without creating
// circular imports (since resource packages import cmd for APIClient/Output).
func RegisterCommand(c *cobra.Command, groupID string) {
	c.GroupID = groupID
	rootCmd.AddCommand(c)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
