package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var apiMetricsTableDef = output.TableDef{
	Headers:      []string{"ID", "Source", "Requests", "Errors", "Latency"},
	StatusColumn: -1,
}

func newAPIMetricsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Query API metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query API metrics for last 24 hours
  revenium metrics api

  # Query API metrics with time range
  revenium metrics api --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/api")
			if err := cmd.APIClient.DoList(c.Context(), path, &metrics); err != nil {
				return err
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No metrics found.")
				return nil
			}
			return cmd.Output.Render(apiMetricsTableDef, toAPIMetricsRows(metrics), metrics)
		},
	}
}

func toAPIMetricsRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "id"),
			str(m, "source"),
			formatNumber(floatVal(m, "requests")),
			formatNumber(floatVal(m, "errors")),
			fmt.Sprintf("%.2fms", floatVal(m, "latency")),
		}
	}
	return rows
}
