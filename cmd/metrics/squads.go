package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// squadsPeriodFlag holds the --period value for this command. Empty means
// the flag was not passed — the query param is omitted entirely and the
// server applies its documented THIRTY_DAYS default (do not hardcode that
// default client-side).
var squadsPeriodFlag string

// newSquadsCmd returns the deprecated `revenium metrics squads` command
// (D-04/D-05). It is kept (not deleted, per Out-of-Scope) but now delegates
// entirely to the shared cmd.FetchSquadExecutions/cmd.ToSquadExecutionRows
// helper (from Plan 01) instead of calling the fictional startDate/endDate
// path and reading fictional transactionId/name/executions fields. New
// usage should prefer `revenium squads executions`.
func newSquadsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "squads",
		Short: "Query squad metrics (DEPRECATED: use `revenium squads executions`)",
		Long: `DEPRECATED: this command is superseded by "revenium squads executions", which
provides the same flat squad-execution data under the dedicated squads command
group. It is kept here for backward compatibility only.

Query squad execution metrics from the Squads API.`,
		Args: cobra.NoArgs,
		Example: `  # Query squad execution metrics
  revenium metrics squads

  # Query squad execution metrics for the last 7 days
  revenium metrics squads --period SEVEN_DAYS`,
		RunE: func(c *cobra.Command, args []string) error {
			// T-2-02 mitigation: reject unknown --period values before they
			// ever reach the URL.
			if err := cmd.ValidatePeriod(squadsPeriodFlag); err != nil {
				return err
			}
			metrics, err := cmd.FetchSquadExecutions(c.Context(), squadsPeriodFlag)
			if err != nil {
				return err
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No squad metrics found.")
				return nil
			}
			return cmd.Output.Render(cmd.SquadExecutionsTableDef, cmd.ToSquadExecutionRows(metrics), metrics)
		},
	}

	c.Flags().StringVar(&squadsPeriodFlag, "period", "",
		"Time period: HOUR, EIGHT_HOURS, TWENTY_FOUR_HOURS, SEVEN_DAYS, "+
			"THIRTY_DAYS, NINETY_DAYS, SIX_MONTHS, TWELVE_MONTHS (default THIRTY_DAYS server-side)")
	return c
}
