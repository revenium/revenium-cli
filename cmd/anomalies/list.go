package anomalies

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all anomaly detection rules",
		Args:  cobra.NoArgs,
		Example: `  # List all anomaly detection rules
  revenium anomalies list

  # List anomaly rules as JSON
  revenium anomalies list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var anomalies []map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/ai/anomaly", nil, &anomalies); err != nil {
				return err
			}
			if len(anomalies) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No anomalies found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(anomalies), anomalies)
		},
	}
}
