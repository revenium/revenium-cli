// Package cmd implements the Cobra command tree for the Revenium CLI.
package cmd

import (
	"fmt"
	"os"
	"strings"

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

// jsonMode controls JSON output mode (legacy --json flag).
var jsonMode bool

// outputFormat controls the output format (--output flag).
var outputFormat string

// fieldsFlag controls field filtering (--fields flag).
var fieldsFlag string

// quiet controls quiet output mode (suppress non-error output).
var quiet bool

// yesMode controls whether to skip confirmation prompts.
var yesMode bool

// dryRun controls dry-run mode for mutation commands.
var dryRun bool

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
		// Resolve effective JSON mode:
		// --json flag > --output flag > REVENIUM_OUTPUT_FORMAT env > default table
		effectiveJSON := jsonMode
		if !effectiveJSON {
			if outputFormat != "" {
				effectiveJSON = strings.EqualFold(outputFormat, "json")
			} else if envFormat := os.Getenv("REVENIUM_OUTPUT_FORMAT"); envFormat != "" {
				effectiveJSON = strings.EqualFold(envFormat, "json")
			}
		}

		// Always initialize the output formatter so all commands can use it
		Output = output.New(effectiveJSON, quiet)

		// Apply field filtering if --fields is set
		if fieldsFlag != "" {
			fields := strings.Split(fieldsFlag, ",")
			Output.SetFields(fields)
		}

		// Skip config loading for version, config, schema, and completion commands
		if cmd.Name() == "version" || cmd.Name() == "schema" || cmd.Parent() != nil && cmd.Parent().Name() == "config" || cmd.Name() == "config" {
			return nil
		}
		// Completion commands (bash, zsh, fish, powershell) are children of
		// the Cobra-generated "completion" command and don't need API access.
		// Only match when "completion" is a direct child of the root command
		// to avoid collisions with subcommands like "meter completion".
		for p := cmd; p != nil; p = p.Parent() {
			if p.Name() == "completion" && p.Parent() != nil && p.Parent().Parent() == nil {
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

// JSONMode returns true if JSON output mode is active (via --json, --output json,
// or REVENIUM_OUTPUT_FORMAT=json). Used by main.go to decide error rendering format.
func JSONMode() bool {
	if jsonMode {
		return true
	}
	if strings.EqualFold(outputFormat, "json") {
		return true
	}
	if envFormat := os.Getenv("REVENIUM_OUTPUT_FORMAT"); strings.EqualFold(envFormat, "json") {
		return true
	}
	return false
}

// YesMode returns true if --yes flag is active. Used by subcommands
// to skip confirmation prompts.
func YesMode() bool {
	return yesMode
}

// DryRun returns true if --dry-run flag is active.
func DryRun() bool {
	return dryRun
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "", "Output format: json, table (default table)")
	rootCmd.PersistentFlags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to include in output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVarP(&yesMode, "yes", "y", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview the action without executing it")

	// Hide --json (replaced by --output json, kept for backward compat)
	rootCmd.PersistentFlags().MarkHidden("json")

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

	SetupHelp(rootCmd)
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

// Root returns the root command. Used by the schema command to walk the
// command tree.
func Root() *cobra.Command {
	return rootCmd
}
