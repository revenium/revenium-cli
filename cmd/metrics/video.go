package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var videoTableDef = output.TableDef{
	Headers:      []string{"ID", "Model", "Duration", "Cost", "Squad"},
	StatusColumn: -1,
}

func newVideoCmd() *cobra.Command {
	var squadID string
	c := &cobra.Command{
		Use:   "video",
		Short: "Query AI video metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query video metrics for last 24 hours
  revenium metrics video

  # Query with time range
  revenium metrics video --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z

  # Filter to a single squad (client-side)
  revenium metrics video --squad-id squad-loan-proc-12345`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/video")
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
				// Client-side exact-match filter (D-09) — no server-side
				// squadId query param exists on this endpoint.
				metrics = filterBySquadID(metrics, squadID)
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

	c.Flags().StringVar(&squadID, "squad-id", "", "Filter to a single squad (client-side, exact match on squadId — no server-side squad filter exists on this endpoint)")
	cmd.AddListFlags(c)
	return c
}

func toVideoRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "transactionId"),
			str(m, "model"),
			formatNumber(floatVal(m, "durationSeconds")),
			formatCost(floatVal(m, "totalCost")),
			squadCell(m),
		}
	}
	return rows
}
