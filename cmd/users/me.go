package users

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// meCmd is the parent for `users me` and its four nested sub-lists
// (credentials, invoices, subscriptions, period-charges — D-06 nested-under-me
// shape). Declared as a package-level var (rather than constructed inline in
// init()) so the sibling files me_credentials.go/me_invoices.go/
// me_subscriptions.go/me_period_charges.go can each register their own
// subcommand onto it from their own init(), avoiding file-ordering issues —
// Go initializes all package-level vars before any init() runs, so this is
// safe regardless of file compile order (same pattern as
// cmd/teams/prompt_capture.go's initPromptCapture()).
var meCmd = newMeCmd()

func newMeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "me",
		Short: "Show the current authenticated user",
		Args:  cobra.NoArgs,
		Example: `  # Show the current authenticated user
  revenium users me

  # As JSON
  revenium users me --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var user map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/users/me", nil, &user); err != nil {
				return err
			}
			return renderUser(user)
		},
	}
	return c
}

// currentUserID resolves the authenticated user's id via GET /v2/api/users/me.
// Shared by the four `users me <sublist>` commands, each of which must do this
// two-call resolution first — there is no literal /users/me/<sublist> path in
// the spec (05-RESEARCH RES-05).
func currentUserID(c *cobra.Command) (string, error) {
	var user map[string]interface{}
	if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/users/me", nil, &user); err != nil {
		return "", err
	}
	id, _ := user["id"].(string)
	if id == "" {
		return "", fmt.Errorf("could not resolve current user id from /v2/api/users/me")
	}
	return id, nil
}
