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
//	  filters/filtersChanged := resolveFilters(c, ...)              // PLAN D-01 — conflict before HTTP
//	  body      := only fields gated by c.Flags().Changed("name") etc  // D-07
//	  len(body)==0 -> fmt.Errorf("no fields specified to update")  // D-09 verbatim
//	  path      := /v2/api/ai/cost-controls/<url.PathEscape(id)>   // RESEARCH endpoint correction
//	  --dry-run -> dryrun.Render("update", "budget rule", path, body) // CF-12-17
//	  PATCH      path with body                                    // D-08 literal method
//	  renderRule(result)                                           // from budget_rules.go
//
// PLAN 260524-kvj adds three more PATCH-changeable fields (gated identically):
//   - --filter dim:op:val (repeatable) — composes filters array
//   - --filters-json '<JSON>'         — escape hatch (mutually exclusive with --filter, D-01)
//   - --notification-channel-id <id>  (repeatable) — composes notificationChannelIds array
//
// PATCH-replace semantics (PLAN D-04 locked): both filters and
// notificationChannelIds are REPLACED wholesale by the supplied slice — there
// is no per-element add/remove API. This is documented in the flag help text.
func newBudgetRulesUpdateCmd() *cobra.Command {
	var (
		name                   string
		filterFlags            []string
		filtersJSON            string
		notificationChannelIDs []string
	)

	c := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a budget rule (PATCH)",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a budget rule's display name
  revenium guardrails budget-rules update jR2kmLs --name "New name"

  # Replace the filters scoping this rule (PATCH replaces the entire array — there is no per-element add/remove)
  revenium guardrails budget-rules update jR2kmLs --filter MODEL:IS:gpt-4 --filter PROVIDER:IS:openai

  # Replace the notification channels (PATCH replaces the entire array)
  revenium guardrails budget-rules update jR2kmLs --notification-channel-id chan-1 --notification-channel-id chan-2`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			// PLAN D-01: resolve filters FIRST so the mutual-exclusion check
			// between --filter and --filters-json fires before the empty-body
			// check, before the dry-run gate, and before the HTTP call.
			filters, filtersChanged, err := resolveFilters(c, filterFlags, filtersJSON)
			if err != nil {
				return err
			}

			// D-07: only fields the user explicitly set land in the body.
			body := make(map[string]interface{})
			if c.Flags().Changed("name") {
				body["name"] = name
			}
			if filtersChanged {
				body["filters"] = filters
			}
			if c.Flags().Changed("notification-channel-id") {
				body["notificationChannelIds"] = notificationChannelIDs
			}

			// D-09: empty body returns the verbatim error BEFORE any HTTP call.
			// Now correctly fires only when NONE of name/filter/filters-json/
			// notification-channel-id were supplied.
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
	// PLAN 260524-kvj D-04: PATCH replaces these arrays wholesale — call that
	// out in the help text so users do not assume per-element add/remove.
	c.Flags().StringArrayVar(&filterFlags, "filter", nil, "Repeatable filter in dim:op:val form (e.g. --filter MODEL:IS:gpt-4). Known dimensions: AGENT, MODEL, PROVIDER, ORGANIZATION, CREDENTIAL, PRODUCT, SUBSCRIBER, TASK_TYPE. Known operators: IS, IS_NOT. Server validates values. (PATCH replaces the entire array — there is no per-element add/remove)")
	c.Flags().StringVar(&filtersJSON, "filters-json", "", "Alternative to --filter: full filters array as JSON, e.g. '[{\"dimension\":\"MODEL\",\"operator\":\"IS\",\"value\":\"gpt-4\"}]'. Mutually exclusive with --filter. (PATCH replaces the entire array)")
	c.Flags().StringArrayVar(&notificationChannelIDs, "notification-channel-id", nil, "Repeatable notification channel ID to attach to this rule (PATCH replaces the entire array — there is no per-element add/remove)")

	return c
}
