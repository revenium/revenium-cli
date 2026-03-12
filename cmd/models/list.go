package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all AI models",
		Args:  cobra.NoArgs,
		Example: `  # List all AI models
  revenium models list

  # List models as JSON
  revenium models list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var models []map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/ai/models", nil, &models); err != nil {
				return err
			}
			if len(models) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No models found.")
				return nil
			}
			return cmd.Output.Render(modelTableDef, toModelRows(models), models)
		},
	}
}
