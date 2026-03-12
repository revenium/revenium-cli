package alerts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newBudgetGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <anomaly-id>",
		Short: "Get budget progress for an anomaly",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get budget progress
  revenium alerts budget get anom-123

  # Get budget progress as JSON
  revenium alerts budget get anom-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			anomalyID := args[0]
			path := fmt.Sprintf("/v2/api/ai/alerts/%s/budget/progress", anomalyID)
			var progress map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &progress); err != nil {
				return err
			}
			currency := str(progress, "currency")
			rows := [][]string{{
				anomalyID,
				formatCurrency(floatVal(progress, "budgetThreshold"), currency),
				formatCurrency(floatVal(progress, "currentValue"), currency),
				formatCurrency(floatVal(progress, "remainingBudget"), currency),
				fmt.Sprintf("%.1f%%", floatVal(progress, "percentUsed")),
			}}
			return cmd.Output.Render(budgetTableDef, rows, progress)
		},
	}
}
