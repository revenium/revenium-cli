package tools

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
		Short:       "Delete a tool",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a tool (with confirmation)
  revenium tools delete tool-1

  # Delete without confirmation
  revenium tools delete tool-1 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "tool", "/v2/api/tools/"+id, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("tool", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", "/v2/api/tools/"+id, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted tool %s.\n", id)
			}
			return nil
		},
	}

	return c
}
