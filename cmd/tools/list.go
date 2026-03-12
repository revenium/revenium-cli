package tools

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tools",
		Args:  cobra.NoArgs,
		Example: `  # List all tools
  revenium tools list

  # List tools as JSON
  revenium tools list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var tools []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/tools", &tools); err != nil {
				return err
			}
			if len(tools) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No tools found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(tools), tools)
		},
	}
}
