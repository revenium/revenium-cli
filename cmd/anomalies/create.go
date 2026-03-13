package anomalies

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var (
		name           string
		alertType      string
		metricType     string
		operatorType   string
		threshold      float64
		periodDuration string
		notifications  []string
	)

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new anomaly detection rule",
		Example: `  # Create a cost anomaly rule
  revenium anomalies create --name "High Cost Alert" --threshold 100

  # Create a tokens-per-minute anomaly rule
  revenium anomalies create --name "TPM Spike" --metric-type TOKENS_PER_MINUTE --threshold 5000 --period ONE_HOUR --notify admin@example.com`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name":                  name,
				"alertType":             alertType,
				"metricType":            metricType,
				"operatorType":          operatorType,
				"threshold":             threshold,
				"periodDuration":        periodDuration,
				"notificationAddresses": notifications,
				"slackConfigurations":   []interface{}{},
				"webhookConfigurations": []interface{}{},
				"enabled":               true,
				"firing":                false,
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/sources/ai/anomaly", body, &result); err != nil {
				return err
			}
			return renderAnomaly(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Anomaly rule name")
	c.Flags().StringVar(&alertType, "alert-type", "THRESHOLD", "Alert type (THRESHOLD, RELATIVE_CHANGE, CUMULATIVE_USAGE)")
	c.Flags().StringVar(&metricType, "metric-type", "TOTAL_COST", "Metric type (TOTAL_COST, TOKEN_COUNT, TOKENS_PER_MINUTE, etc.)")
	c.Flags().StringVar(&operatorType, "operator-type", "GREATER_THAN", "Operator (GREATER_THAN, LESS_THAN, etc.)")
	c.Flags().Float64Var(&threshold, "threshold", 0, "Threshold value")
	c.Flags().StringVar(&periodDuration, "period", "DAILY", "Evaluation period (DAILY, WEEKLY, MONTHLY, QUARTERLY)")
	c.Flags().StringSliceVar(&notifications, "notify", []string{}, "Notification email addresses")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("threshold")

	return c
}
