package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var videoTableDef = output.TableDef{
	Headers:      []string{"ID", "Model", "Duration", "Cost"},
	StatusColumn: -1,
}

func newVideoCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "video",
		Short: "Query AI video metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query video metrics for last 24 hours
  revenium metrics video

  # Query with time range
  revenium metrics video --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/video")
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &metrics); err != nil {
				return err
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No metrics found.")
				return nil
			}
			return cmd.Output.Render(videoTableDef, toVideoRows(metrics), metrics)
		},
	}

	cmd.AddListFlags(c)
	return c
}

func toVideoRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "transactionId"),
			str(m, "model"),
			formatNumber(floatVal(m, "totalDuration")),
			formatCost(floatVal(m, "totalCost")),
		}
	}
	return rows
}
