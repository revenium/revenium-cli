// CRITICAL: budget-rules update uses HTTP PATCH (true partial update). Per
// RESEARCH "PATCH Verification" and D-08, /v2/api/ai/cost-controls/{id}
// does not accept PUT — the GET+merge+PUT helper is forbidden here. The
// HTTP call MUST be cmd.APIClient.Do(c.Context(), "PATCH", path, body, &result).
package guardrails

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newBudgetRulesUpdateCmd builds the `revenium guardrails budget-rules update <id>` subcommand.
//
// Wire diagram (D-07, D-08, D-09, CF-12-16, CF-12-17, CF-12-20):
//
//	RunE:
//	  id        := args[0]                                          // user-supplied id
//	  body      := only fields gated by c.Flags().Changed("name")  // D-07
//	  len(body)==0 -> fmt.Errorf("no fields specified to update")  // D-09 verbatim
//	  path      := /v2/api/ai/cost-controls/<url.PathEscape(id)>   // RESEARCH endpoint correction
//	  --dry-run -> dryrun.Render("update", "budget rule", path, body) // CF-12-17
//	  PATCH      path with body                                    // D-08 literal method
//	  renderRule(result)                                           // from budget_rules.go
func newBudgetRulesUpdateCmd() *cobra.Command {
	var name string

	c := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a budget rule (PATCH)",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a budget rule's display name
  revenium guardrails budget-rules update jR2kmLs --name "New name"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			// D-07: only fields the user explicitly set land in the body.
			body := make(map[string]interface{})
			if c.Flags().Changed("name") {
				body["name"] = name
			}

			// D-09: empty body returns the verbatim error BEFORE any HTTP call.
			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			// Defensive url.PathEscape because id is user-supplied (CF-12-16).
			path := fmt.Sprintf("/v2/api/ai/cost-controls/%s", url.PathEscape(id))

			// CF-12-17: dry-run gate fires BEFORE the HTTP call.
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "budget rule", path, body)
			}

			// D-08 LOAD-BEARING: TRUE HTTP PATCH — literal method string.
			// NOT the GET+merge+PUT helper. NOT PUT.
			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PATCH", path, body, &result); err != nil {
				return err
			}
			return renderRule(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Budget rule display name")

	return c
}
