// Package refunds implements the read-only refunds commands for the Revenium CLI.
package refunds

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent refunds command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "refunds",
	Short: "Manage refunds",
	Example: `  # List all refunds
  revenium refunds list

  # Get a specific refund
  revenium refunds get ref-123`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
}

// refundsTableDef defines the shared column layout for the list/get verbs.
var refundsTableDef = output.TableDef{
	Headers:      []string{"ID", "Amount", "Currency", "Status"},
	StatusColumn: 3,
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// refundsToRows converts a slice of refund maps to table row strings.
func refundsToRows(refunds []map[string]interface{}) [][]string {
	rows := make([][]string, len(refunds))
	for i, r := range refunds {
		rows[i] = []string{
			str(r, "id"),
			str(r, "amount"),
			str(r, "currency"),
			str(r, "status"),
		}
	}
	return rows
}

// renderRefund renders a single refund as a single-row table or JSON.
func renderRefund(refund map[string]interface{}) error {
	rows := [][]string{{
		str(refund, "id"),
		str(refund, "amount"),
		str(refund, "currency"),
		str(refund, "status"),
	}}
	return cmd.Output.Render(refundsTableDef, rows, refund)
}
