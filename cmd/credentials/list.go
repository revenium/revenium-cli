package credentials

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all provider credentials",
		Args:  cobra.NoArgs,
		Example: `  # List all credentials
  revenium credentials list

  # List credentials as JSON
  revenium credentials list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var credentials []map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/credentials", nil, &credentials); err != nil {
				return err
			}
			if len(credentials) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No credentials found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(credentials), credentials)
		},
	}
}
