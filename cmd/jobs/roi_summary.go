package jobs

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// init registers newROISummaryCmd() onto the package-level jobs.Cmd.
// Per Phase 12 D-21 and Go's multi-init-per-package semantics, this composes
// with the init()s in jobs.go (list+get), create.go, update.go, delete.go,
// outcome.go, types.go, roi.go, and conversion_funnel.go.
//
// D-29-01: a FLAT sibling of `roi`, never a child of it. `jobs roi
// <agenticJobId>` is a shipped ExactArgs(1) leaf (roi.go:37-41) and hanging a
// child under it would convert a released leaf into a parent — the move
// D-28-02 refused.
func init() {
	Cmd.AddCommand(newROISummaryCmd())
}

// roiSummaryScalarsTableDef is the 2-column key-value layout for the Summary
// section. Headers match roiTableDef (roi.go:24-27), the sibling ROI command —
// deliberately NOT economicsScalarsTableDef, whose first header is "Setting",
// which is the wrong word for a measurement and would couple two renders that
// have no reason to move together (RESEARCH D2).
// StatusColumn: -1 disables status colorization — no Value cell here carries
// status semantics.
var roiSummaryScalarsTableDef = output.TableDef{
	Headers:      []string{"Metric", "Value"},
	StatusColumn: -1,
}

// roiSummaryByJobTypeTableDef is the 8-column per-job-type layout.
//
// Two header choices are deliberate:
//
//  1. "Jobs" rather than "Total Jobs": the Summary block above already carries
//     a "Total Jobs" row for the whole period, and this column is per-type, so
//     the longer label here would be actively misleading (RESEARCH D5).
//  2. "ROI" BARE, with no percent sign. D-29-10 forbids asserting a unit the
//     schema does not state, and a header naming a unit while the cell carries
//     none is exactly the wrong-but-plausible presentation D-26-01 rejects.
//     RESEARCH D5 suggested a percent-suffixed header; it is overridden here
//     on D-29-09's own header list and D-29-10's reasoning.
//
// StatusColumn: -1 — job type names are not status tokens.
var roiSummaryByJobTypeTableDef = output.TableDef{
	Headers: []string{
		"Job Type", "Jobs", "Total Cost", "Total Value",
		"ROI", "Success Rate", "Cost/Conversion", "Avg Value",
	},
	StatusColumn: -1,
}

// validMetricTypes is the exact ten-value set the operation's own metricType
// query parameter declares, in the document's order, read from
// .cache/openapi/revenium-analytics-api.json (GET
// /api/v2/analytics/jobs/roi-summary).
//
// It is deliberately FILE-LOCAL. Nothing else in this repo declares this enum;
// validDimensionNames in cmd/metrics is file-local for the same reason, and
// putting this under internal/ would invent a shared owner that does not exist
// (RESEARCH D4).
//
// D-27-11: the set is copied from the spec, so if the server ever widens the
// enum this check goes stale and starts refusing a value the API accepts. That
// failure should read as OUR list being out of date — which is why the source
// document and the parameter it came from are named right here — rather than as
// a mystery refusal with no provenance.
var validMetricTypes = []string{
	"sum", "avg", "max", "min", "count",
	"median", "p50", "p90", "p95", "p99",
}

// validateMetricType refuses a --metric-type outside the declared enum before
// any request object exists (T-29-04 mitigation), mirroring
// validateDimensionName in cmd/metrics/dimensions.go:26-34.
//
// It is wired to PreRunE rather than to an args validator: this command takes
// no positional argument at all (D-29-02), so there is no args[0] to inspect,
// and validating a flag from an args validator reads oddly. Cobra runs
// PersistentPreRunE, then PreRunE, then RunE — so a refusal here lands after
// the host swap and before the path is built, which is what makes "zero HTTP
// calls for an invalid value" true by construction rather than by luck.
//
// An absent flag is not an invalid one: the parameter is optional in the spec,
// so the check is skipped entirely unless the operator actually passed it. An
// explicitly-empty value IS passed, and an empty string is not a member of the
// set, so it is refused.
//
// The error is a plain formatted one with no semantic exit code. It lands on
// the general exit code exactly as the shipped dimension validator does; the
// three sibling cmd/jobs commands that map status codes do so for write paths,
// and this is a read (RESEARCH D6).
func validateMetricType(c *cobra.Command, _ []string) error {
	f := c.Flag("metric-type")
	if f == nil || !f.Changed {
		return nil
	}
	value := f.Value.String()
	for _, v := range validMetricTypes {
		if value == v {
			return nil
		}
	}
	return fmt.Errorf("metric type %q is not valid (expected one of: %s)", value, strings.Join(validMetricTypes, ", "))
}

