package products

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a product by ID",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get a product by ID
  revenium products get prod-123

  # Get a product as JSON
  revenium products get prod-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var product map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/products/"+args[0], nil, &product); err != nil {
				return err
			}
			return renderProduct(product)
		},
	}
}
