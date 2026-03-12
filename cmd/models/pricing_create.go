package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newPricingCreateCmd() *cobra.Command {
	var (
		name       string
		dimType    string
		price      float64
	)

	c := &cobra.Command{
		Use:   "create <model-id>",
		Short: "Create a pricing dimension for a model",
		Args:  cobra.ExactArgs(1),
		Example: `  # Create a pricing dimension
  revenium models pricing create abc-123 --name "Input Tokens" --type input --price 0.003`,
		RunE: func(c *cobra.Command, args []string) error {
			modelID := args[0]
			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions", modelID)

			body := map[string]interface{}{
				"name":     name,
				"unitPrice": price,
			}
			if c.Flags().Changed("type") {
				body["dimensionType"] = dimType
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", path, body, &result); err != nil {
				return err
			}
			return renderPricingDimension(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Dimension name")
	c.Flags().StringVar(&dimType, "type", "", "Dimension type (e.g., input, output)")
	c.Flags().Float64Var(&price, "price", 0, "Unit price")
	_ = c.MarkFlagRequired("name")

	return c
}
