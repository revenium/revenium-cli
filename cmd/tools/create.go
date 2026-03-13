package tools

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
	var name, toolID, toolType, description, toolProvider string
	var enabled bool

	c := &cobra.Command{
		Use:   "create",
		Short:       "Create a new tool",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Create a tool with required fields
  revenium tools create --name "My Tool" --tool-id my-tool --tool-type MCP_SERVER

  # Create a tool with all fields
  revenium tools create --name "My Tool" --tool-id my-tool --tool-type MCP_SERVER --description "A test tool" --tool-provider acme --enabled=false`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name":     name,
				"toolId":   toolID,
				"toolType": toolType,
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

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "tool", "/v2/api/tools", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/tools", body, &result); err != nil {
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
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("tool-id")
	_ = c.MarkFlagRequired("tool-type")

	return c
}
