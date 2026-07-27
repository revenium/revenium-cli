package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newChildrenCmd returns the `revenium teams children <parentId>` command.
// Endpoint: GET /v2/api/teams/parent/{parentId} — literal `parent/` path segment
// is mandatory; `/{id}/children` returns 404 (RESEARCH Pitfall 6, confirmed twice
// across organizations and teams). Renders the shared teams tableDef (ID / Name)
// populated via toRows; empty list emits the no-children-found empty-state phrase
// for non-JSON or [] JSON. DoList handles HATEOAS `_embedded` unwrap and
// auto-pagination via cmd.AddListFlags.
func newChildrenCmd() *cobra.Command {
	var isDemo bool

	c := &cobra.Command{
		Use:   "children <parentId>",
		Short: "List direct child teams of a parent team",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # List direct children of a team
  revenium teams children parent-team-123

  # List children as JSON
  revenium teams children parent-team-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			// RESEARCH Pitfall 6: literal `parent/` path segment — NOT `/{id}/children`.
			path := fmt.Sprintf("/v2/api/teams/parent/%s", url.PathEscape(args[0]))
			if c.Flags().Changed("is-demo") {
				path += fmt.Sprintf("?isDemo=%t", isDemo)
			}
			var teamsResp []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &teamsResp); err != nil {
				return err
			}
			if len(teamsResp) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No child teams found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(teamsResp), teamsResp)
		},
	}

	c.Flags().BoolVar(&isDemo, "is-demo", false, "Filter to demo teams only")
	cmd.AddListFlags(c)
	return c
}
