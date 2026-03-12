package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newPricingListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <model-id>",
		Short: "List pricing dimensions for a model",
		Args:  cobra.ExactArgs(1),
		Example: `  # List pricing dimensions
  revenium models pricing list abc-123

  # List as JSON
  revenium models pricing list abc-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			modelID := args[0]
			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions", modelID)

			var dimensions []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, &dimensions); err != nil {
				return err
			}
			if len(dimensions) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No pricing dimensions found.")
				return nil
			}
			return cmd.Output.Render(pricingTableDef, toPricingRows(dimensions), dimensions)
		},
	}
}
