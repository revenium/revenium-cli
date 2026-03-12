package charts

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var label, chartType, description string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new chart definition",
		Example: `  # Create a chart definition with label only
  revenium charts create --label "Revenue Chart"

  # Create a chart definition with all fields
  revenium charts create --label "Revenue Chart" --type bar --description "Monthly revenue"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"label": label,
			}
			if c.Flags().Changed("type") {
				body["type"] = chartType
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/reports/chart-definitions", body, &result); err != nil {
				return err
			}
			return renderChart(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Chart label")
	c.Flags().StringVar(&chartType, "type", "", "Chart type (e.g. bar, line, pie)")
	c.Flags().StringVar(&description, "description", "", "Chart description")
	_ = c.MarkFlagRequired("label")

	return c
}
