package squads

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func init() { Cmd.AddCommand(newExecutionsCmd()) }

// newExecutionsCmd returns the `revenium squads executions [groupId]`
// subcommand (SQUAD-08, D-02/D-03). It is one overloaded command: the
// presence/absence of the positional arg switches the endpoint (D-03).
//
//   - No arg: flat executions feed across all squads
//     (GET /v2/api/squads), delegating to the shared cmd.FetchSquadExecutions
//     helper so this branch never diverges from the deprecated
//     `revenium metrics squads` command's fetch/render logic (D-04).
//   - One arg <groupId>: executions within one squad group
//     (GET /v2/api/squads/entities/{squadId}/executions). HELD per D-12 —
//     see runGroupExecutions.
//
// Identity-space resolution (D-11/D-12, 02-04 Task 2 checkpoint,
// 2026-07-20): live disambiguation was attempted against the operator's
// configured team (`revenium squads list` across every --period value that
// returned 200: default/THIRTY_DAYS, NINETY_DAYS, SIX_MONTHS). All returned
// zero squad-group records, so there was no squad-group id available to
// exercise GET /v2/api/squads/entities/{squadId}/executions against and
// confirm whether {squadId} is the squad-group hash id (e.g. "JMwX9g4") or
// the raw telemetry squadId (e.g. "squad-loan-proc-12345"). This is exactly
// D-12's stated fallback trigger ("the operator's configured team has no
// squad-group data to test against"): the flat branch ships; the
// within-group branch is held behind a clear error until a team with
// squad-group data can complete the live disambiguation.
func newExecutionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "executions [groupId]",
		Short: "List squad executions (flat, or within one squad group)",
		Long: `List squad executions.

With no argument, lists the flat executions feed across all squads
(GET /v2/api/squads) -- the same underlying fetch used by the deprecated
"revenium metrics squads" command (D-04).

With a <groupId> argument: HELD per D-12. Live validation (02-04 Task 2)
found the configured team had no squad-group data to test
GET /v2/api/squads/entities/{groupId}/executions against, so the {groupId}
identity space (squad-group hash id vs. raw telemetry squadId) could not be
confirmed. This form returns an error until a team with squad-group data
completes the live disambiguation -- use the no-arg flat form meanwhile.`,
		Args: cobra.MaximumNArgs(1),
		Example: `  # List flat executions across all squads
  revenium squads executions

  # Scope to a period
  revenium squads executions --period SEVEN_DAYS

  # HELD per D-12 (returns an error until identity space is confirmed):
  #   revenium squads executions JMwX9g4`,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runFlatExecutions(c)
			}
			return runGroupExecutions(c, args)
		},
	}
	return c
}

// runFlatExecutions handles the no-arg branch: GET /v2/api/squads via the
// shared helper (D-04) — never duplicate this fetch/render logic here.
func runFlatExecutions(c *cobra.Command) error {
	metrics, err := cmd.FetchSquadExecutions(c.Context(), periodFlag)
	if err != nil {
		return err
	}
	return renderSquadExecutions(c, metrics)
}

// runGroupExecutions handles the with-arg branch: GET
// /v2/api/squads/entities/{squadId}/executions. T-2-01 mitigation:
// cmd.ValidResourceID is still called manually (MaximumNArgs cannot compose
// with MatchAll) so malformed input is rejected before anything else, even
// though the D-12 gate below means no HTTP call is made on this branch yet.
//
// D-12 HELD: the {squadId} identity space (squad-group hash id vs. raw
// telemetry squadId, D-11) could not be confirmed live because the
// operator's configured team had zero squad-group records to test with
// (02-04 Task 2 checkpoint, 2026-07-20 — see the package comment above for
// the exact periods tried). Ship this branch as a guarded error instead of
// an unconfirmed guess. When a team with squad-group data becomes
// available, re-run the disambiguation procedure in 02-04-PLAN.md Task 2's
// <how-to-verify>, then replace this guard with the real
// url.PathEscape + cmd.APIClient.Do implementation (removed here, still in
// git history at the Task 1 commit).
func runGroupExecutions(c *cobra.Command, args []string) error {
	if err := cmd.ValidResourceID(c, args); err != nil {
		return err
	}
	return fmt.Errorf("squads executions <groupId>: identity space unconfirmed — held per D-12 " +
		"(live validation on 2026-07-20 found no squad-group data in the configured team to test " +
		"GET /v2/api/squads/entities/{squadId}/executions against); use " +
		"`revenium squads executions` (no arg) for the flat executions feed in the meantime")
}

// renderSquadExecutions renders the shared executions table (both branches
// use the same columns per D-04's shared SquadExecutionsTableDef), with the
// standard empty-result exit-0 shape.
func renderSquadExecutions(c *cobra.Command, metrics []map[string]interface{}) error {
	if len(metrics) == 0 {
		if cmd.Output.IsJSON() {
			return cmd.Output.RenderJSON([]interface{}{})
		}
		fmt.Fprintln(c.OutOrStdout(), "No executions found.")
		return nil
	}
	return cmd.Output.Render(cmd.SquadExecutionsTableDef, cmd.ToSquadExecutionRows(metrics), metrics)
}
