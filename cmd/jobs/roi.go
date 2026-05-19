package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// init registers the roi subcommand on the parent jobs.Cmd. Per Phase 12
// D-21 / CF-12, multi-init per package composes — this init() runs alongside
// the init()s in jobs.go, create.go, update.go, delete.go, and outcome.go.
func init() {
	Cmd.AddCommand(newROICmd())
}

// roiTableDef defines the 2-column key-value table layout for ROI output
// per CONTEXT D-03. Mirrors cmd/teams/prompt_capture.go promptCaptureTableDef.
// StatusColumn: -1 disables status colorization — none of the Value cells
// carry status semantics.
var roiTableDef = output.TableDef{
	Headers:      []string{"Metric", "Value"},
	StatusColumn: -1,
}

// newROICmd builds `revenium jobs roi <agenticJobId>`.
//
// Issues GET /v2/api/jobs/<id>/roi and renders the response as a 2-column
// key-value table per CONTEXT D-03 / JOBS-07. Currency cells (Total Cost,
// Outcome Value) are formatted via the Plan 13-01 output.FormatCurrency
// helper — no local helper duplication. --json returns the raw response
// unmodified (the Render dispatcher in internal/output/json.go branches
// on IsJSON internally; renderROI must NOT branch).
func newROICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "roi <agenticJobId>",
		Short: "View ROI metrics for a job",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID), // CF-17
		Example: `  # View ROI metrics for a job
  revenium jobs roi loan-app-12345

  # View ROI metrics as JSON
  revenium jobs roi loan-app-12345 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			// D-25 / CF-13: defensive PathEscape because id is user-supplied
			// (cobra arg validator already rejects ?, &, #, %, ../, ..\ and
			// control chars; PathEscape covers everything else the validator
			// allows through — most importantly "/" which appears as %2F).
			path := fmt.Sprintf("/v2/api/jobs/%s/roi", url.PathEscape(id))
			var roi map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &roi); err != nil {
				return err
			}
			return renderROI(roi)
		},
	}
}

// renderROI builds the explicit 13-row key-value table and dispatches to
// cmd.Output.Render (which auto-branches table-vs-JSON via IsJSON).
//
// Display order is INTENTIONALLY explicit (RESEARCH §Pitfall 4): identity
// first → status → money → tokens. A for/range over the response map would
// produce non-deterministic Go map iteration order; the slice below is the
// source of truth for column 1 ordering and is the file's load-bearing
// contract.
//
// Currency rules:
//   - totalCost uses the literal "USD" — the OAS does not pair totalCost
//     with a currency field (RESEARCH §A2 / Assumption A1; the platform-
//     wide invariant for AI cost is USD). Acceptable v1 risk per CONTEXT
//     D-03; --json returns the raw number for scriptable users.
//   - outcomeValue uses the response's outcomeCurrency field (so EUR / GBP
//     outcomes render with the correct prefix).
//   - The roi field is already a percentage value in the OAS (e.g., 19900
//     means 19,900%), so "%.2f%%" applied directly is correct — DO NOT
//     multiply by 100.
//
// outcomeCurrency is NOT shown as its own row — it's the formatting driver
// for the Outcome Value row.
func renderROI(roi map[string]interface{}) error {
	currency := str(roi, "outcomeCurrency")
	rows := [][]string{
		{"Job ID", str(roi, "agenticJobId")},
		{"Name", str(roi, "agenticJobName")},
		{"Type", str(roi, "agenticJobType")},
		{"Execution Status", str(roi, "executionStatus")},
		{"Outcome Type", str(roi, "outcomeType")},
		{"Has Outcome", str(roi, "hasOutcome")},
		{"Total Cost", output.FormatCurrency(output.FloatVal(roi, "totalCost"), "USD")},
		{"Outcome Value", output.FormatCurrency(output.FloatVal(roi, "outcomeValue"), currency)},
		{"ROI %", fmt.Sprintf("%.2f%%", output.FloatVal(roi, "roi"))},
		{"Transaction Count", str(roi, "transactionCount")},
		{"Input Tokens", str(roi, "inputTokens")},
		{"Output Tokens", str(roi, "outputTokens")},
		{"Total Tokens", str(roi, "totalTokens")},
	}
	return cmd.Output.Render(roiTableDef, rows, roi)
}
