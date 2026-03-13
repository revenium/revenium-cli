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
			dimensions, err := fetchPricingDimensions(c, modelID)
			if err != nil {
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

// fetchPricingDimensions fetches pricing dimensions for a model from the /pricing endpoint.
func fetchPricingDimensions(c *cobra.Command, modelID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing", modelID)
	var wrapper map[string]interface{}
	if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &wrapper); err != nil {
		return nil, err
	}
	dims, ok := wrapper["dimensions"].([]interface{})
	if !ok || len(dims) == 0 {
		return nil, nil
	}
	result := make([]map[string]interface{}, 0, len(dims))
	for _, d := range dims {
		if m, ok := d.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result, nil
}
