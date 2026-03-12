package charts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var label, chartType, description string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a chart definition",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a chart definition label
  revenium charts update chart-123 --label "New Label"

  # Update multiple fields
  revenium charts update chart-123 --label "New Label" --type line`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("label") {
				body["label"] = label
			}
			if c.Flags().Changed("type") {
				body["type"] = chartType
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/reports/chart-definitions/"+id, body, &result); err != nil {
				return err
			}
			return renderChart(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Chart label")
	c.Flags().StringVar(&chartType, "type", "", "Chart type (e.g. bar, line, pie)")
	c.Flags().StringVar(&description, "description", "", "Chart description")

	return c
}
