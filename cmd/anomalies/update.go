package anomalies

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var name string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an anomaly detection rule",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update an anomaly rule name
  revenium anomalies update anom-123 --name "New Name"

  # Update an anomaly rule
  revenium anomalies update anom-123 --name "Updated Rule"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("name") {
				body["name"] = name
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/sources/ai/anomaly/"+id, body, &result); err != nil {
				return err
			}
			return renderAnomaly(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Anomaly rule name")

	return c
}
