package anomalies

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an anomaly detection rule by ID",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get an anomaly rule by ID
  revenium anomalies get anom-123

  # Get an anomaly rule as JSON
  revenium anomalies get anom-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var anomaly map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/ai/anomaly/"+args[0], nil, &anomaly); err != nil {
				return err
			}
			return renderAnomaly(anomaly)
		},
	}
}
