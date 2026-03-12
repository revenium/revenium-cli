package subscriptions

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a subscription by ID",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get a subscription by ID
  revenium subscriptions get sub-123

  # Get a subscription as JSON
  revenium subscriptions get sub-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var sub map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/subscriptions/"+args[0], nil, &sub); err != nil {
				return err
			}
			return renderSubscription(sub)
		},
	}
}
