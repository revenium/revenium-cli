package teams

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a team by ID",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get a team by ID
  revenium teams get team-123

  # Get a team as JSON
  revenium teams get team-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var team map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/teams/"+args[0], nil, &team); err != nil {
				return err
			}
			return renderTeam(team)
		},
	}
}
