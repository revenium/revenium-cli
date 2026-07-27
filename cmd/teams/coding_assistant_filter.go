package teams

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// codingAssistantFilterCmd is the parent coding-assistant-filter subcommand under teams.
var codingAssistantFilterCmd = &cobra.Command{
	Use:   "coding-assistant-filter",
	Short: "Manage coding-assistant filter settings for a team",
	Example: `  # View coding-assistant filter settings
  revenium teams coding-assistant-filter get team-123

  # Set the allowed API rate providers
  revenium teams coding-assistant-filter set team-123 --api-rate-providers ClaudeCode,CursorIde`,
}

// initCodingAssistantFilter registers coding-assistant-filter subcommands. Called
// from teams.go init() to avoid file-ordering issues with Go's init() functions
// (mirrors initPromptCapture).
func initCodingAssistantFilter() {
	codingAssistantFilterCmd.AddCommand(newCodingAssistantFilterGetCmd())
	codingAssistantFilterCmd.AddCommand(newCodingAssistantFilterSetCmd())
}

// codingAssistantFilterTableDef defines the table layout for coding-assistant
// filter settings output.
var codingAssistantFilterTableDef = output.TableDef{
	Headers:      []string{"Setting", "Value"},
	StatusColumn: -1,
}

// renderCodingAssistantFilterSettings renders coding-assistant filter settings
// as a key-value table or JSON.
func renderCodingAssistantFilterSettings(settings map[string]interface{}) error {
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
	return cmd.Output.Render(codingAssistantFilterTableDef, rows, settings)
}
