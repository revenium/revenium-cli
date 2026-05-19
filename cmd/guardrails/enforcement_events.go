package guardrails

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/output"
)

// enforcementEventsCmd is the parent enforcement-events subcommand under guardrails.
var enforcementEventsCmd = &cobra.Command{
	Use:   "enforcement-events",
	Short: "View enforcement event audit trail (read-only)",
	Example: `  # List recent enforcement events
  revenium guardrails enforcement-events list

  # List events since a timestamp
  revenium guardrails enforcement-events list --since 2026-05-01T00:00:00Z

  # Filter events by rule
  revenium guardrails enforcement-events list --rule-id jR2kmLs`,
}

// initEnforcementEvents registers verbs onto enforcementEventsCmd. Called from guardrails.go init().
func initEnforcementEvents() {
	enforcementEventsCmd.AddCommand(newEnforcementEventsListCmd())
}

// enforcementEventsTableDef defines the 5-column layout per RESEARCH D-12: Rule / Action / Metric / Used / Time.
var enforcementEventsTableDef = output.TableDef{
	Headers:      []string{"Rule", "Action", "Metric", "Used", "Time"},
	StatusColumn: 1, // Action
}

// eventTime falls through candidate field names because OAS doesn't name a timestamp field
// (RESEARCH §"EnforcementEvents Columns"). Returns the first non-empty match.
func eventTime(m map[string]interface{}) string {
	for _, k := range []string{"created", "eventTime", "timestamp", "occurredAt"} {
		if v := str(m, k); v != "" {
			return v
		}
	}
	return ""
}

// toEnforcementEventRows builds 5-col rows; "Used" composes "<currentValue>/<threshold>".
func toEnforcementEventRows(events []map[string]interface{}) [][]string {
	rows := make([][]string, len(events))
	for i, e := range events {
		rows[i] = []string{
			str(e, "ruleName"),
			str(e, "action"),
			str(e, "metricType"),
			fmt.Sprintf("%v/%v", e["currentValue"], e["threshold"]),
			eventTime(e),
		}
	}
	return rows
}
