package meteringelements

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

// newDeleteCmd builds `revenium metering-elements delete <id>`.
//
// Deviation from cmd/organizations/delete.go (RESEARCH Assumption A2):
// DELETE /v2/api/metering-element-definitions/{id} returns a
// DeleteResponse_Read body (id/message/created/updated), unlike
// organizations' DELETE which returns no body — decode into &result
// instead of passing nil, and surface it in JSON mode.
func newDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a metering element definition",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a metering element definition (with confirmation prompt in TTY mode)
  revenium metering-elements delete me-123

  # Delete without confirmation
  revenium metering-elements delete me-123 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			// Capture path once so dry-run preview and real DELETE are byte-identical.
			path := fmt.Sprintf("/v2/api/metering-element-definitions/%s", url.PathEscape(id))

			// Dry-run gate fires BEFORE the confirmation prompt.
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "metering element", path, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("metering element", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, &result); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted metering element %s.\n", id)
			}
			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(result)
			}
			return nil
		},
	}

	return c
}
