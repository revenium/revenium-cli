package teams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var name, description string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a team",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a team name
  revenium teams update team-123 --name "New Name"

  # Update multiple fields
  revenium teams update team-123 --name "New Name" --description "New description"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("name") {
				body["name"] = name
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/teams/"+id, body, &result); err != nil {
				return err
			}
			return renderTeam(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Team name")
	c.Flags().StringVar(&description, "description", "", "Team description")

	return c
}
