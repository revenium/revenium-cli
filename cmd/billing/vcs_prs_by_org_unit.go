package billing

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// init registers newVcsPrsByOrgUnitCmd() as a FLAT sibling on the package-level
// billing.Cmd (D-30-04), never as a child of `billing vcs-prs`.
//
// `billing vcs-prs` is a shipped cobra.NoArgs leaf (vcs_prs.go:28). Hanging a
// by-org-unit child under it would convert a released leaf into a parent, which
// changes how a bare `billing vcs-prs` invocation resolves its arguments — a
// behaviour change on a documented, scripted-against command. That is the move
// D-28-02 refused for `metrics` under `jobs outcome` and D-29-01 re-refused for
// `summary` under `jobs roi`.
//
// The hyphenated name is the endpoint's own trailing path segment
// (/v2/api/billing/users/vcs-prs/by-org-unit), which is the naming convention
// every cmd/billing verb already follows.
func init() { Cmd.AddCommand(newVcsPrsByOrgUnitCmd()) }

// vcsPrsByOrgUnitTableDef is the ONE table definition that serves BOTH the
// grouped and the ungrouped form of this report.
//
// This is deliberate, not an economy. The spec declares a single row schema,
// VcsPrsByOrgUnitRow_Read, for both modes: without groupBy the server returns
// "the plain workspace daily total (one row per UTC calendar day)" as that same
// row with orgUnitId and orgUnitName declared NULL — not absent, and not a
// different shape. The (day, org-unit) row is therefore the primary identity
// and the plain daily total is one variant of it.
//
// A second TableDef for the ungrouped case would fork in this CLI a row
// identity the server deliberately kept unified, and it would break the
// reconciliation the description says the Unassigned bucket exists to preserve:
// grouped totals are meant to add up to the ungrouped total precisely because
// they are the same rows. TestByOrgUnitUngrouped and TestByOrgUnitGrouped
// assert against this one definition and one row builder for that reason.
var vcsPrsByOrgUnitTableDef = output.TableDef{
	Headers: []string{
		"Date",
		"Org Unit",
		"Org Unit ID",
		"Pull Requests",
	},
	StatusColumn: -1,
}

