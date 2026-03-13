package charts

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var label, chartType, description string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a chart definition",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a chart definition label
  revenium charts update chart-123 --label "New Label"

  # Update multiple fields
  revenium charts update chart-123 --label "New Label" --chart-type LINE`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("label") {
				updates["label"] = label
			}
			if c.Flags().Changed("chart-type") {
				updates["type"] = chartType
			}
			if c.Flags().Changed("description") {
				updates["description"] = description
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "chart", "/v2/api/reports/chart-definitions/"+id, updates)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/reports/chart-definitions/"+id, updates, &result); err != nil {
				return err
			}
			return renderChart(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Chart label")
	c.Flags().StringVar(&chartType, "chart-type", "", "Chart type (LINE, BAR, COLUMN, DUAL_AXIS, PIE)")
	c.Flags().StringVar(&description, "description", "", "Chart description")

	return c
}
