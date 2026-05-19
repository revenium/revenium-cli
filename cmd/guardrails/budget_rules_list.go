package guardrails

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newBudgetRulesListCmd returns the `revenium guardrails budget-rules list` command.
// Endpoint: GET /v2/api/ai/cost-controls (RESEARCH-corrected base, NOT the CONTEXT.md
// CF-12-22 `/v2/api/budget-rules` guess). Renders the package-level 3-col tableDef
// (ID / Name / Status) populated via toRows; empty list emits "No budget rules found."
// (D-06) for non-JSON or `[]` JSON. DoList handles HATEOAS `_embedded.objectList`
// unwrap and auto-pagination via cmd.AddListFlags (CF-12-19).
func newBudgetRulesListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List budget rules",
		Args:  cobra.NoArgs,
		Example: `  # List all budget rules
  revenium guardrails budget-rules list

  # List budget rules as JSON
  revenium guardrails budget-rules list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var rules []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/ai/cost-controls", cmd.ListOptsFromFlags(c), &rules); err != nil {
				return err
			}
			if len(rules) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No budget rules found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(rules), rules)
		},
	}

	cmd.AddListFlags(c)
	return c
}
