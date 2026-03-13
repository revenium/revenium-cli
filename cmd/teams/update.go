package teams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var name, description string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a team",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a team name
  revenium teams update team-123 --name "New Name"

  # Update multiple fields
  revenium teams update team-123 --name "New Name" --description "New description"`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("name") {
				updates["name"] = name
			}
			if c.Flags().Changed("description") {
				updates["description"] = description
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "team", "/v2/api/teams/"+id, updates)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/teams/"+id, updates, &result); err != nil {
				return err
			}
			return renderTeam(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Team name")
	c.Flags().StringVar(&description, "description", "", "Team description")

	return c
}
