package billing

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newVcsPrHealthCmd()) }

// validVcsSources is the exact two-value set the operation's own source query
// parameter accepts, read from .cache/openapi-dev/revenium-platform-api.json
// (the source parameter of GET /v2/api/billing/users/vcs-pr-health).
//
// The parameter is typed there as a bare "string" with NO enum key, so the two
// legal values live only in the parameter's prose description ("VCS source:
// github or gitlab"). This slice is that prose, made executable.
//
// It is deliberately FILE-LOCAL, exactly as validMetricTypes in
// cmd/jobs/roi_summary.go and validDimensionNames in cmd/metrics/dimensions.go
// are. Nothing else in this repo declares this enum, and hoisting it under
// internal/ would invent a shared owner that does not exist.
//
// D-27-11, applied here: the set is copied from the spec, so if the server ever
// widens it this check goes stale and starts refusing a value the API accepts.
// That failure should read as OUR list being out of date — which is why the
// source document and the parameter it came from are named right here — rather
// than as a mystery refusal with no provenance.
var validVcsSources = []string{"github", "gitlab"}

// validateVcsSource refuses a --source outside the declared set before any
// request object exists (T-30-02 mitigation).
//
// It is wired to PreRunE rather than to an args validator: this command takes
// no positional argument, so there is no args[0] to inspect. Cobra runs
// PreRunE before ValidateRequiredFlags and before RunE, so a refusal here costs
// zero HTTP calls by construction rather than by luck — which is what
// TestPrHealthRejectsSourceOutsideEnum's request-counter assertion pins.
//
// Unlike validateMetricType in cmd/jobs/roi_summary.go, this validator carries
// NO !f.Changed early return. That parameter is optional in its spec; this one
// is required: true, so MarkFlagRequired guarantees presence and an early
// return would create a path where an unset source silently passes validation.
// The cost is that omitting --source lands on this refusal rather than on
// cobra's required-flag message; both are refusals at zero HTTP calls.
//
// The error is a plain formatted one with no semantic exit code, exactly as the
// two shipped enum validators are: this is a read path.
func validateVcsSource(c *cobra.Command, _ []string) error {
	f := c.Flag("source")
	if f == nil {
		return fmt.Errorf("source flag is not registered")
	}
	value := f.Value.String()
	for _, v := range validVcsSources {
		if value == v {
			return nil
		}
	}
	return fmt.Errorf("source %q is not valid (expected one of: %s)", value, strings.Join(validVcsSources, ", "))
}

// prHealthTotalsTableDef defines the organization roll-up section.
//
// "Avg / Merged PR" deliberately names no currency. avgCostPerMergedPr is
// declared ["number","null"] and the schema states no currency anywhere, so a
// header carrying a currency symbol would assert a unit the document does not.
// That is the same call Phase 29 made when it shipped a bare "ROI" header
// rather than a percent-suffixed one (D-29-10).
//
// The two threshold columns are the SERVER's echo of the team's configured
// thresholds, not anything this CLI computed or compared against.
var prHealthTotalsTableDef = output.TableDef{
	Headers: []string{
		"Open PRs",
		"Draft PRs",
		"Aging PRs",
		"Rotting PRs",
		"Rotting (assisted)",
		"Closed Unmerged",
		"Closed Unmerged (assisted)",
		"Avg / Merged PR",
		"Aging Threshold (days)",
		"Rotting Threshold (days)",
		"Last Synced",
	},
	StatusColumn: -1,
}

// prHealthEngineerTableDef defines the per-engineer section — the rows BILL-05
// names: aging and rotting open PRs, classified server-side by inactivity, and
// PRs closed without merging inside the requested window.
var prHealthEngineerTableDef = output.TableDef{
	Headers: []string{
		"Engineer",
		"Email",
		"Open PRs",
		"Aging PRs",
		"Rotting PRs",
		"Closed Unmerged",
		"Oldest Inactive (days)",
	},
	StatusColumn: -1,
}

