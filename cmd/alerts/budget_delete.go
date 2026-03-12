package alerts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/resource"
)

func newBudgetDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete <anomaly-id>",
		Short: "Delete a budget alert",
		Args:  cobra.ExactArgs(1),
		Example: `  # Delete a budget alert (with confirmation)
  revenium alerts budget delete anom-123

  # Delete without confirmation
  revenium alerts budget delete anom-123 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("budget alert", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", "/v2/api/sources/ai/anomaly/"+id, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted budget alert %s.\n", id)
			}
			return nil
		},
	}

	return c
}
