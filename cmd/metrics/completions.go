package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var completionsTableDef = output.TableDef{
	Headers:      []string{"ID", "Model", "Tokens", "Cost"},
	StatusColumn: -1,
}

func newCompletionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completions",
		Short: "Query AI completion metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query completion metrics for last 24 hours
  revenium metrics completions

  # Query with time range
  revenium metrics completions --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/completions")
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
			return cmd.Output.Render(completionsTableDef, toCompletionRows(metrics), metrics)
		},
	}
}

func toCompletionRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "id"),
			str(m, "model"),
			formatNumber(floatVal(m, "totalTokens")),
			fmt.Sprintf("$%.4f", floatVal(m, "totalCost")),
		}
	}
	return rows
}
