package sources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all sources",
		Args:  cobra.NoArgs,
		Example: `  # List all sources
  revenium sources list

  # List sources as JSON
  revenium sources list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var sources []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/sources", cmd.ListOptsFromFlags(c), &sources); err != nil {
				return err
			}
			if len(sources) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No sources found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(sources), sources)
		},
	}

	cmd.AddListFlags(c)
	return c
}
