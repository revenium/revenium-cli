package billing

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newCoverageCmd()) }

// coverageTableDef defines the table layout for the billing provider
// coverage-ratio summary. Full response available via --json (dimensions.go
// idiom — see cmd/anomalies/dimensions.go).
var coverageTableDef = output.TableDef{
	Headers:      []string{"Total Workspaces", "Covered Workspaces", "Coverage Ratio"},
	StatusColumn: -1,
}

// newCoverageCmd returns `revenium billing coverage` (GET
// /v2/api/billing/coverage, single-object report — exact analog to
// cmd/anomalies/dimensions.go's raw-Do-into-map idiom).
func newCoverageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "coverage",
		Short: "Get billing provider coverage ratio",
		Args:  cobra.NoArgs,
		Example: `  # Get provider coverage ratio
  revenium billing coverage

  # Get the full coverage response as JSON
  revenium billing coverage --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/billing/coverage", nil, &resp); err != nil {
				return err
			}

			if !cmd.Output.IsJSON() {
				fmt.Fprintln(c.OutOrStdout(), "(use --json for full detail)")
			}

			rows := [][]string{{
				formatCount(floatVal(resp, "totalWorkspaces")),
				formatCount(floatVal(resp, "coveredWorkspaces")),
				fmt.Sprintf("%.2f", floatVal(resp, "coverageRatio")),
			}}
			return cmd.Output.Render(coverageTableDef, rows, resp)
		},
	}
}
