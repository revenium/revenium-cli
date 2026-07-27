package billing

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newChartCmd()) }

// chartTableDef defines the table layout for the billing provider chart
// summary. Full response available via --json (dimensions.go idiom).
var chartTableDef = output.TableDef{
	Headers:      []string{"# Providers", "Total Cost"},
	StatusColumn: -1,
}

// newChartCmd returns `revenium billing chart` (GET /v2/api/billing/chart,
// single-object report — exact analog to cmd/anomalies/dimensions.go's
// raw-Do-into-map idiom).
func newChartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chart",
		Short: "Get billing provider chart data",
		Args:  cobra.NoArgs,
		Example: `  # Get provider billing chart summary
  revenium billing chart

  # Get the full chart response as JSON
  revenium billing chart --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/billing/chart", nil, &resp); err != nil {
				return err
			}

			if !cmd.Output.IsJSON() {
				fmt.Fprintln(c.OutOrStdout(), "(use --json for full detail)")
			}

			rows := [][]string{{
				countStr(resp, "providers"),
				formatCost(floatVal(resp, "totalCost")),
			}}
			return cmd.Output.Render(chartTableDef, rows, resp)
		},
	}
}
