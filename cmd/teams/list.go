package teams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all teams",
		Args:  cobra.NoArgs,
		Example: `  # List all teams
  revenium teams list

  # List teams as JSON
  revenium teams list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var teams []map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/teams", nil, &teams); err != nil {
				return err
			}
			if len(teams) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No teams found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(teams), teams)
		},
	}
}
