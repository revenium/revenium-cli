package teams

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
	var name, description, parentID string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new team",
		Example: `  # Create a team with name only
  revenium teams create --name "Engineering"

  # Create a team with description
  revenium teams create --name "Engineering" --description "Engineering team"

  # Create a child team under a parent
  revenium teams create --name "Frontend" --parent-id parent-team-123`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name": name,
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}
			if c.Flags().Changed("parent-id") {
				body["parentId"] = parentID
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "team", "/v2/api/teams", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreateWithOwner(c.Context(), "/v2/api/teams", body, &result); err != nil {
				return err
			}
			return renderTeam(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Team name")
	c.Flags().StringVar(&description, "description", "", "Team description")
	c.Flags().StringVar(&parentID, "parent-id", "", "Parent team ID")
	_ = c.MarkFlagRequired("name")

	return c
}
