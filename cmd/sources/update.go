package sources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var name, typ, description string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a source",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a source name
  revenium sources update abc-123 --name "New Name"

  # Update multiple fields
  revenium sources update abc-123 --name "New Name" --type AI --description "Updated"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("name") {
				updates["name"] = name
			}
			if c.Flags().Changed("type") {
				updates["type"] = typ
			}
			if c.Flags().Changed("description") {
				updates["description"] = description
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/sources/"+id, updates, &result); err != nil {
				return err
			}
			return renderSource(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Source name")
	c.Flags().StringVar(&typ, "type", "", "Source type (e.g., API, AI)")
	c.Flags().StringVar(&description, "description", "", "Source description")

	return c
}
