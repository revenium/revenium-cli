package users

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
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
		Use:   "create",
		Short: "Create a new user",
		Example: `  # Create a user with required fields
  revenium users create --email jane@example.com --first-name Jane --last-name Doe --roles ROLE_API_CONSUMER --team-ids team-1

  # Create a user with optional fields
  revenium users create --email jane@example.com --first-name Jane --last-name Doe --roles ROLE_API_CONSUMER --team-ids team-1 --phone-number 555-1234 --can-view-prompt-data`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"email":     email,
				"firstName": firstName,
				"lastName":  lastName,
				"roles":     roles,
				"teamIds":   teamIDs,
			}
			if c.Flags().Changed("phone-number") {
				body["phoneNumber"] = phoneNumber
			}
			if c.Flags().Changed("can-view-prompt-data") {
				body["canViewPromptData"] = canViewPromptData
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "user", "/v2/api/users", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/users", body, &result); err != nil {
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

	_ = c.MarkFlagRequired("email")
	_ = c.MarkFlagRequired("first-name")
	_ = c.MarkFlagRequired("last-name")
	_ = c.MarkFlagRequired("roles")
	_ = c.MarkFlagRequired("team-ids")

	return c
}
