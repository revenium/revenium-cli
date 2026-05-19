package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// init registers newConversionFunnelCmd() onto the package-level jobs.Cmd.
// Per Phase 12 D-21 and Go's multi-init-per-package semantics, this composes
// with the init()s in jobs.go (list+get), create.go, update.go, delete.go,
// outcome.go, and types.go.
func init() {
	Cmd.AddCommand(newConversionFunnelCmd())
}

// funnelTableDef defines the 3-column table layout for the conversion funnel
// output. StatusColumn = -1 disables status colorization — stage names are
// not status tokens.
var funnelTableDef = output.TableDef{
	Headers:      []string{"Stage", "Count", "Conversion %"},
	StatusColumn: -1,
}

// newConversionFunnelCmd builds `revenium jobs conversion-funnel` — JOBS-10.
//
// The OAS surface (GET /v2/api/jobs/conversion-funnel) returns a FLAT object
// with five scalar fields (totalJobs, successfulJobs, convertedJobs,
// successRate, conversionRate). The CLI synthesizes three stage rows
// (Total / Successful / Converted) using the server-supplied successRate and
// conversionRate decimals multiplied by 100 — per RESEARCH §A5 + Pitfall 5,
// the CLI MUST NOT recompute these rates from the counts.
//
// Four optional filter flags translate CLI kebab-case to OAS camelCase
// query-string keys (see the qs.Set calls below). Each is gated by
// c.Flags().Changed so absent flags are not serialized as empty values.
//
// Per CF-17, aggregate verbs (no id segment) use cobra.NoArgs.
func newConversionFunnelCmd() *cobra.Command {
	var (
		from        string
		to          string
		jobType     string
		environment string
	)

	c := &cobra.Command{
		Use:   "conversion-funnel",
		Short: "View aggregate conversion funnel",
		Args:  cobra.NoArgs, // CF-17 — aggregate verb, no id
		Example: `  # View the conversion funnel for the active team
  revenium jobs conversion-funnel

  # Filter by date range and job type
  revenium jobs conversion-funnel --from 2025-09-01T00:00:00Z --to 2025-09-30T23:59:59Z --job-type loan-processing

  # As JSON for scripting
  revenium jobs conversion-funnel --json`,
		RunE: func(c *cobra.Command, args []string) error {
			// Build optional query parameters. CRITICAL: the keys passed to
			// qs.Set are the OAS field names (startDate/endDate/jobType/
			// environment), NOT the CLI flag names.
			qs := url.Values{}
			if c.Flags().Changed("from") {
				qs.Set("startDate", from)
			}
			if c.Flags().Changed("to") {
				qs.Set("endDate", to)
			}
			if c.Flags().Changed("job-type") {
				qs.Set("jobType", jobType)
			}
			if c.Flags().Changed("environment") {
				qs.Set("environment", environment)
			}

			path := "/v2/api/jobs/conversion-funnel"
			if len(qs) > 0 {
				path += "?" + qs.Encode()
			}

			var funnel map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &funnel); err != nil {
				return err
			}
			return renderFunnel(funnel)
		},
	}

	// All four filter flags are optional. No MarkFlagRequired calls.
	c.Flags().StringVar(&from, "from", "", "Filter start date (ISO 8601)")
	c.Flags().StringVar(&to, "to", "", "Filter end date (ISO 8601)")
	c.Flags().StringVar(&jobType, "job-type", "", "Filter by job type")
	c.Flags().StringVar(&environment, "environment", "", "Filter by environment")

	return c
}

// renderFunnel dispatches between the --json branch (raw flat object per
// D-04) and the table branch (3 synthesized rows). It branches on IsJSON()
// at the top because the JSON shape (raw 5 scalars) and the table shape
// (3 synthesized rows) are intentionally different — passing the synthesized
// rows to Render's third argument would leak the table shape into the JSON
// output.
//
// CRITICAL (RESEARCH Pitfall 5): the Successful and Converted stages' rate
// cells use the server-supplied successRate / conversionRate decimals
// (multiplied by 100). The CLI MUST NOT recompute these rates from the
// raw counts — the server's values are the source of truth and match the
// dashboard, and re-deriving them client-side invites floating-point drift.
func renderFunnel(f map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.RenderJSON(f)
	}
	totalJobs := output.FloatVal(f, "totalJobs")
	successfulJobs := output.FloatVal(f, "successfulJobs")
	convertedJobs := output.FloatVal(f, "convertedJobs")
	successRate := output.FloatVal(f, "successRate")       // 0.0–1.0 decimal
	conversionRate := output.FloatVal(f, "conversionRate") // 0.0–1.0 decimal

	rows := [][]string{
		{"Total", fmt.Sprintf("%.0f", totalJobs), "100.00%"},
		{"Successful", fmt.Sprintf("%.0f", successfulJobs), fmt.Sprintf("%.2f%%", successRate*100)},
		{"Converted", fmt.Sprintf("%.0f", convertedJobs), fmt.Sprintf("%.2f%%", conversionRate*100)},
	}
	// Pass the raw flat object f as the third arg even though the IsJSON
	// branch above already handles --json — keeps the call shape uniform
	// with other Render call sites in the package.
	return cmd.Output.Render(funnelTableDef, rows, f)
}
