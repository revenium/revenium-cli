package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var imageTableDef = output.TableDef{
	Headers:      []string{"ID", "Model", "Count", "Cost"},
	StatusColumn: -1,
}

func newImageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "image",
		Short: "Query AI image metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query image metrics for last 24 hours
  revenium metrics image

  # Query with time range
  revenium metrics image --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/images")
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &metrics); err != nil {
				return err
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No metrics found.")
				return nil
			}
			return cmd.Output.Render(imageTableDef, toImageRows(metrics), metrics)
		},
	}
}

func toImageRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "id"),
			str(m, "model"),
			formatNumber(floatVal(m, "totalCount")),
			fmt.Sprintf("$%.4f", floatVal(m, "totalCost")),
		}
	}
	return rows
}
