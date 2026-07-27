package refunds

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newListCmd returns the `revenium refunds list` command.
// Endpoint: GET /v2/api/refunds. DoList handles the HATEOAS
// `_embedded.refundResourceList` unwrap and auto-pagination via
// cmd.AddListFlags.
func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all refunds",
		Args:  cobra.NoArgs,
		Example: `  # List all refunds
  revenium refunds list

  # List refunds as JSON
  revenium refunds list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var refunds []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/refunds", cmd.ListOptsFromFlags(c), &refunds); err != nil {
				return err
			}
			if len(refunds) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No refunds found.")
				return nil
			}
			return cmd.Output.Render(refundsTableDef, refundsToRows(refunds), refunds)
		},
	}

	cmd.AddListFlags(c)
	return c
}
