package billing

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newWorkspacesCmd()) }

// workspacesTableDef defines the table layout for billing workspace-level
// cost metrics rows. StatusColumn: -1 — no natural status field. Distinct
// from the top-level `cmd/workspaces` package — this is the billing
// sub-verb `revenium billing workspaces`, a different endpoint entirely.
var workspacesTableDef = output.TableDef{
	Headers:      []string{"Workspace", "Cost"},
	StatusColumn: -1,
}

// toWorkspaceRows converts a slice of billing workspace-metrics maps to
// table rows.
func toWorkspaceRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, it := range items {
		name := str(it, "workspace")
		if name == "" {
			name = str(it, "workspaceId")
		}
		rows[i] = []string{name, formatCost(floatVal(it, "cost"))}
	}
	return rows
}

// newWorkspacesCmd returns `revenium billing workspaces`.
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
func newWorkspacesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "workspaces",
		Short: "List billing workspace-level cost metrics",
		Args:  cobra.NoArgs,
		Example: `  # List billing workspace-level cost metrics
  revenium billing workspaces

  # Get the full wrapper as JSON, including totalCost/costByProvider
  revenium billing workspaces --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var wrapper map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/billing/workspaces", nil, &wrapper); err != nil {
				return err
			}

			items := extractEmbeddedItems(wrapper)
			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					// Still preserve aggregates on an empty page.
					return cmd.Output.RenderJSON(wrapper)
				}
				fmt.Fprintln(c.OutOrStdout(), "No billing workspace data found.")
				return nil
			}
			// rawData = wrapper (NOT items) — this is the load-bearing
			// difference from the standard list pattern; --json must show
			// totalCost etc.
			return cmd.Output.Render(workspacesTableDef, toWorkspaceRows(items), wrapper)
		},
	}
	return c
}
