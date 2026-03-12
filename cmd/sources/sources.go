// Package sources implements the sources CRUD commands for the Revenium CLI.
package sources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent sources command, exported for registration in root.go.
var Cmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage sources",
	Example: `  # List all sources
  revenium sources list

  # Get a specific source
  revenium sources get abc-123`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
}

// tableDef defines the table layout for source output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Type", "Status"},
	StatusColumn: 3,
}

// toRows converts a slice of source maps to table row strings.
func toRows(sources []map[string]interface{}) [][]string {
	rows := make([][]string, len(sources))
	for i, s := range sources {
		rows[i] = []string{
			str(s, "id"),
			str(s, "name"),
			str(s, "type"),
			str(s, "status"),
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

// renderSource renders a single source as a single-row table or JSON.
func renderSource(source map[string]interface{}) error {
	rows := [][]string{{
		str(source, "id"),
		str(source, "name"),
		str(source, "type"),
		str(source, "status"),
	}}
	return cmd.Output.Render(tableDef, rows, source)
}
