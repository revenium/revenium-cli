package subscriptions

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// billedAmountTableDef defines the table layout for the billed-amount command.
var billedAmountTableDef = output.TableDef{
	Headers:      []string{"ID", "Amount Billed"},
	StatusColumn: -1,
}

// newBilledAmountCmd builds the "subscriptions billed-amount <id>" command (RES-06).
func newBilledAmountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "billed-amount <id>",
		Short: "Get the billed amount for a subscription",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get the billed amount for a subscription
  revenium subscriptions billed-amount sub-123`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/subscriptions/%s/billed-amount", url.PathEscape(id))

			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resp); err != nil {
				return err
			}

			rows := [][]string{{id, str(resp, "amountBilled")}}
			return cmd.Output.Render(billedAmountTableDef, rows, resp)
		},
	}
}
