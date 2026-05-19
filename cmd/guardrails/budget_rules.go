package guardrails

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// budgetRulesCmd is the parent budget-rules subcommand under guardrails.
var budgetRulesCmd = &cobra.Command{
	Use:   "budget-rules",
	Short: "Manage cost-control budget rules",
	Example: `  # List all budget rules
  revenium guardrails budget-rules list

  # Get a budget rule
  revenium guardrails budget-rules get <id>

  # Create a budget rule
  revenium guardrails budget-rules create --name "Q3 OpenAI Budget" --metric-type TOTAL_COST --window-type MONTHLY --action BLOCK --group-by MODEL --warn-threshold 800 --hard-limit 1000 --description "Caps monthly OpenAI spend"

  # Update a budget rule
  revenium guardrails budget-rules update <id> --name "New Name"

  # Delete a budget rule
  revenium guardrails budget-rules delete <id>`,
}

// initBudgetRules registers verbs onto budgetRulesCmd. Called from guardrails.go init().
func initBudgetRules() {
	budgetRulesCmd.AddCommand(newBudgetRulesListCmd())
	budgetRulesCmd.AddCommand(newBudgetRulesGetCmd())
	budgetRulesCmd.AddCommand(newBudgetRulesCreateCmd())
	budgetRulesCmd.AddCommand(newBudgetRulesUpdateCmd())
	budgetRulesCmd.AddCommand(newBudgetRulesDeleteCmd())
}

// tableDef defines the 3-column table layout (D-02): ID / Name / Status.
// Column 2 is sourced from `enabled` (bool → "active"/"inactive") per RESEARCH D-03.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Status"},
	StatusColumn: 2,
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// boolStatus renders a bool field as "active"/"inactive" (RESEARCH D-03).
// Returns "" for missing/nil values; falls back to fmt.Sprint for non-bool types.
func boolStatus(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if b, ok := v.(bool); ok {
		if b {
			return "active"
		}
		return "inactive"
	}
	return fmt.Sprint(v)
}

// toRows converts a slice of budget-rule maps to 3-col table row strings.
func toRows(rules []map[string]interface{}) [][]string {
	rows := make([][]string, len(rules))
	for i, r := range rules {
		rows[i] = []string{
			str(r, "id"),
			str(r, "name"),
			boolStatus(r, "enabled"),
		}
	}
	return rows
}

// renderRule renders a single budget rule as a single-row table or JSON.
func renderRule(rule map[string]interface{}) error {
	rows := [][]string{{
		str(rule, "id"),
		str(rule, "name"),
		boolStatus(rule, "enabled"),
	}}
	return cmd.Output.Render(tableDef, rows, rule)
}
