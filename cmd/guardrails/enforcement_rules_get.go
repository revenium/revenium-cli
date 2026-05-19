package guardrails

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newEnforcementRulesGetCmd returns the `revenium guardrails enforcement-rules get <teamId>` verb.
// GETs /v2/api/ai/enforcement-rules/{teamId} and renders the EnforcementRulesPayload as a
// multi-row table preceded by a "Compiled at: <ts>" metadata header (RESEARCH D-11).
// Read-only per REQUIREMENTS.md Out of Scope — no dry-run / confirmation flow.
func newEnforcementRulesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <teamId>",
		Short: "Get compiled enforcement rules for a team",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # View compiled enforcement rules for a team
  revenium guardrails enforcement-rules get team-123

  # View as JSON
  revenium guardrails enforcement-rules get team-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/ai/enforcement-rules/%s", url.PathEscape(args[0]))
			var payload map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &payload); err != nil {
				return err
			}
			return renderEnforcementRules(c.OutOrStdout(), payload)
		},
	}
}
