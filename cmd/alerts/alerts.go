// Package alerts implements the AI alert and budget threshold commands for the Revenium CLI.
package alerts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent alerts command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage AI alerts and budget thresholds",
	Example: `  # List all AI alerts
  revenium alerts list

  # Get a specific AI alert
  revenium alerts get alert-123

  # List budget alert thresholds
  revenium alerts budget list`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(budgetCmd)
	initBudget()
}

// alertTableDef defines the table layout for alert output.
var alertTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Created"},
	StatusColumn: -1,
}

// toAlertRows converts a slice of alert maps to table row strings.
func toAlertRows(alerts []map[string]interface{}) [][]string {
	rows := make([][]string, len(alerts))
	for i, a := range alerts {
		rows[i] = []string{
			str(a, "id"),
			str(a, "label"),
			str(a, "created"),
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

// renderAlert renders a single alert as a single-row table or JSON.
func renderAlert(alert map[string]interface{}) error {
	rows := [][]string{{
		str(alert, "id"),
		str(alert, "label"),
		str(alert, "created"),
	}}
	return cmd.Output.Render(alertTableDef, rows, alert)
}
