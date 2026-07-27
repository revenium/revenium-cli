package meteringelements

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newUpdateCmd builds `revenium metering-elements update <id> ...` (PUT via
// GET-merge-PUT semantics, cmd.APIClient.DoUpdate).
//
// T-06-13 mitigation: if --type is supplied, validateType runs before the
// no-fields guard / dry-run check / HTTP call.
func newUpdateCmd() *cobra.Command {
	var name, elementType, description string

	c := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a metering element definition (PUT)",
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Rename a metering element definition
  revenium metering-elements update me-123 --name "New Name"

  # Change the type
  revenium metering-elements update me-123 --type STRING`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("name") {
				updates["name"] = name
			}
			if c.Flags().Changed("type") {
				if err := validateType(elementType); err != nil {
					return err
				}
				updates["type"] = elementType
			}
			if c.Flags().Changed("description") {
				updates["description"] = description
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			path := fmt.Sprintf("/v2/api/metering-element-definitions/%s", url.PathEscape(id))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "metering element", path, updates)
			}

			var result map[string]interface{}
			// GET + merge + PUT semantics.
			if err := cmd.APIClient.DoUpdate(c.Context(), path, updates, &result); err != nil {
				return err
			}
			return renderElement(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Metering element name")
	c.Flags().StringVar(&elementType, "type", "", "Metering element type: STRING or NUMBER")
	c.Flags().StringVar(&description, "description", "", "Metering element description")

	return c
}
