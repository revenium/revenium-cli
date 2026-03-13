package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var squadsTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Executions", "Status"},
	StatusColumn: 3,
}

func newSquadsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "squads",
		Short: "Query squad metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query squad metrics for last 24 hours
  revenium metrics squads

  # Query squad metrics with time range
  revenium metrics squads --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/squads")
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &metrics); err != nil {
				return err
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No squad metrics found.")
				return nil
			}
			return cmd.Output.Render(squadsTableDef, toSquadsRows(metrics), metrics)
		},
	}

	cmd.AddListFlags(c)
	return c
}

func toSquadsRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "id"),
			str(m, "name"),
			formatNumber(floatVal(m, "executions")),
			str(m, "status"),
		}
	}
	return rows
}