// newROISummaryCmd builds `revenium jobs roi-summary` — JOBS-18.
//
// Issues GET /api/v2/analytics/jobs/roi-summary against the ANALYTICS host and
// renders two labelled sections: a Summary key-value block, then a By Job Type
// table (D-29-01, D-29-09).
//
// Its PersistentPreRunE runs the root hook first, then swaps the shared
// cmd.APIClient onto the configured analytics base URL with bearer auth
// (D-01/D-04/D-05, D-29-04) — scoped to this command only, so every sibling
// `jobs` leaf command stays on the platform host with x-api-key auth.
//
// Per CF-17 / D-29-02, aggregate verbs take no positional argument at all —
// there is no id segment on this operation.
func newROISummaryCmd() *cobra.Command {
	var (
		from       string
		to         string
		jobType    string
		metricType string
	)

	c := &cobra.Command{
		Use:   "roi-summary",
		Short: "View aggregate ROI summary across job types",
		Args:  cobra.NoArgs, // CF-17 / D-29-02 — aggregate verb, no id segment
		Example: `  # View the aggregate ROI summary across all job types
  revenium jobs roi-summary

  # Filter by date range, job type and aggregation
  revenium jobs roi-summary --from 2025-09-01T00:00:00Z --to 2025-09-30T23:59:59Z --job-type loan-processing --metric-type avg

  # As JSON for scripting
  revenium jobs roi-summary --json`,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			// Run the root PersistentPreRunE first (config/API client init).
			// Guard against self-delegation: when this command is executed
			// unattached (as in unit tests that call
			// newROISummaryCmd().Execute() directly without registering it
			// under jobs.Cmd/rootCmd), c.Root() returns c itself, and the
			// third clause of the condition below is what stops it: calling
			// the command's own PersistentPreRunE from inside itself would
			// recurse infinitely (stack overflow) since root.PersistentPreRunE
			// is this exact closure. This is the identical defect SAFE-02
			// closed in Phase 26, and D-29-06 makes the guard non-optional:
			// every unit test in this file executes the command unattached.
			if root := c.Root(); root != nil && root != c && root.PersistentPreRunE != nil {
				if err := root.PersistentPreRunE(c, args); err != nil {
					return err
				}
			}
			// Analytics endpoints live on a distinct, independently configured
			// host with Authorization: Bearer auth and no teamId/tenantId
			// params (D-01/D-04/D-05) — never derived by string-rewriting
			// BaseURL (Pitfall B6).
			//
			// D-29-04: this swap stays INLINE in this file. Surface
			// attribution in tools/coverage-audit is by source file, so the
			// file that repoints the client and the file whose call sites are
			// attributed to the analytics surface must be the same file. It is
			// deliberately not extracted into a shared helper and deliberately
			// not attached to the package-level jobs.Cmd, which would put every
			// cmd/jobs command on the analytics host.
			//
			// D-29-05 (accepted, recorded, NOT fixed): the mutation is
			// process-wide with no restore, so every later command executed in
			// the same process also talks to the analytics host with bearer
			// auth. Accepted because production runs one command per process;
			// the repair is deferred (29-CONTEXT.md § Deferred Ideas). Tests
			// must therefore construct a fresh client per command execution.
			if cmd.APIClient != nil {
				cmd.APIClient.BaseURL = cmd.APIClient.AnalyticsBaseURL
				cmd.APIClient.UseBearerAuth = true
			}
			return nil
		},
		PreRunE: validateMetricType,
		RunE: func(c *cobra.Command, args []string) error {
			// D-29-03: this command deliberately does NOT call requireTeam().
			// Six sibling cmd/jobs commands do, so a reviewer scanning the
			// package would otherwise read this as an oversight. The analytics
			// spec declares no team or tenant parameter across any of its
			// operations, and bearer auth suppresses the teamId/tenantId
			// injection at internal/api/client.go:74 regardless — so a team
			// guard here would refuse a request that needs no team.

			// Build the optional query parameters. CRITICAL: the keys
			// handed to the setter below are the OAS field names
			// (startDate/endDate/jobType/metricType), NOT the CLI flag names
			// (the rule conversion_funnel.go:64-66 already states).
			//
			// Two properties of this shape are load-bearing and neither is
			// cosmetic:
			//
			//  1. Every parameter name is a STRING LITERAL. The drift audit's
			//     field extractor reads these calls by AST
			//     (tools/coverage-audit/fields.go:1234-1251) and a name it
			//     cannot resolve to a literal is a FATAL extraction error, not
			//     a warning — a name built from a variable, a constant or a
			//     range over a map takes the whole audit down.
			//  2. Encoding goes through url.Values, never string
			//     concatenation. An operator-supplied --job-type carrying & or
			//     = would otherwise inject a second query key onto a request
			//     that reaches a different host with a bearer token on it
			//     (T-29-04).
			//
			// The changed-bit gate is what keeps an unpassed flag out of the
			// query entirely rather than sending it as an empty value. That is
			// also what keeps this endpoint's row count in the audit's field
			// report at zero: a declared parameter the CLI never sends is a
			// spec-only field row on an endpoint the audit reports as covered
			// (D-29-11, RESEARCH Pitfall 5).
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
			if c.Flags().Changed("metric-type") {
				qs.Set("metricType", metricType)
			}

			// The analytics spec's own paths already carry the full
			// /api/v2/... prefix: no platform prefix and no path builder here
			// (the reason dimensions.go:82-84 records).
			path := "/api/v2/analytics/jobs/roi-summary"
			if len(qs) > 0 {
				path += "?" + qs.Encode()
			}

			// Decoded as a map rather than a typed struct so a server field
			// this code does not know about still round-trips through --json
			// (the reason economics_render.go:63-66 states). Do, not DoList:
			// this operation returns a single object, not an embedded-items
			// envelope.
			var resource map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resource); err != nil {
				return err
			}
			return renderROISummary(resource)
		},
	}

	// All four query parameters this operation declares are optional in the
	// spec, so none of them is marked required here.
	//
	// --from, --to and --job-type deliberately reuse the names `jobs
	// conversion-funnel` already carries (conversion_funnel.go:94-98): an
	// operator must not have to learn two names for one concept inside one
	// command group. --metric-type is the fourth and is new to this command;
	// its usage string names the ten legal values so `--help` answers the
	// question before the operator has to guess it.
	//
	// The dates are passed through UNPARSED and UNREFORMATTED, exactly as that
	// sibling does. The metrics package carries an unexported helper that
	// appends a trailing Z, and reaching it would mean moving or duplicating
	// shipped code for one requirement — the blast radius D-29-05 declined.
	// Two cmd/jobs commands sharing a flag name must behave identically
	// (RESEARCH D3).
	c.Flags().StringVar(&from, "from", "", "Filter start date (ISO 8601)")
	c.Flags().StringVar(&to, "to", "", "Filter end date (ISO 8601)")
	c.Flags().StringVar(&jobType, "job-type", "", "Filter by job type")
	c.Flags().StringVar(&metricType, "metric-type", "", "Aggregation to apply (one of: sum, avg, max, min, count, median, p50, p90, p95, p99)")

	return c
}

