package tenants

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newUsageCmd returns the `revenium tenants usage <id>` command.
//
// LANDMINE (06-RESEARCH.md Pitfall 4): GET /v2/api/tenants/{id}/usage's
// response schema is {"type": "string"} — a bare JSON string, NOT an
// object. Decoding into map[string]interface{} (the pattern every other
// verb in this phase uses) fails with "json: cannot unmarshal string into
// Go value of type map[string]interface {}". Decode into a plain string
// instead, and do NOT route this through tenantsTableDef/renderTenant —
// there is no map to extract columns from.
func newUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage <id>",
		Short: "Get a tenant's usage summary",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a tenant's usage summary
  revenium tenants usage ten-123

  # Get a tenant's usage summary as JSON
  revenium tenants usage ten-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/tenants/%s/usage", url.PathEscape(args[0]))
			var usage string
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &usage); err != nil {
				return err
			}
			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(usage)
			}
			fmt.Fprintln(c.OutOrStdout(), usage)
			return nil
		},
	}
}
