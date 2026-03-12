package alerts

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newBudgetCreateCmd() *cobra.Command {
	var (
		name      string
		threshold float64
		currency  string
	)

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a budget alert by configuring a cumulative usage anomaly rule",
		Example: `  # Create a budget alert with threshold
  revenium alerts budget create --name "Monthly Budget" --threshold 5000

  # Create a budget alert with non-USD currency
  revenium alerts budget create --name "EU Budget" --threshold 10000 --currency EUR`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name":            name,
				"type":            "CUMULATIVE_USAGE",
				"budgetThreshold": threshold,
				"currency":        currency,
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/sources/ai/anomaly", body, &result); err != nil {
				return err
			}
			return renderAlert(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Budget alert name")
	c.Flags().Float64Var(&threshold, "threshold", 0, "Budget threshold amount")
	c.Flags().StringVar(&currency, "currency", "USD", "Currency code (e.g., USD, EUR)")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("threshold")

	return c
}
