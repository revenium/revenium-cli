package invoices

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newGetCmd returns the `revenium invoices get <id>` command.
// Endpoint: GET /v2/api/invoices/{url.PathEscape(id)}.
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an invoice by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get an invoice by ID
  revenium invoices get inv-123

  # Get an invoice as JSON
  revenium invoices get inv-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/invoices/%s", url.PathEscape(args[0]))
			var invoice map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &invoice); err != nil {
				return err
			}
			return renderInvoice(invoice)
		},
	}
}
