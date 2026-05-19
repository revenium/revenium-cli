package alerts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/output"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage budget alert thresholds",
	Example: `  # List all budget alerts
  revenium alerts budget list

  # Get budget progress for an anomaly
  revenium alerts budget get anom-123`,
}

func initBudget() {
	budgetCmd.AddCommand(newBudgetListCmd())
	budgetCmd.AddCommand(newBudgetGetCmd())
	budgetCmd.AddCommand(newBudgetCreateCmd())
	budgetCmd.AddCommand(newBudgetUpdateCmd())
	budgetCmd.AddCommand(newBudgetDeleteCmd())
}

var budgetTableDef = output.TableDef{
	Headers:      []string{"Alert ID", "Name", "Budget", "Current", "Remaining", "% Used", "Risk"},
	StatusColumn: 6,
}

// toBudgetRows converts a slice of budget maps to table row strings.
func toBudgetRows(budgets []map[string]interface{}) [][]string {
	rows := make([][]string, len(budgets))
	for i, b := range budgets {
		currency := str(b, "currency")
		rows[i] = []string{
			str(b, "alertId"),
			str(b, "name"),
			output.FormatCurrency(output.FloatVal(b, "threshold"), currency),
			output.FormatCurrency(output.FloatVal(b, "currentValue"), currency),
			output.FormatCurrency(output.FloatVal(b, "remaining"), currency),
			fmt.Sprintf("%.1f%%", output.FloatVal(b, "percentUsed")),
			str(b, "risk"),
		}
	}
	return rows
}
