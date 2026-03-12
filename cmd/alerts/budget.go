package alerts

import (
	"encoding/json"
	"fmt"
	"strings"

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

// formatCurrency formats a number as currency with commas and 2 decimal places.
// Uses "$" prefix for USD or empty currency. Other currencies use code prefix (e.g., "EUR 1,000.00").
func formatCurrency(amount float64, currency string) string {
	formatted := fmt.Sprintf("%.2f", amount)
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	negative := ""
	if strings.HasPrefix(intPart, "-") {
		negative = "-"
		intPart = intPart[1:]
	}
	if len(intPart) > 3 {
		var result []byte
		for i, c := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result = append(result, ',')
			}
			result = append(result, byte(c))
		}
		intPart = string(result)
	}
	symbol := "$"
	if currency != "" && currency != "USD" {
		symbol = currency + " "
	}
	return fmt.Sprintf("%s%s%s.%s", negative, symbol, intPart, parts[1])
}

// floatVal safely extracts a float64 from a map, handling float64 and json.Number types.
func floatVal(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return n
		case json.Number:
			f, _ := n.Float64()
			return f
		}
	}
	return 0
}

// toBudgetRows converts a slice of budget maps to table row strings.
func toBudgetRows(budgets []map[string]interface{}) [][]string {
	rows := make([][]string, len(budgets))
	for i, b := range budgets {
		currency := str(b, "currency")
		rows[i] = []string{
			str(b, "alertId"),
			str(b, "name"),
			formatCurrency(floatVal(b, "threshold"), currency),
			formatCurrency(floatVal(b, "currentValue"), currency),
			formatCurrency(floatVal(b, "remaining"), currency),
			fmt.Sprintf("%.1f%%", floatVal(b, "percentUsed")),
			str(b, "risk"),
		}
	}
	return rows
}
