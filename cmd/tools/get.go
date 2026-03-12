package tools

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a tool by ID",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get a tool by ID
  revenium tools get tool-1

  # Get a tool as JSON
  revenium tools get tool-1 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var tool map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/tools/"+args[0], nil, &tool); err != nil {
				return err
			}
			return renderTool(tool)
		},
	}
}
