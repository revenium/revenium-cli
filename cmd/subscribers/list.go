package subscribers

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all subscribers",
		Args:  cobra.NoArgs,
		Example: `  # List all subscribers
  revenium subscribers list

  # List subscribers as JSON
  revenium subscribers list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var subscribers []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/subscribers", cmd.ListOptsFromFlags(c), &subscribers); err != nil {
				return err
			}
			if len(subscribers) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No subscribers found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(subscribers), subscribers)
		},
	}

	cmd.AddListFlags(c)
	return c
}
