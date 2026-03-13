package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newPricingUpdateCmd() *cobra.Command {
	var (
		price float64
	)

	c := &cobra.Command{
		Use:   "update <model-id> <dimension-id>",
		Short:       "Update a pricing dimension",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(2), cmd.ValidResourceID),
		Example: `  # Update a pricing dimension price
  revenium models pricing update model-123 dim-456 --price 0.005`,
		RunE: func(c *cobra.Command, args []string) error {
			modelID := args[0]
			dimID := args[1]

			if !c.Flags().Changed("price") {
				return fmt.Errorf("no fields specified to update")
			}

			if cmd.DryRun() {
				path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions/%s", modelID, dimID)
				return dryrun.Render(cmd.Output, "update", "pricing dimension", path, map[string]interface{}{"unitPrice": price})
			}

			// Fetch existing dimension from the pricing list endpoint
			dimensions, err := fetchPricingDimensions(c, modelID)
			if err != nil {
				return err
			}

			var existing map[string]interface{}
			for _, d := range dimensions {
				if id, _ := d["id"].(string); id == dimID {
					existing = d
					break
				}
			}
			if existing == nil {
				return fmt.Errorf("pricing dimension %s not found", dimID)
			}

			existing["unitPrice"] = price

			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions/%s", modelID, dimID)
			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, existing, &result); err != nil {
				return err
			}
			return renderPricingDimension(result)
		},
	}

	c.Flags().Float64Var(&price, "price", 0, "Unit price")

	return c
}
