package subscribers

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a subscriber by ID",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get a subscriber by ID
  revenium subscribers get abc-123

  # Get a subscriber as JSON
  revenium subscribers get abc-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var subscriber map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/subscribers/"+args[0], nil, &subscriber); err != nil {
				return err
			}
			return renderSubscriber(subscriber)
		},
	}
}
