package billing

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newDailyCmd()) }

// dailyTableDef defines the table layout for the billing daily cost/credits
// breakdown rows. StatusColumn: -1 — no natural status field.
var dailyTableDef = output.TableDef{
	Headers:      []string{"Date", "Cost", "Credits Applied"},
	StatusColumn: -1,
}

// toDailyRows converts a slice of billing daily-breakdown maps to table rows.
func toDailyRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = []string{str(it, "date"), formatCost(floatVal(it, "cost")), formatCost(floatVal(it, "creditsApplied"))}
	}
	return rows
}

// newDailyCmd returns `revenium billing daily`.
//
// DEVIATION FROM the standard paginated-list client helper (CONTEXT.md
// D-04, RESEARCH Pitfall 2): this endpoint's response schema
// (AnalyticsPagedModel_Read) has a paginated `_embedded` array PLUS sibling
// aggregate fields (totalCost, costByProvider, creditsAppliedByProvider,
// lastRefreshDate) that live OUTSIDE _embedded. The standard list helper
// decodes straight into []map[string]interface{} and would
// silently discard every aggregate field — the entire point of a
// cost-attribution command. So: raw Do into a wrapper map, manual _embedded
// extraction (extractEmbeddedItems) for table rows, and the FULL wrapper
// (not just the array) passed as rawData to cmd.Output.Render so --json
// preserves the aggregates. Same landmine-density comment as
// cmd/users/me_period_charges.go's cursor deviation.
func newDailyCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "daily",
		Short: "Get billing daily cost/credits breakdown",
		Args:  cobra.NoArgs,
		Example: `  # Get daily cost/credits breakdown
  revenium billing daily

  # Get the full wrapper as JSON, including totalCost/costByProvider
  revenium billing daily --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var wrapper map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/billing/daily", nil, &wrapper); err != nil {
				return err
			}

			items := extractEmbeddedItems(wrapper)
			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					// Still preserve aggregates on an empty page.
					return cmd.Output.RenderJSON(wrapper)
				}
				fmt.Fprintln(c.OutOrStdout(), "No billing daily data found.")
				return nil
			}
			// rawData = wrapper (NOT items) — this is the load-bearing
			// difference from the standard list pattern; --json must show
			// totalCost etc.
			return cmd.Output.Render(dailyTableDef, toDailyRows(items), wrapper)
		},
	}
	return c
}
