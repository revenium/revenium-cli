package subscribers

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
	var email, firstName, lastName, subscriberID string
	var organizationIDs []string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new subscriber",
		Example: `  # Create a subscriber with email only
  revenium subscribers create --email user@example.com

  # Create a subscriber with all fields
  revenium subscribers create --email user@example.com --first-name John --last-name Doe

  # Create a subscriber with a custom subscriber ID
  revenium subscribers create --email user@example.com --subscriber-id sub-custom-123

  # Create a subscriber in specific organizations
  revenium subscribers create --email user@example.com --organization-ids org-1,org-2`,
		Annotations: map[string]string{"mutating": "true"},
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
			if c.Flags().Changed("subscriber-id") {
				body["subscriberId"] = subscriberID
			}
			// Use explicit --organization-ids if provided, otherwise default to configured team ID
			if c.Flags().Changed("organization-ids") {
				body["organizationIds"] = organizationIDs
			} else if cmd.APIClient.TeamID != "" {
				body["organizationIds"] = []string{cmd.APIClient.TeamID}
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "subscriber", "/v2/api/subscribers", body)
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
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Custom subscriber identifier")
	c.Flags().StringSliceVar(&organizationIDs, "organization-ids", nil, "Comma-separated list of organization IDs")
	_ = c.MarkFlagRequired("email")

	return c
}
