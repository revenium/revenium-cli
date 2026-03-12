package tools

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var name, toolID, toolType, description, toolProvider string
	var enabled bool

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a tool",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a tool name
  revenium tools update tool-1 --name "Updated Tool"

  # Update type and provider
  revenium tools update tool-1 --tool-type MCP_SERVER --tool-provider acme`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("name") {
				body["name"] = name
			}
			if c.Flags().Changed("tool-id") {
				body["toolId"] = toolID
			}
			if c.Flags().Changed("tool-type") {
				body["toolType"] = toolType
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}
			if c.Flags().Changed("tool-provider") {
				body["toolProvider"] = toolProvider
			}
			if c.Flags().Changed("enabled") {
				body["enabled"] = enabled
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/tools/"+id, body, &result); err != nil {
				return err
			}
			return renderTool(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Tool name")
	c.Flags().StringVar(&toolID, "tool-id", "", "Tool identifier")
	c.Flags().StringVar(&toolType, "tool-type", "", "Tool type (e.g. MCP_SERVER)")
	c.Flags().StringVar(&description, "description", "", "Tool description")
	c.Flags().StringVar(&toolProvider, "tool-provider", "", "Tool provider")
	c.Flags().BoolVar(&enabled, "enabled", true, "Whether the tool is enabled")

	return c
}
