// Package charts implements the chart definition CRUD commands for the Revenium CLI.
package charts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent charts command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "charts",
	Short: "Manage chart definitions",
	Example: `  # List all chart definitions
  revenium charts list

  # Get a specific chart definition
  revenium charts get chart-123

  # Create a chart definition
  revenium charts create --label "Revenue Chart"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for chart output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Label", "Type", "Created"},
	StatusColumn: -1,
}

// toRows converts a slice of chart maps to table row strings.
func toRows(charts []map[string]interface{}) [][]string {
	rows := make([][]string, len(charts))
	for i, c := range charts {
		rows[i] = []string{
			str(c, "id"),
			str(c, "label"),
			str(c, "type"),
			str(c, "created"),
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

// renderChart renders a single chart as a single-row table or JSON.
func renderChart(chart map[string]interface{}) error {
	rows := [][]string{{
		str(chart, "id"),
		str(chart, "label"),
		str(chart, "type"),
		str(chart, "created"),
	}}
	return cmd.Output.Render(tableDef, rows, chart)
}
