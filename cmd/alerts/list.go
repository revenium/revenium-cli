package alerts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all AI alerts",
		Args:  cobra.NoArgs,
		Example: `  # List all AI alerts
  revenium alerts list

  # List AI alerts as JSON
  revenium alerts list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var alerts []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/sources/ai/alert", cmd.ListOptsFromFlags(c), &alerts); err != nil {
				return err
			}
			if len(alerts) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No alerts found.")
				return nil
			}
			return cmd.Output.Render(alertTableDef, toAlertRows(alerts), alerts)
		},
	}

	cmd.AddListFlags(c)
	return c
}
