package users

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() {
	meCmd.AddCommand(newMeInvoicesCmd())
}

var meInvoicesTableDef = output.TableDef{
	Headers:      []string{"ID", "Invoice Number", "State", "Total", "Currency"},
	StatusColumn: -1,
}

func meInvoicesToRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, m := range items {
		rows[i] = []string{
			str(m, "id"),
			str(m, "invoiceNumber"),
			str(m, "state"),
			str(m, "totalAmount"),
			str(m, "currency"),
		}
	}
	return rows
}

// newMeInvoicesCmd returns `revenium users me invoices`. Two-call pattern
// (05-RESEARCH RES-05): resolves the current user id via currentUserID, then
// GETs /v2/api/users/{id}/invoices.
func newMeInvoicesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "invoices",
		Short: "List invoices for the current user",
		Args:  cobra.NoArgs,
		Example: `  # List invoices for the current user
  revenium users me invoices`,
		RunE: func(c *cobra.Command, args []string) error {
			id, err := currentUserID(c)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v2/api/users/%s/invoices", url.PathEscape(id))
			var items []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &items); err != nil {
				return err
			}
			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No invoices found.")
				return nil
			}
			return cmd.Output.Render(meInvoicesTableDef, meInvoicesToRows(items), items)
		},
	}

	cmd.AddListFlags(c)
	return c
}
