// Package anomalies implements the anomaly detection rule CRUD commands for the Revenium CLI.
package anomalies

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent anomalies command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "anomalies",
	Short: "Manage AI anomaly detection rules",
	Example: `  # List all anomaly detection rules
  revenium anomalies list

  # Get a specific anomaly rule
  revenium anomalies get anom-123

  # Create an anomaly detection rule
  revenium anomalies create --name "High Cost Alert"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for anomaly output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Status"},
	StatusColumn: 2,
}

// toRows converts a slice of anomaly maps to table row strings.
func toRows(anomalies []map[string]interface{}) [][]string {
	rows := make([][]string, len(anomalies))
	for i, a := range anomalies {
		rows[i] = []string{
			str(a, "id"),
			str(a, "label"),
			str(a, "status"),
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

// renderAnomaly renders a single anomaly as a single-row table or JSON.
func renderAnomaly(anomaly map[string]interface{}) error {
	rows := [][]string{{
		str(anomaly, "id"),
		str(anomaly, "label"),
		str(anomaly, "status"),
	}}
	return cmd.Output.Render(tableDef, rows, anomaly)
}