// newVcsPrHealthCmd returns `revenium billing vcs-pr-health` (GET
// /v2/api/billing/users/vcs-pr-health, single-object report — the
// raw-Do-into-map idiom cmd/billing/vcs_prs.go establishes for this package).
//
// It is registered as a flat sibling on billing.Cmd (D-30-04), never as a child
// of vcs-prs: that command is a shipped leaf and turning it into a parent would
// change the meaning of an already-published invocation.
//
// This command declares NO cobra pre-run hook beyond PreRunE above, and
// specifically not the persistent variant. The hook at
// cmd/jobs/roi_summary.go:149-190 repoints the shared client at the analytics
// host with bearer auth, process-wide and without restore. Billing lives on the
// platform host with x-api-key (billing.go:18-21), so copying that block here
// would send a platform API key to another host as a bearer token (T-30-04).
func newVcsPrHealthCmd() *cobra.Command {
	var (
		source string
		from   string
		to     string
	)

	c := &cobra.Command{
		Use:   "vcs-pr-health",
		Short: "Get per-engineer PR health for a VCS source",
		Args:  cobra.NoArgs,
		Example: `  # Get PR health for GitHub over a date range
  revenium billing vcs-pr-health --source github --from 2026-05-17 --to 2026-08-17

  # Get the full response, including the oldest open-PR detail, as JSON
  revenium billing vcs-pr-health --source github --from 2026-05-17 --to 2026-08-17 --json`,
		PreRunE: validateVcsSource,
		RunE: func(c *cobra.Command, args []string) error {
			// CRITICAL: the keys handed to the setter below are the OAS
			// parameter names, NOT the CLI flag names. This operation uses the
			// repo-standard startDate/endDate spelling; /v2/api/billing/seats
			// is the outlier that spells the same concept fromDate/toDate. A
			// reader moving between the two files must not "correct" one to
			// match the other — the resulting 400 reads like a date-format
			// problem rather than a wrong parameter name.
			//
			// Two properties of this shape are load-bearing and neither is
			// cosmetic:
			//
			//  1. Every parameter name is a STRING LITERAL. The drift audit's
			//     field extractor reads these calls by AST
			//     (tools/coverage-audit/fields.go, handleSet) and a name it
			//     cannot resolve to a literal is a FATAL extraction error, not
			//     a warning — a name built from a variable, a constant or a
			//     range over a map takes the whole audit down.
			//  2. Encoding goes through url.Values, never string
			//     concatenation. An operator-supplied value carrying & or =
			//     would otherwise append a second query key to an
			//     authenticated request (T-30-01).
			//
			// All three parameters are required: true in the spec and are
			// marked required below, so the changed-bit gate roi_summary.go
			// uses for its optional params does not apply: the sets are
			// unconditional.
			qs := url.Values{}
			qs.Set("source", source)
			qs.Set("startDate", from)
			qs.Set("endDate", to)

			// Decoded as a map rather than a typed struct so a server field
			// this code does not know about still round-trips through --json.
			// Do, not DoList: this operation returns a single object, not an
			// embedded-items envelope. VcsPrHealthResponse_Read declares no
			// _embedded and no _links, so the package's HAL unwrap helper does
			// not apply either.
			path := "/v2/api/billing/users/vcs-pr-health" + "?" + qs.Encode()
			var resource map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resource); err != nil {
				return err
			}
			return renderPrHealth(resource)
		},
	}

	// The dates are passed through UNPARSED and UNREFORMATTED, exactly as the
	// cmd/jobs and cmd/billing date flags are. The usage strings are the spec's
	// own parameter descriptions, so --help answers the format question rather
	// than leaving the operator to guess it.
	c.Flags().StringVar(&source, "source", "", "VCS source: github or gitlab")
	c.Flags().StringVar(&from, "from", "", "Start date (ISO yyyy-MM-dd) of the closed/merged window")
	c.Flags().StringVar(&to, "to", "", "End date (ISO yyyy-MM-dd), inclusive")
	// All three parameters are required: true in the spec. The discarded-error
	// form is repo-wide (cmd/billing/seats.go, cmd/tools/create.go:57-59).
	_ = c.MarkFlagRequired("source")
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")

	return c
}

// renderPrHealth renders the two labelled sections: the organization roll-up,
// then the per-engineer rows.
//
// It opens with exactly ONE output-mode branch. Nothing below that line may
// consult output mode a second time — calling Render once per section would
// emit two JSON documents, and a nil payload on one of them would emit a JSON
// null. The resource is never mutated before being handed to Render, so --json
// is the server's document verbatim.
//
// NO COMPUTED DOLLAR FIGURE IS RENDERED, deliberately. The operation
// description states that dollar estimates are computed CLIENT-SIDE as
// count x avgCostPerMergedPr, on an org-average basis, and labeled an estimate.
// That arithmetic would be this CLI inventing a number the server did not send,
// on a basis the schema itself declares nullable. The standing repo rule
// (cmd/jobs/roi_summary.go:296-303) is that this CLI never re-derives a number
// the server owns; here there is no server value at all, so any such cell would
// be editorial. A number this tool derived and a number the platform measured
// must never occupy the same kind of cell (T-30-03). The recorded decision is
// here so a later reader finds the reasoning rather than re-litigating it.
//
// The oldest[] detail list is likewise rendered as no third table. BILL-05
// names per-engineer PR health, inactivity-classified aging and rotting counts,
// and closed-without-merge counts — all of which the two sections below deliver
// in full. oldest[] is additional per-PR detail the requirement does not name,
// and it reaches the operator complete and unmodified through --json, which
// TestPrHealthJSONIsVerbatim asserts. This is a table-layout decision: it
// removes no data and no requirement, and the advisory line below names the
// flag that carries it.
func renderPrHealth(resource map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.Render(prHealthTotalsTableDef, nil, resource)
	}

	// The Formatter's writer is io.Discard in quiet mode, so the headings and
	// the advisory line below need no extra branch of their own.
	w := cmd.Output.Writer()

	fmt.Fprintln(w, "(use --json for the oldest open-PR detail)")

	fmt.Fprintln(w, "Totals")
	if err := cmd.Output.RenderTable(prHealthTotalsTableDef, prHealthTotalRows(resource)); err != nil {
		return err
	}

	fmt.Fprintln(w, "By Engineer")
	engineers := objectsAt(resource, "engineers")
	if len(engineers) == 0 {
		// Table-path-only empty state. The arm above has already returned the
		// complete server document, which carries its own typed empty array.
		// A window with no per-engineer rows is a success, not an error, and
		// the Totals section the server DID populate has already rendered.
		_, err := fmt.Fprintln(w, "No engineers are reported for this window.")
		return err
	}

	// Deterministic row order. The spec states no ordering for engineers, so
	// sorting here makes row order a property of OUR output rather than an
	// assumption about a server contract that was never made.
	//
	// SliceStable, not Slice: two engineers comparing equal on authorLogin keep
	// the server's relative order rather than an arbitrary one that can change
	// between runs on the same input.
	//
	// Ascending by login, deliberately NOT descending by rotting count: ranking
	// would make row order an editorial claim about which engineer matters.
	//
	// This runs entirely BELOW the single output-mode branch at the top, on the
	// local slice objectsAt built — never on the resource handed to Render. A
	// sort above that branch would mean --json stops being the server's
	// document in the server's order, which is that flag's whole contract.
	//
	// str() supplies the key so a missing or non-string authorLogin sorts as an
	// empty string rather than panicking; it is the one cell in this table that
	// is genuinely a non-nullable string.
	sort.SliceStable(engineers, func(i, j int) bool {
		return str(engineers[i], "authorLogin") < str(engineers[j], "authorLogin")
	})
	return cmd.Output.RenderTable(prHealthEngineerTableDef, prHealthEngineerRows(engineers))
}

