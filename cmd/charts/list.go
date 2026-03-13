package charts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all chart definitions",
		Args:  cobra.NoArgs,
		Example: `  # List all chart definitions
  revenium charts list

  # List chart definitions as JSON
  revenium charts list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var charts []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/reports/chart-definitions", cmd.ListOptsFromFlags(c), &charts); err != nil {
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

	cmd.AddListFlags(c)
	return c
}
