package periodcharges

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newGetCmd returns the `revenium period-charges get <id>` command.
// Endpoint: GET /v2/api/period-charges/{url.PathEscape(id)}. This is an
// ordinary single-object Do — the cursor pagination landmine documented in
// list.go is list-only and does not apply here.
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a period charge by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a period charge by ID
  revenium period-charges get pc-123

  # Get a period charge as JSON
  revenium period-charges get pc-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/period-charges/%s", url.PathEscape(args[0]))
			var pc map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &pc); err != nil {
				return err
			}
			return renderPeriodCharge(pc)
		},
	}
}
