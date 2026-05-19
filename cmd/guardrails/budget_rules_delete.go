package guardrails

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

// newBudgetRulesDeleteCmd builds the `revenium guardrails budget-rules delete <id>` subcommand.
//
// Wire diagram (CF-12-16, CF-12-17, CF-12-25):
//
//	RunE:
//	  id        := args[0]                                              // user-supplied id
//	  path      := /v2/api/ai/cost-controls/<url.PathEscape(id)>        // RESEARCH endpoint correction
//	  --dry-run -> dryrun.Render("delete", "budget rule", path, nil)    // dry-run BEFORE confirm
//	  yes,_     := --yes flag
//	  ok,err    := resource.ConfirmDelete("budget rule", id, yes, IsJSON())
//	  DELETE     path                                                    // literal HTTP method
//	  !IsQuiet  -> "Deleted budget rule <id>." to stdout                // CF-12-25
func newBudgetRulesDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a budget rule",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a budget rule (with confirmation prompt in TTY mode)
  revenium guardrails budget-rules delete jR2kmLs

  # Delete without confirmation
  revenium guardrails budget-rules delete jR2kmLs --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			// Capture path once so dry-run preview and real DELETE are byte-identical.
			path := fmt.Sprintf("/v2/api/ai/cost-controls/%s", url.PathEscape(id))

			// CF-12-17: dry-run gate fires BEFORE confirmation prompt.
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "budget rule", path, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			// CF-12-25: ConfirmDelete handles --yes skip, JSON/non-TTY auto-confirm.
			ok, err := resource.ConfirmDelete("budget rule", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, nil); err != nil {
				return err
			}

			// CF-12-25: success line suppressed in quiet mode.
			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted budget rule %s.\n", id)
			}
			return nil
		},
	}

	return c
}
