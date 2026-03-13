package tools

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var name, toolID, toolType, description, toolProvider string
	var enabled bool

	c := &cobra.Command{
		Use:   "update <id>",
		Short:       "Update a tool",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a tool name
  revenium tools update tool-1 --name "Updated Tool"

  # Update type and provider
  revenium tools update tool-1 --tool-type MCP_SERVER --tool-provider acme`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("name") {
				updates["name"] = name
			}
			if c.Flags().Changed("tool-id") {
				updates["toolId"] = toolID
			}
			if c.Flags().Changed("tool-type") {
				updates["toolType"] = toolType
			}
			if c.Flags().Changed("description") {
				updates["description"] = description
			}
			if c.Flags().Changed("tool-provider") {
				updates["toolProvider"] = toolProvider
			}
			if c.Flags().Changed("enabled") {
				updates["enabled"] = enabled
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "tool", "/v2/api/tools/"+id, updates)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/tools/"+id, updates, &result); err != nil {
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
