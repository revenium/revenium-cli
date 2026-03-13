package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var aiTableDef = output.TableDef{
	Headers:      []string{"ID", "Model", "Input", "Output", "Cached", "Reasoning", "TTFT", "Tok/Min", "Duration", "Stop Reason", "Cost", "Organization", "Agent", "Subscriber"},
	StatusColumn: -1,
}

func newAICmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "ai",
		Short: "Query AI metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query AI metrics for last 24 hours
  revenium metrics ai

  # Query AI metrics with time range
  revenium metrics ai --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai")
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
			return cmd.Output.Render(aiTableDef, toAIRows(metrics), metrics)
		},
	}

	cmd.AddListFlags(c)
	return c
}

func toAIRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "transactionId"),
			str(m, "model"),
			formatNumber(floatVal(m, "inputTokenCount")),
			formatNumber(floatVal(m, "outputTokenCount")),
			formatNumber(floatVal(m, "cacheReadTokenCount")),
			formatNumber(floatVal(m, "reasoningTokenCount")),
			formatDuration(floatVal(m, "timeToFirstToken")),
			formatNumber(floatVal(m, "tokensPerMinute")),
			formatDuration(floatVal(m, "requestDuration")),
			str(m, "stopReason"),
			formatCost(floatVal(m, "totalCost")),
			nestedStr(m, "organization", "label"),
			str(m, "agent"),
			nestedStr(m, "subscriberCredential", "label"),
		}
	}
	return rows
}
