package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var tracesTableDef = output.TableDef{
	Headers:      []string{"Trace ID", "Entries", "Model", "Total Tokens", "Total Cost"},
	StatusColumn: -1,
}

func newTracesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "traces",
		Short: "Query AI traces",
		Args:  cobra.NoArgs,
		Example: `  # Query traces for last 24 hours
  revenium metrics traces

  # Query traces with time range
  revenium metrics traces --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/traces")
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &metrics); err != nil {
				return err
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No traces found.")
				return nil
			}
			// JSON mode passes raw ungrouped data
			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(metrics)
			}
			grouped := groupByTraceId(metrics)
			return cmd.Output.Render(tracesTableDef, toTracesRows(grouped), grouped)
		},
	}

	cmd.AddListFlags(c)
	return c
}

// groupByTraceId aggregates trace entries by traceId, summing tokens and cost.
func groupByTraceId(metrics []map[string]interface{}) []map[string]interface{} {
	groups := make(map[string]map[string]interface{})
	var order []string

	for _, m := range metrics {
		tid := str(m, "traceId")
		if _, exists := groups[tid]; !exists {
			groups[tid] = map[string]interface{}{
				"traceId":     tid,
				"count":       0.0,
				"model":       str(m, "model"),
				"source":      str(m, "source"),
				"totalTokens": 0.0,
				"totalCost":   0.0,
			}
			order = append(order, tid)
		}
		g := groups[tid]
		g["count"] = floatVal(g, "count") + 1
		g["totalTokens"] = floatVal(g, "totalTokens") + floatVal(m, "totalTokenCount")
		g["totalCost"] = floatVal(g, "totalCost") + floatVal(m, "totalCost")
	}

	result := make([]map[string]interface{}, len(order))
	for i, tid := range order {
		result[i] = groups[tid]
	}
	return result
}

func toTracesRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "traceId"),
			formatNumber(floatVal(m, "count")),
			str(m, "model"),
			formatNumber(floatVal(m, "totalTokens")),
			formatCost(floatVal(m, "totalCost")),
		}
	}
	return rows
}
