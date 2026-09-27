// Package guardrails implements the guardrails command group (budget-rules CRUD + enforcement-rules read + enforcement-events list).
package guardrails

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent guardrails command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "guardrails",
	Short: "Manage budget rules and enforcement state",
	Example: `  # List budget rules
  revenium guardrails budget-rules list

  # View compiled enforcement rules for a team
  revenium guardrails enforcement-rules get team-123

  # List recent enforcement events
  revenium guardrails enforcement-events list`,
}

func init() {
	Cmd.AddCommand(budgetRulesCmd)
	initBudgetRules()
	Cmd.AddCommand(enforcementRulesCmd)
	initEnforcementRules()
	Cmd.AddCommand(enforcementEventsCmd)
	initEnforcementEvents()
	// org-unit-group-preview is a LEAF, not a noun-group, so it needs no
	// initX() indirection. It sits flat alongside the three sub-parents rather
	// than under budget-rules: budget-rules is a CRUD noun-group, and this is
	// not a budget rule — it is the question you ask before creating one
	// (D-31-22).
	Cmd.AddCommand(newOrgUnitGroupPreviewCmd())
}
