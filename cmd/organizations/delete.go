package organizations

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

// newDeleteCmd builds the `revenium organizations delete <id>` subcommand.
//
// Wire diagram (CF-12-16, CF-12-17, CF-12-25):
//   - RunE pulls id from args[0] and builds path /v2/api/organizations/<escaped-id>.
//   - When the global --dry-run flag is set, the dry-run renderer fires BEFORE
//     the confirmation prompt (CF-12-17).
//   - Otherwise, the resource confirm-delete helper handles --yes skip and the
//     JSON / non-TTY auto-confirm path.
//   - The CLI then issues a DELETE on the wire and (per CF-12-25) prints the
//     success line "Deleted organization <id>." unless quiet mode is active.
func newDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete an organization",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete an organization (with confirmation prompt in TTY mode)
  revenium organizations delete org-123

  # Delete without confirmation
  revenium organizations delete org-123 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			// Capture path once so dry-run preview and real DELETE are byte-identical.
			path := fmt.Sprintf("/v2/api/organizations/%s", url.PathEscape(id))

			// CF-12-17: dry-run gate fires BEFORE confirmation prompt.
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "organization", path, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			// CF-12-25: ConfirmDelete handles --yes skip, JSON/non-TTY auto-confirm.
			ok, err := resource.ConfirmDelete("organization", id, yes, cmd.Output.IsJSON())
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
				fmt.Fprintf(c.OutOrStdout(), "Deleted organization %s.\n", id)
			}
			return nil
		},
	}

	return c
}
