package sources

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
		Short:       "Delete a source",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a source (with confirmation)
  revenium sources delete abc-123

  # Delete without confirmation
  revenium sources delete abc-123 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "source", "/v2/api/sources/"+id, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("source", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", "/v2/api/sources/"+id, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted source %s.\n", id)
			}
			return nil
		},
	}

	return c
}
