package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newPricingUpdateCmd() *cobra.Command {
	var (
		name    string
		dimType string
		price   float64
	)

	c := &cobra.Command{
		Use:   "update <model-id> <dimension-id>",
		Short: "Update a pricing dimension",
		Args:  cobra.ExactArgs(2),
		Example: `  # Update a pricing dimension price
  revenium models pricing update model-123 dim-456 --price 0.005`,
		RunE: func(c *cobra.Command, args []string) error {
			modelID := args[0]
			dimID := args[1]
			body := make(map[string]interface{})

			if c.Flags().Changed("name") {
				body["name"] = name
			}
			if c.Flags().Changed("type") {
				body["dimensionType"] = dimType
			}
			if c.Flags().Changed("price") {
				body["unitPrice"] = price
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions/%s", modelID, dimID)
			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, body, &result); err != nil {
				return err
			}
			return renderPricingDimension(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Dimension name")
	c.Flags().StringVar(&dimType, "type", "", "Dimension type (e.g., input, output)")
	c.Flags().Float64Var(&price, "price", 0, "Unit price")

	return c
}
