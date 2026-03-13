package charts

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
	var label, chartType, description, mode, dateRange, baseMetric, calculationType string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new chart definition",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Create a chart definition
  revenium charts create --label "Revenue Chart" --chart-type line

  # Create a chart definition with all fields
  revenium charts create --label "Revenue Chart" --chart-type bar --description "Monthly revenue" --date-range 30d`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name": label,
				"configuration": map[string]interface{}{
					"chartType": chartType,
					"mode":      mode,
					"dateRange": dateRange,
					"primaryMetric": map[string]interface{}{
						"baseMetric":      baseMetric,
						"calculationType": calculationType,
					},
				},
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "chart", "/v2/api/reports/chart-definitions", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/reports/chart-definitions", body, &result); err != nil {
				return err
			}
			return renderChart(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Chart label")
	c.Flags().StringVar(&chartType, "chart-type", "line", "Chart type (line, bar, column, dual-axis, pie)")
	c.Flags().StringVar(&description, "description", "", "Chart description")
	c.Flags().StringVar(&mode, "mode", "time", "Chart mode (time, aggregate)")
	c.Flags().StringVar(&dateRange, "date-range", "30d", "Date range (1h, 8h, 24h, 7d, 30d, 90d, 180d, 365d, custom)")
	c.Flags().StringVar(&baseMetric, "metric", "totalCost", "Base metric (totalCost, requestCount, tokenCount, etc.)")
	c.Flags().StringVar(&calculationType, "calculation", "total", "Calculation type (total, average, per-request, etc.)")
	_ = c.MarkFlagRequired("label")

	return c
}
