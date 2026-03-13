package alerts

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newBudgetCreateCmd() *cobra.Command {
	var (
		name           string
		threshold      float64
		currency       string
		periodDuration string
		notifications  []string
	)

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a budget alert by configuring a cumulative usage anomaly rule",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Create a budget alert with threshold
  revenium alerts budget create --name "Monthly Budget" --threshold 5000

  # Create a budget alert with non-USD currency
  revenium alerts budget create --name "EU Budget" --threshold 10000 --currency EUR

  # Create with notification
  revenium alerts budget create --name "Budget" --threshold 5000 --notify admin@example.com`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name":                  name,
				"alertType":             "CUMULATIVE_USAGE",
				"metricType":            "TOTAL_COST",
				"operatorType":          "GREATER_THAN",
				"budgetThreshold":       threshold,
				"threshold":             threshold,
				"currency":              currency,
				"periodDuration":        periodDuration,
				"notificationAddresses": notifications,
				"slackConfigurations":   []interface{}{},
				"webhookConfigurations": []interface{}{},
				"enabled":               true,
				"firing":                false,
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "budget alert", "/v2/api/sources/ai/anomaly", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/sources/ai/anomaly", body, &result); err != nil {
				return err
			}
			return renderAlert(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Budget alert name")
	c.Flags().Float64Var(&threshold, "threshold", 0, "Budget threshold amount")
	c.Flags().StringVar(&currency, "currency", "USD", "Currency code (e.g., USD, EUR)")
	c.Flags().StringVar(&periodDuration, "period", "MONTHLY", "Evaluation period (DAILY, WEEKLY, MONTHLY, QUARTERLY)")
	c.Flags().StringSliceVar(&notifications, "notify", []string{}, "Notification email addresses")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("threshold")

	return c
}
