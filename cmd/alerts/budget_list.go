package alerts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newBudgetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all budget alerts",
		Args:  cobra.NoArgs,
		Example: `  # List all budget alerts
  revenium alerts budget list

  # List budget alerts as JSON
  revenium alerts budget list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var budgets []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/ai/alerts/budgets/portfolio", &budgets); err != nil {
				return err
			}
			if len(budgets) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No budget alerts found.")
				return nil
			}
			return cmd.Output.Render(budgetTableDef, toBudgetRows(budgets), budgets)
		},
	}
}
