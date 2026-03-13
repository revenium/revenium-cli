package teams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

func newDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a team",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a team (with confirmation)
  revenium teams delete team-123

  # Delete without confirmation
  revenium teams delete team-123 --yes`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "team", "/v2/api/teams/"+id, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("team", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", "/v2/api/teams/"+id, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted team %s.\n", id)
			}
			return nil
		},
	}

	return c
}