// prHealthTotalRows builds the single Totals row.
//
// The two threshold cells come from the TOP-LEVEL resource, not from totals:
// the operation description states the thresholds "come from the team settings
// endpoint and are echoed here". The CLI renders that echo and performs no
// threshold comparison of its own — aging and rotting classify by INACTIVITY,
// server-side, against the org's own configuration. Nor does the CLI call the
// settings endpoint to fetch them: that is Phase 31's TEAM-08/TEAM-09.
//
// Note also that the requested date range does NOT filter the open-PR columns.
// The description states open-PR figures reflect the current synced state,
// while only the closed-without-merge counts and the cost basis are scoped to
// the window.
//
// A totals object that is absent or not an object yields a row of undeclared
// markers rather than a panic: num and nullableStr are both total on a nil map.
func prHealthTotalRows(resource map[string]interface{}) [][]string {
	totals, _ := resource["totals"].(map[string]interface{})

	// Every numeric cell goes through num(). None goes through str() (which
	// emits scientific notation at or above 1e6) and none goes through the
	// floatVal/formatCount pair in billing.go, which collapses a value the
	// server withheld and a value it measured as zero into the same cell —
	// avgCostPerMergedPr is declared nullable, so that distinction is real here
	// (T-30-03).
	return [][]string{{
		num(totals, "openPrs", "%.0f"),
		num(totals, "draftPrs", "%.0f"),
		num(totals, "agingPrs", "%.0f"),
		num(totals, "rottingPrs", "%.0f"),
		num(totals, "rottingPrsAssisted", "%.0f"),
		num(totals, "closedUnmerged", "%.0f"),
		num(totals, "closedUnmergedAssisted", "%.0f"),
		num(totals, "avgCostPerMergedPr", "%.2f"),
		num(resource, "agingDays", "%.0f"),
		num(resource, "rottingDays", "%.0f"),
		nullableStr(totals, "lastSyncedAt"),
	}}
}

// prHealthEngineerRows builds the per-engineer rows.
//
// mappedEmail is declared ["string","null"] and oldestInactiveDays
// ["integer","null"], so both route through the nullable helpers: an engineer
// the VCS login was never mapped to an email for renders the undeclared marker
// rather than an empty cell, and an engineer with no oldest inactive PR renders
// the marker rather than a confident 0 — which is a different fact from an
// engineer whose oldest open PR went inactive today.
//
// Every numeric cell goes through num(). None goes through str() (which emits
// scientific notation at or above 1e6) and none goes through the
// floatVal/formatCount pair in billing.go, which collapses absent and zero into
// one cell.
func prHealthEngineerRows(engineers []map[string]interface{}) [][]string {
	rows := make([][]string, 0, len(engineers))
	for _, e := range engineers {
		rows = append(rows, []string{
			str(e, "authorLogin"),
			nullableStr(e, "mappedEmail"),
			num(e, "openPrs", "%.0f"),
			num(e, "agingPrs", "%.0f"),
			num(e, "rottingPrs", "%.0f"),
			num(e, "closedUnmerged", "%.0f"),
			num(e, "oldestInactiveDays", "%.0f"),
		})
	}
	return rows
}
