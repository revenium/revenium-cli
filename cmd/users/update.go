package users

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var (
		email             string
		firstName         string
		lastName          string
		roles             []string
		teamIDs           []string
		phoneNumber       string
		canViewPromptData bool
	)

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a user",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a user's email
  revenium users update user-123 --email new@example.com

  # Update a user's roles
  revenium users update user-123 --roles ROLE_ADMIN,ROLE_API_CONSUMER`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			// The API requires roles and teamIds on PUT but doesn't return them in GET.
			// If the user doesn't specify them, we must include defaults.
			if !c.Flags().Changed("roles") {
				roles = []string{"ROLE_API_CONSUMER"}
			}
			if !c.Flags().Changed("team-ids") {
				teamIDs = []string{cmd.APIClient.TeamID}
			}

			updates := make(map[string]interface{})
			updates["roles"] = roles
			updates["teamIds"] = teamIDs

			if c.Flags().Changed("email") {
				updates["email"] = email
			}
			if c.Flags().Changed("first-name") {
				updates["firstName"] = firstName
			}
			if c.Flags().Changed("last-name") {
				updates["lastName"] = lastName
			}
			if c.Flags().Changed("roles") {
				updates["roles"] = roles
			}
			if c.Flags().Changed("team-ids") {
				updates["teamIds"] = teamIDs
			}
			if c.Flags().Changed("phone-number") {
				updates["phoneNumber"] = phoneNumber
			}
			if c.Flags().Changed("can-view-prompt-data") {
				updates["canViewPromptData"] = canViewPromptData
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "user", "/v2/api/users/"+id, updates)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/users/"+id, updates, &result); err != nil {
				return err
			}
			return renderUser(result)
		},
	}

	c.Flags().StringVar(&email, "email", "", "User email address")
	c.Flags().StringVar(&firstName, "first-name", "", "User first name")
	c.Flags().StringVar(&lastName, "last-name", "", "User last name")
	c.Flags().StringSliceVar(&roles, "roles", nil, "User roles (comma-separated)")
	c.Flags().StringSliceVar(&teamIDs, "team-ids", nil, "Team IDs (comma-separated)")
	c.Flags().StringVar(&phoneNumber, "phone-number", "", "User phone number")
	c.Flags().BoolVar(&canViewPromptData, "can-view-prompt-data", false, "Whether user can view prompt data")

	return c
}
