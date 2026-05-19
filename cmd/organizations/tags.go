package organizations

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newTagsCmd returns the `tags <id>` sub-command. Reads the OAS-LOCKED endpoint
// `GET /v2/api/organizations/{id}/tags` whose response is a flat `[]string`
// (Candidate A per RESEARCH D-02 LOCKED). Renders via the single-column
// tagsTableDef declared in organizations.go.
func newTagsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tags <id>",
		Short: "View tags for an organization",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # View tags for an organization
  revenium organizations tags org-123

  # View as JSON
  revenium organizations tags org-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/organizations/%s/tags", url.PathEscape(args[0]))

			var tags []string
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &tags); err != nil {
				return err
			}

			if len(tags) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No tags.")
				return nil
			}

			rows := make([][]string, len(tags))
			for i, t := range tags {
				rows[i] = []string{t}
			}
			return cmd.Output.Render(tagsTableDef, rows, tags)
		},
	}
}
