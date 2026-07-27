// Package alerts — events.go implements `alerts events`, the fired-alert feed
// against the actual AI Alert resource: GET /v2/api/sources/ai/alert (tag
// "AI Alert").
//
// Phase 8 DRIFT-01 finding: this package's existing list/get/create verbs
// (list.go, get.go, create.go) call /v2/api/sources/ai/anomaly — the anomaly
// detection *rule* resource, byte-for-byte identical to what cmd/anomalies
// already covers. The actual "AI Alert" (fired-alert instance) resource had
// zero CLI coverage before this file. This command intentionally does NOT
// "fix" the existing list/get/create verbs — no locked decision authorizes
// that rename/refactor scope in this phase; it is tracked for a future
// Phase 8 drift-report decision. See 05-RESEARCH.md RES-03 for the full
// finding.
package alerts

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// eventsTableDef defines the table layout for AI Alert (fired-alert) output.
var eventsTableDef = output.TableDef{
	Headers:      []string{"ID", "Label", "Triggered", "Resolved", "Value"},
	StatusColumn: -1,
}

// toEventRows converts a slice of AI Alert maps to table row strings.
func toEventRows(events []map[string]interface{}) [][]string {
	rows := make([][]string, len(events))
	for i, e := range events {
		rows[i] = []string{
			str(e, "id"),
			str(e, "label"),
			str(e, "triggeredTimestamp"),
			str(e, "resolved"),
			str(e, "triggeredValue"),
		}
	}
	return rows
}

// renderEvent renders a single AI Alert as a single-row table or JSON.
func renderEvent(event map[string]interface{}) error {
	rows := [][]string{{
		str(event, "id"),
		str(event, "label"),
		str(event, "triggeredTimestamp"),
		str(event, "resolved"),
		str(event, "triggeredValue"),
	}}
	return cmd.Output.Render(eventsTableDef, rows, event)
}

// newEventsCmd returns the `revenium alerts events` command. See the
// package-level doc comment above for the naming-collision finding this
// command resolves for RES-03.
func newEventsCmd() *cobra.Command {
	var alertType, start, end, anomalyID, ownerID string
	var resolved bool

	c := &cobra.Command{
		Use:   "events",
		Short: "List fired AI alerts (the AI Alert resource)",
		Args:  cobra.NoArgs,
		Example: `  # List all fired alerts
  revenium alerts events

  # List only resolved alerts
  revenium alerts events --resolved

  # List alerts as JSON
  revenium alerts events --json`,
		RunE: func(c *cobra.Command, args []string) error {
			q := url.Values{}
			if c.Flags().Changed("type") {
				q.Set("type", alertType)
			}
			if c.Flags().Changed("start") {
				q.Set("start", start)
			}
			if c.Flags().Changed("end") {
				q.Set("end", end)
			}
			if c.Flags().Changed("anomaly-id") {
				q.Set("anomalyId", anomalyID)
			}
			if c.Flags().Changed("owner-id") {
				q.Set("ownerId", ownerID)
			}
			if c.Flags().Changed("resolved") {
				q.Set("resolved", strconv.FormatBool(resolved))
			}

			path := "/v2/api/sources/ai/alert"
			if len(q) > 0 {
				path += "?" + q.Encode()
			}

			var events []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &events); err != nil {
				return err
			}
			if len(events) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No alert events found.")
				return nil
			}
			return cmd.Output.Render(eventsTableDef, toEventRows(events), events)
		},
	}

	c.Flags().StringVar(&alertType, "type", "", "Filter by alert type")
	c.Flags().StringVar(&start, "start", "", "Filter by start timestamp (RFC3339)")
	c.Flags().StringVar(&end, "end", "", "Filter by end timestamp (RFC3339)")
	c.Flags().StringVar(&anomalyID, "anomaly-id", "", "Filter by originating anomaly rule ID")
	c.Flags().StringVar(&ownerID, "owner-id", "", "Filter by owner ID")
	c.Flags().BoolVar(&resolved, "resolved", false, "Filter by resolved status")
	cmd.AddListFlags(c)

	c.AddCommand(newEventsGetCmd())
	return c
}

// newEventsGetCmd returns the `revenium alerts events get <id>` command, a
// natural sibling to the list verb above: GET /v2/api/sources/ai/alert/{id}.
func newEventsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a fired AI alert by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a fired alert by ID
  revenium alerts events get alert-123`,
		RunE: func(c *cobra.Command, args []string) error {
			path := "/v2/api/sources/ai/alert/" + url.PathEscape(args[0])
			var event map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &event); err != nil {
				return err
			}
			return renderEvent(event)
		},
	}
}
