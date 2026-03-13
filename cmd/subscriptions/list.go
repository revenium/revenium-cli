package subscriptions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all subscriptions",
		Args:  cobra.NoArgs,
		Example: `  # List all subscriptions
  revenium subscriptions list

  # List subscriptions as JSON
  revenium subscriptions list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var subs []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/subscriptions", cmd.ListOptsFromFlags(c), &subs); err != nil {
				return err
			}
			if len(subs) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No subscriptions found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(subs), subs)
		},
	}

	cmd.AddListFlags(c)
	return c
}
