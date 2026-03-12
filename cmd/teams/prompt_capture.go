package teams

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// promptCaptureCmd is the parent prompt-capture subcommand under teams.
var promptCaptureCmd = &cobra.Command{
	Use:   "prompt-capture",
	Short: "Manage prompt capture settings for a team",
	Example: `  # View prompt capture settings
  revenium teams prompt-capture get team-123

  # Enable prompt capture
  revenium teams prompt-capture set team-123 --enabled true`,
}

// initPromptCapture registers prompt-capture subcommands. Called from teams.go init()
// to avoid file-ordering issues with Go's init() functions.
func initPromptCapture() {
	promptCaptureCmd.AddCommand(newPromptCaptureGetCmd())
	promptCaptureCmd.AddCommand(newPromptCaptureSetCmd())
}

// promptCaptureTableDef defines the table layout for prompt capture settings output.
var promptCaptureTableDef = output.TableDef{
	Headers:      []string{"Setting", "Value"},
	StatusColumn: -1,
}

// renderPromptSettings renders prompt capture settings as a key-value table or JSON.
func renderPromptSettings(settings map[string]interface{}) error {
	var rows [][]string
	for key, val := range settings {
		if key == "_links" {
			continue
		}
		rows = append(rows, []string{key, fmt.Sprint(val)})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})
	return cmd.Output.Render(promptCaptureTableDef, rows, settings)
}