// renderROISummary renders the two labelled sections.
//
// It opens with exactly ONE output-mode branch, following
// economics_render.go:55-98. Calling Render once per section would emit two
// separate JSON documents in JSON mode, and passing a nil payload on one of
// them would emit a JSON null — which is why the branch is at the top and
// nothing below it may consult output mode a second time. Routing the JSON arm
// through Render rather than the JSON-only renderer keeps --fields filtering
// working exactly as it does everywhere else in the repo.
//
// The resource is never mutated before being handed to Render, so `--json` is
// the server's document verbatim.
//
// CRITICAL (the rule conversion_funnel.go:110-114 already states by name): the
// Overall ROI, ROI and Success Rate cells use the SERVER-supplied values. The
// CLI must NOT recompute or re-derive them from the raw counts — the server's
// values are the source of truth and match the dashboard, and re-deriving them
// client-side produces a number that silently disagrees with the product.
func renderROISummary(resource map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.Render(roiSummaryScalarsTableDef, nil, resource)
	}

	// Sections render through RenderTable directly, which already returns
	// without output in quiet mode (internal/output/table.go). Headings go to
	// the Formatter's writer, which is io.Discard in quiet mode.
	w := cmd.Output.Writer()

	fmt.Fprintln(w, "Summary")
	if err := cmd.Output.RenderTable(roiSummaryScalarsTableDef, roiSummaryScalarRows(resource)); err != nil {
		return err
	}

	fmt.Fprintln(w, "By Job Type")
	byJobType := objectsAt(resource, "byJobType")
	if len(byJobType) == 0 {
		// Table-path-only empty state (renderMetricsSection,
		// economics_render.go:154-160). The arm above has already returned the
		// complete server document, which carries its own typed empty array.
		_, err := fmt.Fprintln(w, "No job types are reported for this period.")
		return err
	}

	// Deterministic row order (D-29-10, D-27-08). The spec states no ordering
	// for byJobType, so sorting here makes row order a property of OUR output
	// rather than an assumption about a server contract that was never made.
	//
	// Ascending by job type, deliberately NOT descending by ROI: ranking would
	// make row order an editorial claim about which job type matters, and it
	// would key that claim on the one field whose unit convention is unsettled.
	//
	// This runs entirely BELOW the single output-mode branch at the top, on the
	// local slice objectsAt built — never on the resource handed to Render. A
	// sort above that branch would mean --json stops being the server's
	// document in the server's order, which is that flag's whole contract
	// (RESEARCH Pitfall 9).
	//
	// str() supplies the key so a missing or non-string jobType sorts as an
	// empty string rather than panicking; it is the one cell in this table that
	// is genuinely a string (format.go:21-33).
	sort.SliceStable(byJobType, func(i, j int) bool {
		return str(byJobType[i], "jobType") < str(byJobType[j], "jobType")
	})
	return cmd.Output.RenderTable(roiSummaryByJobTypeTableDef, roiSummaryByJobTypeRows(byJobType))
}

