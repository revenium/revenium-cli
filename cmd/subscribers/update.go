package subscribers

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var email, firstName, lastName string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a subscriber",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a subscriber email
  revenium subscribers update abc-123 --email new@example.com

  # Update multiple fields
  revenium subscribers update abc-123 --email new@example.com --first-name Jane --last-name Smith`,
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

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/subscribers/"+id, body, &result); err != nil {
				return err
			}
			return renderSubscriber(result)
		},
	}

	c.Flags().StringVar(&email, "email", "", "Subscriber email address")
	c.Flags().StringVar(&firstName, "first-name", "", "Subscriber first name")
	c.Flags().StringVar(&lastName, "last-name", "", "Subscriber last name")

	return c
}
