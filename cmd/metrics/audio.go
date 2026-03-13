package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var audioTableDef = output.TableDef{
	Headers:      []string{"ID", "Model", "Duration", "Cost"},
	StatusColumn: -1,
}

func newAudioCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "audio",
		Short: "Query AI audio metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query audio metrics for last 24 hours
  revenium metrics audio

  # Query with time range
  revenium metrics audio --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/audio")
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
			return cmd.Output.Render(audioTableDef, toAudioRows(metrics), metrics)
		},
	}

	cmd.AddListFlags(c)
	return c
}

func toAudioRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "id"),
			str(m, "model"),
			formatNumber(floatVal(m, "totalDuration")),
			fmt.Sprintf("$%.4f", floatVal(m, "totalCost")),
		}
	}
	return rows
}
