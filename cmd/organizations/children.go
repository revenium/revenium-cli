package organizations

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newChildrenCmd returns the `revenium organizations children <parentId>` command.
// Endpoint: GET /v2/api/organizations/parent/{parentId} (RESEARCH Pitfall 1 — the
// literal `parent/` segment is mandatory; `/{id}/children` returns 404). Renders
// the shared 3-col tableDef (ID / Name / Status) populated via toRows; empty list
// emits the no-children-found empty-state phrase (D-03) for non-JSON or [] JSON. DoList
// handles HATEOAS `_embedded.organizationResourceList` unwrap and auto-pagination
// via cmd.AddListFlags (CF-12-19).
func newChildrenCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "children <parentId>",
		Short: "List direct child organizations of a parent organization",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # List direct children of an organization
  revenium organizations children parent-org-123

  # List children as JSON
  revenium organizations children parent-org-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			// RESEARCH Pitfall 1: literal `parent/` path segment — NOT `/{id}/children`.
			path := fmt.Sprintf("/v2/api/organizations/parent/%s", url.PathEscape(args[0]))
			var orgs []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &orgs); err != nil {
				return err
			}
			if len(orgs) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No child organizations found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(orgs), orgs)
		},
	}

	cmd.AddListFlags(c)
	return c
}
