package subscriptions

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// quotaTableDef defines the table layout for the quota command.
var quotaTableDef = output.TableDef{
	Headers:      []string{"ID", "Consumed", "Limit"},
	StatusColumn: -1,
}

// newQuotaCmd builds the "subscriptions quota <id>" command (RES-06).
// The CLI verb is "quota" per roadmap naming; the underlying spec path segment
// is "quota-consumed" (naming discretion — see 05-RESEARCH.md § RES-06).
func newQuotaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quota <id>",
		Short: "Get quota consumption for a subscription",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get quota consumption for a subscription
  revenium subscriptions quota sub-123`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/subscriptions/%s/quota-consumed", url.PathEscape(id))

			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resp); err != nil {
				return err
			}

			rows := [][]string{{id, str(resp, "consumed"), str(resp, "limit")}}
			return cmd.Output.Render(quotaTableDef, rows, resp)
		},
	}
}
