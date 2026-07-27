package workspaces

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newListCmd returns the `revenium workspaces list` command.
// Endpoint: GET /v2/api/workspaces (PagedModel_Read — generic
// `_embedded.objectList`, handled by DoList regardless of the embedded key
// name). This endpoint IS page-based (unlike period-charges), so
// cmd.AddListFlags/cmd.ListOptsFromFlags apply normally.
//
// --query and --provider are flag-gated optional query params, appended
// only when explicitly passed (idiom copied from
// cmd/anomalies/dimensions.go lines 46-51); when both are set they chain
// with `&` (idiom copied from cmd/squads/squads.go's buildSquadsPath).
func newListCmd() *cobra.Command {
	var query, provider string

	c := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Args:  cobra.NoArgs,
		Example: `  # List all workspaces
  revenium workspaces list

  # Filter by provider
  revenium workspaces list --provider aws

  # Filter by query
  revenium workspaces list --query prod`,
		RunE: func(c *cobra.Command, args []string) error {
			path := "/v2/api/workspaces"
			var params []string
			if c.Flags().Changed("query") {
				params = append(params, "query="+url.QueryEscape(query))
			}
			if c.Flags().Changed("provider") {
				params = append(params, "provider="+url.QueryEscape(provider))
			}
			for i, p := range params {
				sep := "&"
				if i == 0 {
					sep = "?"
				}
				path += sep + p
			}

			var workspaces []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &workspaces); err != nil {
				return err
			}
			if len(workspaces) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No workspaces found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(workspaces), workspaces)
		},
	}

	c.Flags().StringVar(&query, "query", "", "Filter by query string")
	c.Flags().StringVar(&provider, "provider", "", "Filter by provider")
	cmd.AddListFlags(c)
	return c
}
