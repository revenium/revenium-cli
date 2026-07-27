package meteringelements

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newGetCmd returns the `revenium metering-elements get <id>` command.
// Endpoint: GET /v2/api/metering-element-definitions/{url.PathEscape(id)}.
// Args validation uses cobra.MatchAll(ExactArgs(1), ValidResourceID) plus
// url.PathEscape on the interpolated segment (T-06-12 mitigation).
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a metering element definition by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a metering element definition by ID
  revenium metering-elements get me-123

  # Get a metering element definition as JSON
  revenium metering-elements get me-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/metering-element-definitions/%s", url.PathEscape(args[0]))
			var element map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &element); err != nil {
				return err
			}
			return renderElement(element)
		},
	}
}
