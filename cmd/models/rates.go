package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// ratesTableDef defines the table layout for AI model rate output.
var ratesTableDef = output.TableDef{
	Headers:      []string{"Model", "Provider", "Input $/M", "Output $/M", "Cache Read $/M", "Cache Write $/M"},
	StatusColumn: -1,
}

// toRateRows converts a slice of AIModelRate_Read maps to table row strings.
func toRateRows(rates []map[string]interface{}) [][]string {
	rows := make([][]string, len(rates))
	for i, r := range rates {
		rows[i] = []string{
			str(r, "model"),
			str(r, "provider"),
			str(r, "inputPricePerMillion"),
			str(r, "outputPricePerMillion"),
			str(r, "cacheReadPricePerMillion"),
			str(r, "cacheWritePricePerMillion"),
		}
	}
	return rows
}

// fetchModelRates fetches the flat AIModelRatesResponse_Read wrapper
// (`{"models":[...]}`) and extracts the array. This is NOT a HATEOAS
// `_embedded` list, so the generic list-unwrap helper won't find it — mirrors
// the existing fetchPricingDimensions pattern in pricing_list.go.
func fetchModelRates(c *cobra.Command) ([]map[string]interface{}, error) {
	var wrapper map[string]interface{}
	if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/ai/models/rates", nil, &wrapper); err != nil {
		return nil, err
	}
	models, ok := wrapper["models"].([]interface{})
	if !ok || len(models) == 0 {
		return nil, nil
	}
	result := make([]map[string]interface{}, 0, len(models))
	for _, m := range models {
		if mm, ok := m.(map[string]interface{}); ok {
			result = append(result, mm)
		}
	}
	return result, nil
}

// newRatesCmd returns the `revenium models rates` command.
//
// RES-01: GETs /v2/api/sources/ai/models/rates, a flat single-object response
// wrapping an array (not a HATEOAS list) — uses the raw Do() call, not the
// list-pagination helper.
func newRatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rates",
		Short: "List current AI model pricing rates",
		Args:  cobra.NoArgs,
		Example: `  # List model rates
  revenium models rates

  # As JSON
  revenium models rates --json`,
		RunE: func(c *cobra.Command, args []string) error {
			rates, err := fetchModelRates(c)
			if err != nil {
				return err
			}
			if len(rates) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No rates found.")
				return nil
			}
			return cmd.Output.Render(ratesTableDef, toRateRows(rates), rates)
		},
	}
}
