package guardrails

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newBudgetRulesGetCmd returns the `revenium guardrails budget-rules get <id>` command.
// Endpoint: GET /v2/api/ai/cost-controls/{id} (RESEARCH-corrected base). The single-row
// 3-col render (D-05) is delegated to the package-level renderRule helper from
// budget_rules.go. Args validation uses cobra.MatchAll(ExactArgs(1), ValidResourceID)
// per CF-12-16. 404 / other errors flow through internal/api/client.go default mapping
// (CF-13-18).
func newBudgetRulesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a budget rule by id",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a budget rule by id
  revenium guardrails budget-rules get jR2kmLs

  # Get a budget rule as JSON
  revenium guardrails budget-rules get jR2kmLs --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/ai/cost-controls/%s", url.PathEscape(args[0]))
			var rule map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &rule); err != nil {
				return err
			}
			return renderRule(rule)
		},
	}
}