// newVcsPrsByOrgUnitCmd returns `revenium billing vcs-prs-by-org-unit` (GET
// /v2/api/billing/users/vcs-prs/by-org-unit, single-object report — the
// raw-Do-into-map idiom cmd/billing/vcs_prs.go establishes for this package).
//
// This command declares NO cobra pre-run hook of any kind. The hook at
// cmd/jobs/roi_summary.go:149-190 repoints the shared client at the analytics
// host with bearer auth, process-wide and without restore. Billing lives on the
// platform host with x-api-key (billing.go:18-21), so copying that block here
// would send a platform API key to another host as a bearer token (T-30-04).
func newVcsPrsByOrgUnitCmd() *cobra.Command {
	var (
		from               string
		to                 string
		groupBy            string
		orgUnitID          int64
		includeDescendants bool
	)

	c := &cobra.Command{
		Use:   "vcs-prs-by-org-unit",
		Short: "Get the daily merged-PR series, optionally grouped by department",
		Args:  cobra.NoArgs,
		Example: `  # Get the plain workspace daily total of merged PRs
  revenium billing vcs-prs-by-org-unit --from 2026-03-15 --to 2026-04-14

  # Break the same series down by department
  revenium billing vcs-prs-by-org-unit --from 2026-03-15 --to 2026-04-14 --group-by orgUnit

  # Get the full response as JSON
  revenium billing vcs-prs-by-org-unit --from 2026-03-15 --to 2026-04-14 --json`,
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
			//     concatenation. This command has the phase's largest
			//     operator-controlled query surface and --group-by is a
			//     free-form string with no closed enum in front of it, so an
			//     operator value carrying & or = would otherwise append a
			//     SECOND query key to an authenticated request (T-30-01).
			//     TestByOrgUnitQueryIsEncoded asserts exactly that.
			qs := url.Values{}

			// Both dates are required: true in the spec and marked required
			// below, so the changed-bit gate does not apply to them: the sets
			// are unconditional. Gating a required parameter would create a
			// path on which it can go missing.
			qs.Set("startDate", from)
			qs.Set("endDate", to)

			// The three optional parameters, each behind its own changed-bit
			// gate (the shape cmd/jobs/roi_summary.go:227-247 establishes).
			//
			// The gate keeps an unpassed flag out of the query ENTIRELY rather
			// than sending it as an empty value — an empty groupBy is not the
			// same request as no groupBy. It also keeps this endpoint's row
			// count in the drift audit's field report honest: a parameter the
			// spec declares and the CLI never sends is a spec-only field row on
			// an endpoint the audit reports as covered
			// (roi_summary.go:239-241).
			if c.Flags().Changed("group-by") {
				qs.Set("groupBy", groupBy)
			}
			if c.Flags().Changed("org-unit-id") {
				// The only non-string query parameter in this phase. url.Values
				// needs a string, and FormatInt is what produces DIGITS: routing
				// an int64 through the unformatted print family would emit
				// scientific notation for an id at or above 1e6. The KEY stays a
				// string literal regardless of how the value is produced.
				qs.Set("orgUnitId", strconv.FormatInt(orgUnitID, 10))
			}
			if c.Flags().Changed("include-descendants") {
				// Sent whenever the flag was set, with NO client-side interlock
				// on --org-unit-id. The spec states the server ignores this
				// parameter when orgUnitId is omitted; that rule is the
				// server's, the flag's usage string records it, and a local
				// refusal would be a second copy of a server rule that can
				// drift away from it.
				qs.Set("includeDescendants", strconv.FormatBool(includeDescendants))
			}

			// Decoded as a map rather than a typed struct so a server field
			// this code does not know about still round-trips through --json.
			// Do, not DoList: this operation returns a single object, not an
			// embedded-items envelope. VcsPrsByOrgUnitResponse_Read declares no
			// _embedded and no _links, so the package's HAL unwrap helper does
			// not apply either.
			path := "/v2/api/billing/users/vcs-prs/by-org-unit" + "?" + qs.Encode()
			var resource map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resource); err != nil {
				return err
			}
			return renderVcsPrsByOrgUnit(resource)
		},
	}

	// The dates are passed through UNPARSED and UNREFORMATTED, exactly as the
	// cmd/jobs and cmd/billing date flags are. The usage strings are the spec's
	// own parameter descriptions, so --help answers the format question rather
	// than leaving the operator to guess it.
	//
	// There is deliberately NO client-side day-count check against the cap the
	// operation description states for this range. A cap copied into the client
	// goes stale exactly the way a copied enum does — it starts refusing a
	// window the API would have accepted, and the refusal reads as a CLI bug
	// with no provenance. Nothing in .planning/REQUIREMENTS.md asks for one
	// here (unlike TEAM-09, where the ask is explicit), and the server's own
	// 400 names the actual limit it enforced, which no message composed here
	// could promise to still be true. So an over-wide range reaches the server
	// and the server's message is relayed unchanged through the shared client
	// error path. This is a choice, not an omission.
	c.Flags().StringVar(&from, "from", "", "Start date (ISO yyyy-MM-dd)")
	c.Flags().StringVar(&to, "to", "", "End date (ISO yyyy-MM-dd)")
	// Both parameters are required: true in the spec. The discarded-error form
	// is repo-wide (cmd/billing/seats.go, cmd/tools/create.go:57-59).
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")

	c.Flags().StringVar(&groupBy, "group-by", "", "Return one row per (day, department) instead of the plain daily total; only 'orgUnit' is supported by the server")
	c.Flags().Int64Var(&orgUnitID, "org-unit-id", 0, "Scope the result to a single department, which must belong to the caller's organization")
	c.Flags().BoolVar(&includeDescendants, "include-descendants", false, "Also include the descendant departments of --org-unit-id; the server ignores this when --org-unit-id is omitted")

	return c
}

