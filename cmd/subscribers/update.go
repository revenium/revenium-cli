package subscribers

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var email, firstName, lastName, subscriberID string
	var organizationIDs []string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a subscriber",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a subscriber email
  revenium subscribers update abc-123 --email new@example.com

  # Update multiple fields
  revenium subscribers update abc-123 --email new@example.com --first-name Jane --last-name Smith

  # Set subscriber ID
  revenium subscribers update abc-123 --subscriber-id sub-custom-456

  # Update organization membership
  revenium subscribers update abc-123 --organization-ids org-1,org-2`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("email") {
				updates["email"] = email
			}
			if c.Flags().Changed("first-name") {
				updates["firstName"] = firstName
			}
			if c.Flags().Changed("last-name") {
				updates["lastName"] = lastName
			}
			if c.Flags().Changed("subscriber-id") {
				updates["subscriberId"] = subscriberID
			}
			if c.Flags().Changed("organization-ids") {
				updates["organizationIds"] = organizationIDs
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "subscriber", "/v2/api/subscribers/"+id, updates)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/subscribers/"+id, updates, &result); err != nil {
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

	return c
}
