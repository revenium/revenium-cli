// Package tools implements the tools CRUD commands for the Revenium CLI.
package tools

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent tools command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage tools",
	Example: `  # List all tools
  revenium tools list

  # Get a specific tool
  revenium tools get tool-1

  # Create a tool
  revenium tools create --name "My Tool" --tool-id my-tool --tool-type MCP_SERVER`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for tool output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Type", "Provider", "Enabled"},
	StatusColumn: -1,
}

// toRows converts a slice of tool maps to table row strings.
func toRows(tools []map[string]interface{}) [][]string {
	rows := make([][]string, len(tools))
	for i, t := range tools {
		rows[i] = []string{
			str(t, "id"),
			str(t, "name"),
			str(t, "toolType"),
			str(t, "toolProvider"),
			boolStr(t, "enabled"),
		}
	}
	return rows
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// boolStr safely extracts a boolean value from a map as a string.
func boolStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// renderTool renders a single tool as a single-row table or JSON.
func renderTool(tool map[string]interface{}) error {
	rows := [][]string{{
		str(tool, "id"),
		str(tool, "name"),
		str(tool, "toolType"),
		str(tool, "toolProvider"),
		boolStr(tool, "enabled"),
	}}
	return cmd.Output.Render(tableDef, rows, tool)
}
