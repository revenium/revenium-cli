package charts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all chart definitions",
		Args:  cobra.NoArgs,
		Example: `  # List all chart definitions
  revenium charts list

  # List chart definitions as JSON
  revenium charts list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var charts []map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/reports/chart-definitions", nil, &charts); err != nil {
				return err
			}
			if len(charts) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No charts found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(charts), charts)
		},
	}
}
