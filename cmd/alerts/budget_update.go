package alerts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newBudgetUpdateCmd() *cobra.Command {
	var (
		name      string
		threshold float64
		currency  string
	)

	c := &cobra.Command{
		Use:   "update <anomaly-id>",
		Short: "Update a budget alert by modifying the underlying anomaly rule",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a budget alert threshold
  revenium alerts budget update anom-123 --threshold 10000

  # Update a budget alert name and currency
  revenium alerts budget update anom-123 --name "New Name" --currency EUR`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("name") {
				body["name"] = name
			}
			if c.Flags().Changed("threshold") {
				body["budgetThreshold"] = threshold
			}
			if c.Flags().Changed("currency") {
				body["currency"] = currency
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/sources/ai/anomaly/"+id, body, &result); err != nil {
				return err
			}
			return renderAlert(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Budget alert name")
	c.Flags().Float64Var(&threshold, "threshold", 0, "Budget threshold amount")
	c.Flags().StringVar(&currency, "currency", "", "Currency code (e.g., USD, EUR)")

	return c
}
