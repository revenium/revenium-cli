package organizations

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newListCmd returns the `revenium organizations list` command.
// Endpoint: GET /v2/api/organizations (RESEARCH-LOCKED — `tenantId` is auto-injected
// as a query param by Client.Do from the configured client tenant). Renders the
// shared 3-col tableDef (ID / Name / Status) populated via toRows. Empty list
// emits the canonical empty-state phrase per D-04d for non-JSON or `[]` JSON. DoList
// handles HATEOAS `_embedded.organizationResourceList` unwrap and auto-pagination
// via cmd.AddListFlags (CF-12-19).
func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all organizations",
		Args:  cobra.NoArgs,
		Example: `  # List all organizations
  revenium organizations list

  # List organizations as JSON
  revenium organizations list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var orgs []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/organizations", cmd.ListOptsFromFlags(c), &orgs); err != nil {
				return err
			}
			if len(orgs) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No organizations found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(orgs), orgs)
		},
	}

	cmd.AddListFlags(c)
	return c
}
