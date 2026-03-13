package anomalies

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
		Short: "Delete an anomaly detection rule",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete an anomaly rule (with confirmation)
  revenium anomalies delete anom-123

  # Delete without confirmation
  revenium anomalies delete anom-123 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "anomaly", "/v2/api/sources/ai/anomaly/"+id, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("anomaly", id, yes, cmd.Output.IsJSON())
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
				fmt.Fprintf(c.OutOrStdout(), "Deleted anomaly %s.\n", id)
			}
			return nil
		},
	}

	return c
}
