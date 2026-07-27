// Package invoices implements the read-only invoices commands for the Revenium CLI.
package invoices

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent invoices command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "invoices",
	Short: "Manage invoices",
	Example: `  # List all invoices
  revenium invoices list

  # Get a specific invoice
  revenium invoices get inv-123`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newDownloadCmd())
}

// invoicesTableDef defines the shared column layout for the list/get verbs.
var invoicesTableDef = output.TableDef{
	Headers:      []string{"ID", "Invoice Number", "State", "Total", "Currency"},
	StatusColumn: 2,
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// invoicesToRows converts a slice of invoice maps to table row strings.
func invoicesToRows(invoices []map[string]interface{}) [][]string {
	rows := make([][]string, len(invoices))
	for i, inv := range invoices {
		rows[i] = []string{
			str(inv, "id"),
			str(inv, "invoiceNumber"),
			str(inv, "state"),
			str(inv, "total"),
			str(inv, "currency"),
		}
	}
	return rows
}

// renderInvoice renders a single invoice as a single-row table or JSON.
func renderInvoice(invoice map[string]interface{}) error {
	rows := [][]string{{
		str(invoice, "id"),
		str(invoice, "invoiceNumber"),
		str(invoice, "state"),
		str(invoice, "total"),
		str(invoice, "currency"),
	}}
	return cmd.Output.Render(invoicesTableDef, rows, invoice)
}
