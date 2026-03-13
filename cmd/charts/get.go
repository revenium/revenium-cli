package charts

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a chart definition by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a chart definition by ID
  revenium charts get chart-123

  # Get a chart definition as JSON
  revenium charts get chart-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var chart map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/reports/chart-definitions/"+args[0], nil, &chart); err != nil {
				return err
			}
			return renderChart(chart)
		},
	}
}
