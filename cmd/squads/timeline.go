package squads

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newTimelineCmd()) }

// timelineTableDef defines the event-waterfall table layout (SQUAD-07).
// Columns follow the real SquadTimelineEvent_Read fields.
var timelineTableDef = output.TableDef{
	Headers:      []string{"Timestamp", "Agent", "Role", "Model", "Duration", "Cost"},
	StatusColumn: -1,
}

// toTimelineRows converts a slice of timeline-event maps to table row
// strings.
func toTimelineRows(events []map[string]interface{}) [][]string {
	rows := make([][]string, len(events))
	for i, e := range events {
		rows[i] = []string{
			str(e, "startTime"),
			str(e, "agent"),
			str(e, "role"),
			str(e, "model"),
			formatDuration(floatVal(e, "duration")),
			formatCost(floatVal(e, "cost")),
		}
	}
	return rows
}

// newTimelineCmd returns the `revenium squads timeline <squadId>`
// subcommand (GET /v2/api/squads/{squadId}/timeline, SQUAD-07, D-02). The
// response is a wrapper object ({squadId, squadName, startTime, endTime,
// totalDuration, events: [...]}), not a bare array or Spring HATEOAS
// envelope, so it uses cmd.APIClient.Do (not DoList) with manual extraction
// of the nested events array — mirrors cmd/jobs/transactions.go. --period is
// threaded via the SAME buildSquadsPath helper get.go uses (RESEARCH
// confirms the timeline endpoint accepts the optional period enum).
func newTimelineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "timeline <squadId>",
		Short: "View a squad's event timeline (waterfall)",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # View a squad's event timeline
  revenium squads timeline squad-loan-proc-12345

  # View a squad's timeline for a specific period
  revenium squads timeline squad-loan-proc-12345 --period SEVEN_DAYS

  # View a squad's timeline as JSON (preserves the wrapper object)
  revenium squads timeline squad-loan-proc-12345 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := buildSquadsPath(fmt.Sprintf("/v2/api/squads/%s/timeline", url.PathEscape(args[0])))

			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resp); err != nil {
				return err
			}

			// Extract the events array from the wrapper object (NOT Spring
			// HATEOAS) — Do, not DoList. Mis-typed elements are silently
			// skipped, matching cmd/jobs/transactions.go's defensive extraction.
			raw, _ := resp["events"].([]interface{})
			events := make([]map[string]interface{}, 0, len(raw))
			for _, r := range raw {
				if m, ok := r.(map[string]interface{}); ok {
					events = append(events, m)
				}
			}

			if len(events) == 0 {
				if cmd.Output.IsJSON() {
					// Preserve the wrapper (NOT a bare []) so squadId/squadName/
					// totalDuration survive for script consumers.
					return cmd.Output.RenderJSON(resp)
				}
				fmt.Fprintln(c.OutOrStdout(), "No timeline events found.")
				return nil
			}

			// Render's third arg is the full wrapper resp (NOT events) so
			// --json emits it whole.
			return cmd.Output.Render(timelineTableDef, toTimelineRows(events), resp)
		},
	}
}
