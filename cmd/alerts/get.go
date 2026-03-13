package alerts

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an AI alert by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get an AI alert by ID
  revenium alerts get alert-123

  # Get an AI alert as JSON
  revenium alerts get alert-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var alert map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/ai/anomaly/"+args[0], nil, &alert); err != nil {
				return err
			}
			return renderAlert(alert)
		},
	}
}
