package invoices

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newListCmd returns the `revenium invoices list` command.
// Endpoint: GET /v2/api/invoices. DoList handles the HATEOAS
// `_embedded.invoiceResourceList` unwrap and auto-pagination via
// cmd.AddListFlags.
func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all invoices",
		Args:  cobra.NoArgs,
		Example: `  # List all invoices
  revenium invoices list

  # List invoices as JSON
  revenium invoices list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var invoices []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/invoices", cmd.ListOptsFromFlags(c), &invoices); err != nil {
				return err
			}
			if len(invoices) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No invoices found.")
				return nil
			}
			return cmd.Output.Render(invoicesTableDef, invoicesToRows(invoices), invoices)
		},
	}

	cmd.AddListFlags(c)
	return c
}
