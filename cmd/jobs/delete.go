package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

// init wires the delete subcommand onto the package-level Cmd parent.
// Multiple init() per package is core Go — Plan 01 wires list+get in jobs.go,
// each subsequent plan adds its own AddCommand call in its own file (D-21).
func init() {
	Cmd.AddCommand(newDeleteCmd())
}

// newDeleteCmd builds the `revenium jobs delete <agenticJobId>` subcommand.
//
// Wire diagram (D-12, D-16, D-17, D-20, D-22, D-24, D-25):
//
//	RunE:
//	  id        := args[0]                                  // user-supplied agenticJobId (D-24)
//	  path      := /v2/api/jobs/<url.PathEscape(id)>        // single-item path (D-22 + D-25)
//	  --dry-run -> dryrun.Render("delete", "job", path, nil) // D-17
//	  yes,_     := --yes flag
//	  ok,err    := resource.ConfirmDelete("job", id, yes, IsJSON()) // D-12
//	  DELETE     path                                       // D-20 — literal HTTP method
//	  !IsQuiet  -> "Deleted job <id>." to stdout            // pattern S8
//
// RESEARCH §Risk 4: the path MUST always include /{agenticJobId}. ExactArgs(1)
// makes a no-arg invocation impossible at the cobra layer; delete_test.go also
// asserts r.URL.Path != "/v2/api/jobs" as a defensive belt-and-suspenders guard.
func newDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:         "delete <agenticJobId>",
		Short:       "Delete a job",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a job (with confirmation)
  revenium jobs delete loan-app-12345

  # Delete without confirmation
  revenium jobs delete loan-app-12345 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			// Capture path ONCE so dry-run preview and real DELETE are byte-identical.
			path := fmt.Sprintf("/v2/api/jobs/%s", url.PathEscape(id))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "job", path, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("job", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted job %s.\n", id)
			}
			return nil
		},
	}

	return c
}
