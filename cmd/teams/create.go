package teams

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var name, description string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new team",
		Example: `  # Create a team with name only
  revenium teams create --name "Engineering"

  # Create a team with description
  revenium teams create --name "Engineering" --description "Engineering team"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name": name,
			}
			if c.Flags().Changed("description") {
				body["description"] = description
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
	_ = c.MarkFlagRequired("name")

	return c
}
