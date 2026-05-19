package subscribers

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newLookupCmd() *cobra.Command {
	var email string

	c := &cobra.Command{
		Use:   "lookup",
		Short: "Look up a subscriber by email",
		Args:  cobra.NoArgs,
		Example: `  # Look up a subscriber by email
  revenium subscribers lookup --email user@example.com

  # As JSON
  revenium subscribers lookup --email user@example.com --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/subscribers/lookup-by-email?email=%s", url.QueryEscape(email))
			var subscriber map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &subscriber); err != nil {
				return err
			}
			return renderSubscriber(subscriber)
		},
	}
	c.Flags().StringVar(&email, "email", "", "Email address of the subscriber to look up")
	_ = c.MarkFlagRequired("email")
	return c
}
