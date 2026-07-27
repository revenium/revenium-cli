package refunds

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newGetCmd returns the `revenium refunds get <id>` command.
// Endpoint: GET /v2/api/refunds/{url.PathEscape(id)}.
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a refund by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a refund by ID
  revenium refunds get ref-123

  # Get a refund as JSON
  revenium refunds get ref-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/refunds/%s", url.PathEscape(args[0]))
			var refund map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &refund); err != nil {
				return err
			}
			return renderRefund(refund)
		},
	}
}
