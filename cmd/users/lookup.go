package users

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
		Short: "Look up a user by email",
		Args:  cobra.NoArgs,
		Example: `  # Look up a user by email
  revenium users lookup --email jane@example.com

  # As JSON
  revenium users lookup --email jane@example.com --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/users/lookup-by-email?email=%s", url.QueryEscape(email))
			var user map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &user); err != nil {
				return err
			}
			return renderUser(user)
		},
	}

	c.Flags().StringVar(&email, "email", "", "Email address of the user to look up")
	_ = c.MarkFlagRequired("email")

	return c
}
