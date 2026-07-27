package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// init registers the outcome-history subcommand on the parent jobs.Cmd,
// following the established multi-init-per-package pattern (see outcome.go).
func init() {
	Cmd.AddCommand(newOutcomeHistoryCmd())
}

// outcomeHistoryTableDef defines the table layout for outcome revision
// history output (RES-07): Sequence / Reported At / Reported By / Status /
// Reason, mapping JobOutcomeRevisionResource_Read fields.
var outcomeHistoryTableDef = output.TableDef{
	Headers:      []string{"Sequence", "Reported At", "Reported By", "Status", "Reason"},
	StatusColumn: 3,
}

func outcomeHistoryToRows(revisions []map[string]interface{}) [][]string {
	rows := make([][]string, len(revisions))
	for i, r := range revisions {
		rows[i] = []string{
			str(r, "sequence"),
			str(r, "reportedAt"),
			str(r, "reportedBy"),
			str(r, "executionStatus"),
			str(r, "reason"),
		}
	}
	return rows
}

// newOutcomeHistoryCmd builds `revenium jobs outcome-history <agenticJobId>`.
//
// The spec (RES-07 / 05-PATTERNS.md) declares NO pagination parameters on
// this operation — only agenticJobId + teamId. Unlike other list commands in
// this package, this command deliberately skips the shared page/page-size
// flag registration helper (that would advertise flags the server ignores),
// and uses an explicit api.ListOptions{Page: -1, PageSize: -1} (the "not
// set" sentinel, FetchAll left false) rather than the shared flag-derived
// options helper — the latter defaults FetchAll to true in table mode,
// which would cause internal/api.Client's fetch-all path to append
// page/size query params on the first request regardless of flag
// registration, violating the "no pagination params" contract for this
// endpoint.
func newOutcomeHistoryCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "outcome-history <agenticJobId>",
		Short: "List the outcome revision history for a job",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # List outcome revisions for a job
  revenium jobs outcome-history loan-app-12345

  # As JSON
  revenium jobs outcome-history loan-app-12345 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/jobs/%s/outcome/history", url.PathEscape(id))

			var revisions []map[string]interface{}
			opts := api.ListOptions{Page: -1, PageSize: -1}
			if err := cmd.APIClient.DoList(c.Context(), path, opts, &revisions); err != nil {
				return err
			}

			if len(revisions) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No outcome revisions found.")
				return nil
			}

			return cmd.Output.Render(outcomeHistoryTableDef, outcomeHistoryToRows(revisions), revisions)
		},
	}

	return c
}
