package users

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var (
		email            string
		firstName        string
		lastName         string
		roles            []string
		teamIDs          []string
		phoneNumber      string
		canViewPromptData bool
	)

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a user",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a user's email
  revenium users update user-123 --email new@example.com

  # Update a user's roles
  revenium users update user-123 --roles ROLE_ADMIN,ROLE_API_CONSUMER`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("email") {
				body["email"] = email
			}
			if c.Flags().Changed("first-name") {
				body["firstName"] = firstName
			}
			if c.Flags().Changed("last-name") {
				body["lastName"] = lastName
			}
			if c.Flags().Changed("roles") {
				body["roles"] = roles
			}
			if c.Flags().Changed("team-ids") {
				body["teamIds"] = teamIDs
			}
			if c.Flags().Changed("phone-number") {
				body["phoneNumber"] = phoneNumber
			}
			if c.Flags().Changed("can-view-prompt-data") {
				body["canViewPromptData"] = canViewPromptData
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/users/"+id, body, &result); err != nil {
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
