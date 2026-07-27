package meteringelements

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newListCmd returns the `revenium metering-elements list` command.
// Endpoint: GET /v2/api/metering-element-definitions. DoList handles the
// HATEOAS `_embedded.meteringElementDefinitionResourceList` unwrap
// (verified embedded key name, RESEARCH Code Examples) and auto-pagination
// via cmd.AddListFlags. Empty list emits the canonical empty-state phrase.
func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all metering element definitions",
		Args:  cobra.NoArgs,
		Example: `  # List all metering element definitions
  revenium metering-elements list

  # List metering element definitions as JSON
  revenium metering-elements list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var elements []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/metering-element-definitions", cmd.ListOptsFromFlags(c), &elements); err != nil {
				return err
			}
			if len(elements) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No metering element definitions found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(elements), elements)
		},
	}

	cmd.AddListFlags(c)
	return c
}
