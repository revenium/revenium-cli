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

// renderRule renders a single budget rule as a single-row table (with optional
// trailing filters / notification-channels blocks) OR as JSON.
//
// In JSON mode (`--json`), cmd.Output.Render emits the full rule map verbatim;
// filters and notificationChannelIds round-trip "for free" through the map
// passthrough and we MUST NOT print the secondary blocks (they would
// contaminate the JSON output stream).
//
// In table mode, the 3-col ID/Name/Status table is rendered first, then
// (only when present and non-empty) a "Filters:" block and a
// "Notification channels:" block are appended. Absent or empty arrays stay
// silent so rules with no scope or no channels keep their original
// 3-col output unchanged.
func renderRule(rule map[string]interface{}) error {
	rows := [][]string{{
		str(rule, "id"),
		str(rule, "name"),
		boolStatus(rule, "enabled"),
	}}
	if err := cmd.Output.Render(tableDef, rows, rule); err != nil {
		return err
	}

	// PLAN 260524-kvj must-have: surface filters + notificationChannelIds in
	// non-JSON mode so users can SEE these fields. JSON mode skips this entire
	// block — Output.Render already wrote the full rule map.
	if cmd.Output.IsJSON() {
		return nil
	}

	w := cmd.Output.Writer()

	// Filters block — silent when absent, empty, or malformed.
	if filtersRaw, ok := rule["filters"].([]interface{}); ok && len(filtersRaw) > 0 {
		fmt.Fprintln(w, "Filters:")
		for _, item := range filtersRaw {
			fmap, ok := item.(map[string]interface{})
			if !ok {
				// Defensive: skip malformed server responses rather than panic.
				continue
			}
			// Print as "  dimension operator value" so the three triple parts
			// are individually visible to substring-style assertions and easy
			// for humans to read.
			fmt.Fprintf(w, "  %s %s %s\n",
				str(fmap, "dimension"),
				str(fmap, "operator"),
				str(fmap, "value"),
			)
		}
	}

	// Notification channels block — silent when absent or empty.
	if chansRaw, ok := rule["notificationChannelIds"].([]interface{}); ok && len(chansRaw) > 0 {
		fmt.Fprintln(w, "Notification channels:")
		for _, item := range chansRaw {
			fmt.Fprintf(w, "  %s\n", fmt.Sprint(item))
		}
	}

	return nil
}
