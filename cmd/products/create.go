package products

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
	var name, description, version, planType, currency, period string

	c := &cobra.Command{
		Use:   "create",
		Short:       "Create a new product",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Create a product with name only
  revenium products create --name "My Product"

  # Create a product with description
  revenium products create --name "My Product" --description "A great product"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name":    name,
				"version": version,
				"plan": map[string]interface{}{
					"name":     name,
					"type":     planType,
					"currency": currency,
					"period":   period,
				},
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "product", "/v2/api/products", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreateWithOwner(c.Context(), "/v2/api/products", body, &result); err != nil {
				return err
			}
			return renderProduct(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Product name")
	c.Flags().StringVar(&description, "description", "", "Product description")
	c.Flags().StringVar(&version, "version", "1.0.0", "Product version")
	c.Flags().StringVar(&planType, "plan-type", "SUBSCRIPTION", "Plan type (e.g., SUBSCRIPTION)")
	c.Flags().StringVar(&currency, "currency", "USD", "Currency code (e.g., USD, EUR)")
	c.Flags().StringVar(&period, "period", "MONTH", "Billing period (e.g., MONTH, YEAR)")
	_ = c.MarkFlagRequired("name")

	return c
}
