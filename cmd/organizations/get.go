package organizations

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newGetCmd returns the `revenium organizations get <id>` command.
// Endpoint: GET /v2/api/organizations/{url.PathEscape(id)} (RESEARCH-LOCKED).
// The single-row 3-col render (D-04c) is delegated to the package-level renderOrg
// helper from organizations.go. Args validation uses cobra.MatchAll(ExactArgs(1),
// ValidResourceID) per CF-12-16. 404 / other errors flow through internal/api
// default mapping (CF-13-18).
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an organization by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get an organization by ID
  revenium organizations get org-123

  # Get an organization as JSON
  revenium organizations get org-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/organizations/%s", url.PathEscape(args[0]))
			var org map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &org); err != nil {
				return err
			}
			return renderOrg(org)
		},
	}
}
