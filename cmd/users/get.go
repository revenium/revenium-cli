package users

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a user by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a user by ID
  revenium users get user-123

  # Get a user as JSON
  revenium users get user-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var user map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/users/"+args[0], nil, &user); err != nil {
				return err
			}
			return renderUser(user)
		},
	}
}
