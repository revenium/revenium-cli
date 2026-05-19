package guardrails

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// enforcementRulesCmd is the parent enforcement-rules subcommand under guardrails.
var enforcementRulesCmd = &cobra.Command{
	Use:   "enforcement-rules",
	Short: "View compiled enforcement rules per team (read-only)",
	Example: `  # View compiled enforcement rules for a team
  revenium guardrails enforcement-rules get team-123

  # View as JSON
  revenium guardrails enforcement-rules get team-123 --json`,
}

// initEnforcementRules registers verbs onto enforcementRulesCmd. Called from guardrails.go init().
func initEnforcementRules() {
	enforcementRulesCmd.AddCommand(newEnforcementRulesGetCmd())
}

// enforcementRulesTableDef defines the 4-column layout per RESEARCH D-11: Rule ID / Name / Action / Mode.
var enforcementRulesTableDef = output.TableDef{
	Headers:      []string{"Rule ID", "Name", "Action", "Mode"},
	StatusColumn: 2, // Action — anticipates future WARN/LOG tokens
}

// renderEnforcementRules renders a compiled-enforcement-rules payload as a multi-row table
// preceded by a "Compiled at: <ts>" metadata header (non-JSON only). RESEARCH §"Render code sketch".
func renderEnforcementRules(out io.Writer, payload map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.RenderJSON(payload)
	}
	if compiledAt := str(payload, "compiledAt"); compiledAt != "" {
		fmt.Fprintf(out, "Compiled at: %s\n\n", compiledAt)
	}
	rulesRaw, _ := payload["rules"].([]interface{})
	if len(rulesRaw) == 0 {
		fmt.Fprintln(out, "No enforcement rules compiled.")
		return nil
	}
	rules := make([]map[string]interface{}, 0, len(rulesRaw))
	for _, r := range rulesRaw {
		if m, ok := r.(map[string]interface{}); ok {
			rules = append(rules, m)
		}
	}
	rows := make([][]string, len(rules))
	for i, r := range rules {
		mode := "enforce"
		if b, ok := r["shadowMode"].(bool); ok && b {
			mode = "shadow"
		}
		rows[i] = []string{
			str(r, "ruleId"),
			str(r, "name"),
			str(r, "action"),
			mode,
		}
	}
	return cmd.Output.Render(enforcementRulesTableDef, rows, payload)
}