// roiSummaryScalarRows builds the Summary section.
//
// Row order is INTENTIONALLY an explicit ordered slice literal. A for/range
// over the response map would produce non-deterministic Go map iteration order
// — the pitfall roi.go:66-70 and economics_render.go:99-104 already record.
//
// period.start and period.end are plain "type": "string" in the schema with no
// format, so they are rendered as-is: not parsed, not reformatted, not
// validated. The server's rendering is the contract.
//
// Total Cost and Total Value use the literal "USD" because NO currency field
// appears anywhere in this operation — the same accepted v1 risk roi.go:72-76
// already states, not a new one (D-29-10).
func roiSummaryScalarRows(resource map[string]interface{}) [][]string {
	period, _ := objectAt(resource, "period")
	summary, _ := objectAt(resource, "summary")

	// An entirely undeclared period renders as the undeclared marker rather
	// than as a confident empty range.
	periodCell := undeclared
	if start, end := str(period, "start"), str(period, "end"); start != "" || end != "" {
		periodCell = start + " → " + end
	}

	return [][]string{
		{"Period", periodCell},
		{"Total Job Types", num(summary, "totalJobTypes", "%.0f")},
		{"Total Jobs", num(summary, "totalJobs", "%.0f")},
		{"Total Cost", money(summary, "totalCost", "USD")},
		{"Total Value", money(summary, "totalValue", "USD")},
		// TODO(D-29-10): the schema types overallROI as a bare number and says
		// nothing more, and this repo contains BOTH conventions —
		// conversion_funnel.go's rate is a 0-1 decimal multiplied by 100 for
		// display, while roi.go's value is already a percentage. Settling it
		// needs one real response from the analytics host. Until then: no
		// suffix, no multiplication.
		{"Overall ROI", num(summary, "overallROI", "%.2f")},
	}
}

// roiSummaryByJobTypeRows builds one 8-cell row per byJobType item, in
// roiSummaryByJobTypeTableDef's header order.
//
// Every numeric cell goes through the package's money() or num() helper: both
// distinguish "not declared" (—) from "declared as zero" ($0.00), and both take
// a mandatory format verb because the unformatted print variant emits
// scientific notation at or above 1e6. str() is for strings and booleans ONLY,
// so it is used for jobType and for nothing else here (format.go:21-33 records
// both traps).
//
// TODO(D-29-10): the roi and successRate cells carry the same unsettled unit
// question as Overall ROI above — rendered as a bare %.2f with no suffix and no
// multiplication until one real response settles the convention.
func roiSummaryByJobTypeRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, 0, len(items))
	for _, m := range items {
		rows = append(rows, []string{
			str(m, "jobType"),
			num(m, "totalJobs", "%.0f"),
			money(m, "totalCost", "USD"),
			money(m, "totalValue", "USD"),
			num(m, "roi", "%.2f"),
			num(m, "successRate", "%.2f"),
			money(m, "costPerConversion", "USD"),
			money(m, "averageValue", "USD"),
		})
	}
	return rows
}
