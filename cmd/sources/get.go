package sources

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a source by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a source by ID
  revenium sources get abc-123

  # Get a source as JSON
  revenium sources get abc-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var source map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/"+args[0], nil, &source); err != nil {
				return err
			}
			return renderSource(source)
		},
	}
}
