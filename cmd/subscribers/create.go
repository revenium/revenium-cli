package subscribers

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var email, firstName, lastName string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new subscriber",
		Example: `  # Create a subscriber with email only
  revenium subscribers create --email user@example.com

  # Create a subscriber with all fields
  revenium subscribers create --email user@example.com --first-name John --last-name Doe`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"email": email,
			}
			if c.Flags().Changed("first-name") {
				body["firstName"] = firstName
			}
			if c.Flags().Changed("last-name") {
				body["lastName"] = lastName
			}
			// The API requires organizationIds; default to the configured team ID
			if cmd.APIClient.TeamID != "" {
				body["organizationIds"] = []string{cmd.APIClient.TeamID}
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/subscribers", body, &result); err != nil {
				return err
			}
			return renderSubscriber(result)
		},
	}

	c.Flags().StringVar(&email, "email", "", "Subscriber email address")
	c.Flags().StringVar(&firstName, "first-name", "", "Subscriber first name")
	c.Flags().StringVar(&lastName, "last-name", "", "Subscriber last name")
	_ = c.MarkFlagRequired("email")

	return c
}
