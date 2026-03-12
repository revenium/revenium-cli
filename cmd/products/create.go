package products

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var name, description string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new product",
		Example: `  # Create a product with name only
  revenium products create --name "My Product"

  # Create a product with description
  revenium products create --name "My Product" --description "A great product"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name": name,
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/products", body, &result); err != nil {
				return err
			}
			return renderProduct(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Product name")
	c.Flags().StringVar(&description, "description", "", "Product description")
	_ = c.MarkFlagRequired("name")

	return c
}
