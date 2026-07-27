package tenants

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newGetCmd returns the `revenium tenants get <id>` command.
// Endpoint: GET /v2/api/tenants/{url.PathEscape(id)}. Renders the object via
// the package-level renderTenant helper. Args validation uses
// cobra.MatchAll(ExactArgs(1), ValidResourceID) (T-06-06 mitigation).
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a tenant by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a tenant by ID
  revenium tenants get ten-123

  # Get a tenant as JSON
  revenium tenants get ten-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/tenants/%s", url.PathEscape(args[0]))
			var tenant map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &tenant); err != nil {
				return err
			}
			return renderTenant(tenant)
		},
	}
}
