package guardrails

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newEnforcementEventsListCmd returns the `revenium guardrails enforcement-events list` verb.
// GETs /v2/api/ai/enforcement-events and renders a 5-col multi-row table (Rule / Action / Metric
// / Used / Time). teamId is auto-injected by cmd.APIClient.Do from Client.TeamID — NOT a CLI flag.
// Exposes --since and --rule-id filter flags whose values are appended to the path as
// `?since=...` and `?ruleId=...` BEFORE DoList adds page/size (RESEARCH D-12).
// Read-only per REQUIREMENTS.md Out of Scope — no dry-run / confirmation flow.
func newEnforcementEventsListCmd() *cobra.Command {
	var since string
	var ruleID string

	c := &cobra.Command{
		Use:   "list",
		Short: "List recent enforcement events for the configured team",
		Args:  cobra.NoArgs,
		Example: `  # List recent enforcement events
  revenium guardrails enforcement-events list

  # List events since a timestamp
  revenium guardrails enforcement-events list --since 2026-05-01T00:00:00Z

  # Filter events by rule
  revenium guardrails enforcement-events list --rule-id jR2kmLs`,
		RunE: func(c *cobra.Command, args []string) error {
			path := "/v2/api/ai/enforcement-events"
			sep := "?"
			if c.Flags().Changed("since") {
				path += sep + "since=" + url.QueryEscape(since)
				sep = "&"
			}
			if c.Flags().Changed("rule-id") {
				path += sep + "ruleId=" + url.QueryEscape(ruleID)
				sep = "&"
			}
			_ = sep // sep is reassigned after last read; retain for future filter additions

			var events []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &events); err != nil {
				return err
			}
			if len(events) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No enforcement events found.")
				return nil
			}
			return cmd.Output.Render(enforcementEventsTableDef, toEnforcementEventRows(events), events)
		},
	}

	c.Flags().StringVar(&since, "since", "", "Only return events at or after this ISO-8601 timestamp (e.g., 2026-05-01T00:00:00Z)")
	c.Flags().StringVar(&ruleID, "rule-id", "", "Filter to events produced by this cost-control rule ID")
	cmd.AddListFlags(c)
	return c
}
