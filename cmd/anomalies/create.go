package anomalies

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var name string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new anomaly detection rule",
		Example: `  # Create an anomaly rule
  revenium anomalies create --name "High Cost Alert"

  # Create an anomaly rule (API validates required fields)
  revenium anomalies create --name "Latency Spike"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name": name,
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/sources/ai/anomaly", body, &result); err != nil {
				return err
			}
			return renderAnomaly(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Anomaly rule name")
	_ = c.MarkFlagRequired("name")

	return c
}
