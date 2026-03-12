package products

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all products",
		Args:  cobra.NoArgs,
		Example: `  # List all products
  revenium products list

  # List products as JSON
  revenium products list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var products []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/products", &products); err != nil {
				return err
			}
			if len(products) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No products found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(products), products)
		},
	}
}
