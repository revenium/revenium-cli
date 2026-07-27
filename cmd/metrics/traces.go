package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var tracesTableDef = output.TableDef{
	Headers:      []string{"Trace ID", "Entries", "Model", "Total Tokens", "Total Cost", "Squad"},
	StatusColumn: -1,
}

func newTracesCmd() *cobra.Command {
	var squadID string
	c := &cobra.Command{
		Use:   "traces",
		Short: "Query AI traces",
		Args:  cobra.NoArgs,
		Example: `  # Query traces for last 24 hours
  revenium metrics traces

  # Query traces with time range
  revenium metrics traces --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z

  # Filter to a single squad (client-side)
  revenium metrics traces --squad-id squad-loan-proc-12345`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/traces")
			opts := cmd.ListOptsFromFlags(c)
			if c.Flags().Changed("squad-id") {
				// D-10/Pitfall 5: force a full-pageset fetch so a match on a
				// later page isn't missed. FetchAll alone is not enough —
				// api.Client.DoList only fetches all pages when neither Page
				// nor PageSize is explicitly set, so any explicit --page/
				// --page-size must also be cleared here.
				opts.FetchAll = true
				opts.Page = -1
				opts.PageSize = -1
			}
			if err := cmd.APIClient.DoList(c.Context(), path, opts, &metrics); err != nil {
				return err
			}
			if c.Flags().Changed("squad-id") {
				// Client-side exact-match filter (D-09), applied to the raw
				// pre-grouping slice so both the JSON-mode (raw ungrouped)
				// and table-mode (grouped) outputs reflect the filter
				// consistently — filter-then-group, not group-then-filter
				// (Pitfall 3).
				metrics = filterBySquadID(metrics, squadID)
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

	c.Flags().StringVar(&squadID, "squad-id", "", "Filter to a single squad (client-side, exact match on squadId — no server-side squad filter exists on this endpoint)")
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
				"squadId":     str(m, "squadId"), // threaded through aggregation (Pitfall 3 fix)
				"totalTokens": 0.0,
				"totalCost":   0.0,
			}
			order = append(order, tid)
		}
		g := groups[tid]
		g["count"] = floatVal(g, "count") + 1
		g["totalTokens"] = floatVal(g, "totalTokens") + floatVal(m, "totalTokenCount")
		g["totalCost"] = floatVal(g, "totalCost") + floatVal(m, "totalCost")
		// First-non-empty-wins for squadId, same convention as model/source.
		if str(g, "squadId") == "" {
			g["squadId"] = str(m, "squadId")
		}
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
			squadCell(m),
		}
	}
	return rows
}
