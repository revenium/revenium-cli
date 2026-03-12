package users

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Args:  cobra.NoArgs,
		Example: `  # List all users
  revenium users list

  # List users as JSON
  revenium users list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var users []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/users", &users); err != nil {
				return err
			}
			if len(users) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No users found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(users), users)
		},
	}
}
