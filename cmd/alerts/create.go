package alerts

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var name string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create an anomaly detection rule that generates AI alerts",
		Example: `  # Create an alert rule
  revenium alerts create --name "High Cost Alert"

  # Create an alert rule for latency monitoring
  revenium alerts create --name "Latency Spike"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name": name,
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/sources/ai/anomaly", body, &result); err != nil {
				return err
			}
			return renderAlert(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Alert rule name")
	_ = c.MarkFlagRequired("name")

	return c
}
