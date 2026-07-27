// Package periodcharges implements the read-only period-charges commands for the Revenium CLI.
package periodcharges

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent period-charges command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "period-charges",
	Short: "List and inspect period charges",
	Example: `  # List all period charges
  revenium period-charges list

  # Get a specific period charge
  revenium period-charges get pc-123`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
}

// tableDef defines the shared 3-column layout for the list/get verbs.
// Period charges have no natural lifecycle status field, so StatusColumn is -1.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Amount", "Currency"},
	StatusColumn: -1,
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// toRows converts a slice of period-charge maps to 3-col table row strings.
func toRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, m := range items {
		rows[i] = []string{
			str(m, "id"),
			str(m, "amount"),
			str(m, "currency"),
		}
	}
	return rows
}

// renderPeriodCharge renders a single period charge as a single-row table or JSON.
func renderPeriodCharge(pc map[string]interface{}) error {
	rows := [][]string{{
		str(pc, "id"),
		str(pc, "amount"),
		str(pc, "currency"),
	}}
	return cmd.Output.Render(tableDef, rows, pc)
}
