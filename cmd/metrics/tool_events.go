package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var toolEventsTableDef = output.TableDef{
	Headers:      []string{"ID", "Tool", "Invocations", "Cost"},
	StatusColumn: -1,
}

func newToolEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tool-events",
		Short: "Query tool event metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query tool event metrics for last 24 hours
  revenium metrics tool-events

  # Query tool event metrics with time range
  revenium metrics tool-events --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/tool/events")
			if err := cmd.APIClient.DoList(c.Context(), path, &metrics); err != nil {
				return err
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No tool event metrics found.")
				return nil
			}
			return cmd.Output.Render(toolEventsTableDef, toToolEventsRows(metrics), metrics)
		},
	}
}

func toToolEventsRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "id"),
			str(m, "tool"),
			formatNumber(floatVal(m, "invocations")),
			fmt.Sprintf("$%.4f", floatVal(m, "totalCost")),
		}
	}
	return rows
}