// renderVcsPrsByOrgUnit renders the daily merged-PR series.
//
// It opens with exactly ONE output-mode branch. Nothing below that line may
// consult output mode a second time — calling Render once per section would
// emit two JSON documents, and a nil payload on one of them would emit a JSON
// null. The resource is never mutated before being handed to Render, so --json
// is the server's document verbatim.
func renderVcsPrsByOrgUnit(resource map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.Render(vcsPrsByOrgUnitTableDef, nil, resource)
	}

	// The Formatter's writer is io.Discard in quiet mode, so the two lines
	// written to it below need no extra branch of their own.
	w := cmd.Output.Writer()

	// The server's own total, not one this CLI summed from the rows. It goes
	// through num() rather than str() because encoding/json decodes every JSON
	// number into a float64 and the unformatted print family emits scientific
	// notation at or above 1e6 — a merged-PR total for a large organization is
	// squarely in that range. num() also keeps a total the server did not
	// declare distinguishable from a measured zero.
	fmt.Fprintf(w, "Total pull requests: %s\n", num(resource, "totalPullRequests", "%.0f"))

	rows := objectsAt(resource, "rows")
	if len(rows) == 0 {
		// Table-path-only empty state. The arm above has already returned the
		// complete server document, which carries its own typed empty array. A
		// window with no merged pull requests is a success, not an error, and
		// the total the server DID send has already printed above.
		_, err := fmt.Fprintln(w, "No merged pull requests are reported for this window.")
		return err
	}

	// Deterministic row order. The spec states no ordering for rows, so sorting
	// here makes row order a property of OUR output rather than an assumption
	// about a server contract that was never made.
	//
	// The key is date FIRST and org-unit name second. A date-only key would be
	// enough for the ungrouped form, where there is one row per UTC day — but
	// grouped mode returns several rows per date, and a date-only key would
	// leave those siblings in whatever order the server happened to emit,
	// which can differ between two runs on the same window.
	//
	// SliceStable, not Slice: two rows comparing equal on BOTH keys keep the
	// server's relative order rather than an arbitrary one that can change
	// between runs on the same input.
	//
	// This runs entirely BELOW the single output-mode branch at the top, on the
	// local slice objectsAt built — never on the resource handed to Render. A
	// sort above that branch would mean --json stops being the server's
	// document in the server's order, which is that flag's whole contract.
	//
	// str() supplies the date key so a missing or non-string date sorts as an
	// empty string rather than panicking; date is the one cell in this table
	// that is genuinely a non-nullable string. orgUnitName is declared
	// ["string","null"], so it comes through nullableStr() — the same value the
	// row builder renders, which keeps the sort key and the visible cell from
	// ever disagreeing.
	sort.SliceStable(rows, func(i, j int) bool {
		di, dj := str(rows[i], "date"), str(rows[j], "date")
		if di != dj {
			return di < dj
		}
		return nullableStr(rows[i], "orgUnitName") < nullableStr(rows[j], "orgUnitName")
	})

	return cmd.Output.RenderTable(vcsPrsByOrgUnitTableDef, vcsPrsByOrgUnitRows(rows))
}

// vcsPrsByOrgUnitRows builds the table rows for BOTH modes from the one row
// schema the spec declares.
//
// orgUnitId goes through num() with an EXPLICIT verb, never through str(): it
// is an int64 that encoding/json has already turned into a float64, so str()
// would render a seven-digit department id as scientific notation. It also does
// not go through the floatVal/formatCount pair in billing.go, which would
// render the null that ungrouped mode always sends as a confident 0 —
// indistinguishable from a department whose id genuinely is 0 (T-30-03).
//
// orgUnitName goes through nullableStr() for the same reason in the string
// direction: a null name and a name the server sent as an empty string are
// different facts.
//
// The server's "Unassigned" label is passed through untouched. The description
// states that users who never resolve to a department are bucketed under it
// rather than dropped, so that grouped totals still reconcile against the
// ungrouped total; renaming or hiding that bucket makes a reconciliation
// silently fail to add up. A genuinely null orgUnitName — what the ungrouped
// mode sends — renders as the undeclared marker instead, which is a different
// fact in a different cell.
func vcsPrsByOrgUnitRows(rows []map[string]interface{}) [][]string {
	built := make([][]string, 0, len(rows))
	for _, r := range rows {
		built = append(built, []string{
			str(r, "date"),
			nullableStr(r, "orgUnitName"),
			num(r, "orgUnitId", "%.0f"),
			num(r, "pullRequests", "%.0f"),
		})
	}
	return built
}
