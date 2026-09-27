package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// Every fixture in this file is written from the VERBATIM property sets in
// 30-RESEARCH.md § The API Contract → 3 — NOT from the rendering code. Writing
// a fixture from the code is Pitfall 2: the table then renders whatever the
// code already reads, the test is green, and the field names may not appear in
// any schema at all. That is the defect vcs_prs.go:45-46 shipped.
//
// The spec declares example values only for the five QUERY parameters; the
// response schema carries none. The response values below are therefore
// plausible rather than quoted, and they were chosen to be mutually
// distinguishable so an assertion on one cell cannot be satisfied by a digit
// that belongs to some other cell or to a date.

// byOrgUnitFixtureUngrouped is the response for a bare invocation with no
// groupBy: "the plain workspace daily total (one row per UTC calendar day)".
//
// Both org-unit columns are explicitly null on every row, because that is what
// the ungrouped form of VcsPrsByOrgUnitRow_Read is — the spec declares ONE row
// schema for both modes and simply nulls orgUnitId/orgUnitName here rather than
// omitting them. The render must therefore show the undeclared marker in those
// cells rather than dropping the columns, which is the invariant the plan's
// `promote` decision on the (day, org-unit) row identity encodes.
const byOrgUnitFixtureUngrouped = `{
	"groupBy": null,
	"rows": [
		{"date": "2026-03-15", "orgUnitId": null, "orgUnitName": null, "pullRequests": 12},
		{"date": "2026-03-16", "orgUnitId": null, "orgUnitName": null, "pullRequests": 9}
	],
	"totalPullRequests": 21
}`

// byOrgUnitFixtureGrouped is the response for --group-by orgUnit: "one row per
// (day, department) instead" of the plain daily total.
//
// The Unassigned row is the load-bearing one. The operation description states
// that users who never resolve to a department are bucketed under
// orgUnitId=null, orgUnitName="Unassigned" RATHER THAN DROPPED, so that grouped
// totals still reconcile against the ungrouped total. "Unassigned" is therefore
// a SERVER-SUPPLIED literal, not a placeholder the CLI is free to invent, to
// rename or to hide.
//
// The rows are supplied SHUFFLED on purpose, and each one pins a distinct
// property of the render:
//
//   - the spec states no ordering for rows, so the two output modes must
//     diverge — the table sorts, --json stays the server's document in the
//     server's order;
//   - Platform is dated a day LATER than the three 2026-03-15 rows while
//     sorting EARLIER than "Unassigned" by name, so a render keyed on name
//     alone puts it in the wrong place and this fixture catches it;
//   - the two Engineering rows are equal on BOTH sort keys and differ only in
//     pullRequests, which is what a stable sort must preserve and an unstable
//     one may not;
//   - Platform's orgUnitId is seven digits, which the unformatted print family
//     would render as scientific notation.
const byOrgUnitFixtureGrouped = `{
	"groupBy": "orgUnit",
	"rows": [
		{"date": "2026-03-16", "orgUnitId": 1234567, "orgUnitName": "Platform", "pullRequests": 41},
		{"date": "2026-03-15", "orgUnitId": null, "orgUnitName": "Unassigned", "pullRequests": 7},
		{"date": "2026-03-15", "orgUnitId": 204, "orgUnitName": "Engineering", "pullRequests": 777},
		{"date": "2026-03-15", "orgUnitId": 204, "orgUnitName": "Engineering", "pullRequests": 888}
	],
	"totalPullRequests": 1713
}`

// byOrgUnitCapture records what the stub server actually received. The *url.URL
// is what lets every test inspect the query KEY BY KEY rather than by matching
// a rendered query string, which is the only way to tell an absent optional
// parameter apart from one sent with an empty value.
type byOrgUnitCapture struct {
	url   *url.URL
	calls int
}

// byOrgUnitServer returns an httptest server serving body, plus the capture of
// the request it received.
func byOrgUnitServer(t *testing.T, body string) (*httptest.Server, *byOrgUnitCapture) {
	t.Helper()
	captured := &byOrgUnitCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.calls++
		captured.url = r.URL
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

// runByOrgUnit executes an unattached `billing vcs-prs-by-org-unit` against srv
// with args, in table mode (jsonMode=false) or JSON mode, and returns whatever
// it wrote plus the error it returned.
//
// The command is constructed directly rather than resolved through the tree,
// exactly as every other cmd/billing test does; registration itself is pinned
// separately in registration_test.go (D-30-04).
func runByOrgUnit(t *testing.T, srv *httptest.Server, jsonMode bool, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, jsonMode, false)

	c := newVcsPrsByOrgUnitCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs(args)
	err := c.Execute()
	return buf.String(), err
}

// TestByOrgUnitUngrouped pins the bare invocation: the exact path, the two
// required wire keys, and — the half that actually matters — that NONE of the
// three optional parameters appears on the query at all.
//
// The absence assertions use a two-value map lookup rather than a comparison
// against the empty string on purpose. `q.Get("groupBy") == ""` is satisfied by
// a parameter sent with an empty value, which is exactly the bug the
// changed-bit gate exists to prevent: a declared-but-never-sent parameter is
// also a spec-only field row in the drift audit (roi_summary.go:239-241).
//
// The render half asserts the ungrouped row identity: orgUnitId and
// orgUnitName are null on every row, and both must read as the undeclared
// marker rather than as an empty cell, a confident 0, or a dropped column.
func TestByOrgUnitUngrouped(t *testing.T) {
	srv, captured := byOrgUnitServer(t, byOrgUnitFixtureUngrouped)

	out, err := runByOrgUnit(t, srv, false, "--from", "2026-03-15", "--to", "2026-04-14")
	require.NoError(t, err)
	require.Equal(t, 1, captured.calls, "the command must issue exactly one GET")
	require.NotNil(t, captured.url)

	assert.Equal(t, "/v2/api/billing/users/vcs-prs/by-org-unit", captured.url.Path)

	q := captured.url.Query()
	assert.Equal(t, "2026-03-15", q.Get("startDate"))
	assert.Equal(t, "2026-04-14", q.Get("endDate"))

	// This endpoint spells its range startDate/endDate — fromDate/toDate
	// belongs to /v2/api/billing/seats alone, and getting it wrong here
	// produces a 400 that reads like a date-format problem.
	_, hasFromDate := q["fromDate"]
	assert.False(t, hasFromDate, "fromDate is billing seats' spelling, not this endpoint's")

	for _, key := range []string{"groupBy", "orgUnitId", "includeDescendants"} {
		_, ok := q[key]
		assert.Falsef(t, ok, "%s must be absent from the query entirely when its flag was never set, not present with an empty value", key)
	}

	assert.Contains(t, out, "2026-03-15")
	assert.Contains(t, out, "12", "the daily total must render")
	assert.GreaterOrEqual(t, strings.Count(out, undeclared), 4,
		"both org-unit cells on both ungrouped rows must render the undeclared marker")
}

// TestByOrgUnitGrouped pins --group-by orgUnit and, on the same fixture, every
// property of the grouped render that is only true together:
//
//   - the parameter reaches the wire and the two unset ones stay off it;
//   - the server's Unassigned bucket renders exactly as the server sent it —
//     renaming or hiding it would make a grouped total silently fail to
//     reconcile against the ungrouped total, which is the one property the
//     operation description says the bucket exists to preserve;
//   - table rows sort ascending by date FIRST and by org-unit name second;
//   - rows equal on both keys keep the server's relative order (stable sort);
//   - a seven-digit department id renders as digits, not as 1.234567e+06;
//   - and --json is still the server's document in the SERVER'S order, which is
//     what proves the sort runs below the single output-mode branch.
//
// Deliberately NOT written as t.Run subtests: 30-03-PLAN.md's verify gates count
// `--- PASS: TestByOrgUnit` lines and require an exact total, and go test emits
// an additional indented PASS line per subtest.
func TestByOrgUnitGrouped(t *testing.T) {
	srv, captured := byOrgUnitServer(t, byOrgUnitFixtureGrouped)

	out, err := runByOrgUnit(t, srv, false,
		"--from", "2026-03-15", "--to", "2026-04-14", "--group-by", "orgUnit")
	require.NoError(t, err)
	require.Equal(t, 1, captured.calls)
	require.NotNil(t, captured.url)

	q := captured.url.Query()
	assert.Equal(t, "orgUnit", q.Get("groupBy"))
	for _, key := range []string{"orgUnitId", "includeDescendants"} {
		_, ok := q[key]
		assert.Falsef(t, ok, "%s must stay absent when only --group-by was set", key)
	}

	assert.Contains(t, out, "Engineering")
	assert.Contains(t, out, "Unassigned",
		"the server's Unassigned label is passed through verbatim — the CLI substitutes no local placeholder")
	assert.Contains(t, out, "204", "a real department id must render as digits")
	assert.Contains(t, out, "Total pull requests: 1713",
		"the server's own total must be printed, through the nullable-numeric helper")

	// The seven-digit id is the precision contract. num() with an explicit
	// %.0f verb renders it as digits; str() or a bare print would emit
	// scientific notation, because encoding/json decoded the int64 into a
	// float64 (T-30-03).
	assert.Contains(t, out, "1234567", "a seven-digit department id must render as digits")
	assert.NotContains(t, out, "1.234567e+06",
		"an int64 department id must never reach the operator as scientific notation")

	engineering := strings.Index(out, "Engineering")
	unassigned := strings.Index(out, "Unassigned")
	platform := strings.Index(out, "Platform")
	require.NotEqual(t, -1, engineering)
	require.NotEqual(t, -1, unassigned)
	require.NotEqual(t, -1, platform)

	// Within the shared 2026-03-15 date, ascending by org-unit name.
	assert.Less(t, engineering, unassigned,
		"rows sharing a date must be ordered ascending by org-unit name")
	// Across dates, the LATER date sorts last even though "Platform" sorts
	// before "Unassigned" by name — which is what proves date is the primary
	// key rather than the only key that happened to agree.
	assert.Less(t, unassigned, platform,
		"rows must be ordered by date first: the 2026-03-16 row follows every 2026-03-15 row")

	// Stability: the two Engineering rows compare equal on BOTH sort keys and
	// must keep the order the server sent, which is what sort.SliceStable buys
	// over sort.Slice.
	first := strings.Index(out, "777")
	second := strings.Index(out, "888")
	require.NotEqual(t, -1, first)
	require.NotEqual(t, -1, second)
	assert.Less(t, first, second,
		"rows equal on both date and org-unit name must keep the order the server sent")

	// The JSON arm. --json is the server's document in the server's ORDER, so
	// rows must still be shuffled here. A sort above the output-mode branch
	// would sort this array too and turn this assertion red — which is the
	// whole reason the sort lives below it.
	jsonSrv, jsonCaptured := byOrgUnitServer(t, byOrgUnitFixtureGrouped)
	jsonOut, jsonErr := runByOrgUnit(t, jsonSrv, true,
		"--from", "2026-03-15", "--to", "2026-04-14", "--group-by", "orgUnit")
	require.NoError(t, jsonErr)
	require.Equal(t, 1, jsonCaptured.calls)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(jsonOut), &parsed))
	assert.Equal(t, "orgUnit", parsed["groupBy"], "the server's groupBy echo must survive --json")
	assert.Contains(t, parsed, "totalPullRequests", "the server's total must survive --json")

	jsonRows, ok := parsed["rows"].([]interface{})
	require.True(t, ok)
	require.Len(t, jsonRows, 4)
	firstRow, ok := jsonRows[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2026-03-16", firstRow["date"],
		"--json must preserve the server's row order — the sort belongs below the output-mode branch")
	assert.Equal(t, "Platform", firstRow["orgUnitName"])
}

// TestByOrgUnitQueryIsEncoded is the T-30-01 assertion, and this command is the
// phase's sharpest instance of the threat: --group-by is a free-form operator
// string with no closed enum in front of it.
//
// The two halves are different claims. The first says the value round-trips
// intact; the second says it did not SPLIT. Together they are the difference
// between one percent-encoded value and an operator appending a second query
// parameter to an authenticated request that carries the platform x-api-key.
func TestByOrgUnitQueryIsEncoded(t *testing.T) {
	srv, captured := byOrgUnitServer(t, byOrgUnitFixtureGrouped)

	_, err := runByOrgUnit(t, srv, false,
		"--from", "2026-03-15", "--to", "2026-04-14", "--group-by", "a&b=c")
	require.NoError(t, err)
	require.Equal(t, 1, captured.calls)
	require.NotNil(t, captured.url)

	q := captured.url.Query()
	assert.Equal(t, "a&b=c", q.Get("groupBy"),
		"the operator's value must arrive whole, as a single percent-encoded value")
	_, injected := q["b"]
	assert.False(t, injected,
		"an & or = inside an operator-supplied value must never become a second query key")
}

// TestByOrgUnitEmptyRows covers a window in which the server reports no merged
// pull requests at all. It is a SUCCESS in both output modes — a sentence in
// table mode, the server's own document under --json.
//
// The server's total is still printed in table mode: an implementation that
// returned before printing it would suppress a figure the server did send.
//
// Deliberately NOT written as t.Run subtests, for the reason
// TestByOrgUnitGrouped records.
func TestByOrgUnitEmptyRows(t *testing.T) {
	// Local rather than a package-level const because it exists only to pin
	// this one state. The spec declares rows as a plain array with no minimum,
	// so an empty one is a legitimate success rather than an error.
	const emptyFixture = `{
		"groupBy": null,
		"rows": [],
		"totalPullRequests": 0
	}`

	srv, captured := byOrgUnitServer(t, emptyFixture)
	out, err := runByOrgUnit(t, srv, false, "--from", "2026-03-15", "--to", "2026-04-14")
	require.NoError(t, err, "an empty series is a success, not an error")
	require.Equal(t, 1, captured.calls)

	assert.Contains(t, out, "No merged pull requests are reported for this window.")
	assert.Contains(t, out, "Total pull requests: 0",
		"a measured zero total is a fact the server sent and must still render")

	jsonSrv, jsonCaptured := byOrgUnitServer(t, emptyFixture)
	jsonOut, jsonErr := runByOrgUnit(t, jsonSrv, true, "--from", "2026-03-15", "--to", "2026-04-14")
	require.NoError(t, jsonErr)
	require.Equal(t, 1, jsonCaptured.calls)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(jsonOut), &parsed),
		"--json must still emit the server's document for an empty series")
	assert.Contains(t, parsed, "rows",
		"the server's own typed empty array must survive --json")
}

// TestByOrgUnitDescendants pins the two non-string parameters — the only ones
// in this phase.
//
// The literal "100" is what proves the int64 was serialised as digits:
// encoding a JSON-decoded numeric through the unformatted print family emits
// scientific notation at or above 1e6, and a department id is squarely in that
// range. The flag is typed Int64Var, so cobra refuses a non-integer before the
// query is built at all.
func TestByOrgUnitDescendants(t *testing.T) {
	srv, captured := byOrgUnitServer(t, byOrgUnitFixtureGrouped)

	_, err := runByOrgUnit(t, srv, false,
		"--from", "2026-03-15", "--to", "2026-04-14",
		"--org-unit-id", "100", "--include-descendants")
	require.NoError(t, err)
	require.Equal(t, 1, captured.calls)
	require.NotNil(t, captured.url)

	q := captured.url.Query()
	assert.Equal(t, "100", q.Get("orgUnitId"), "the int64 id must serialise as digits")
	assert.Equal(t, "true", q.Get("includeDescendants"))
}

// TestByOrgUnitOmitsUnsetFlags is the negative half of the changed-bit gate,
// and simultaneously the assertion that the CLI adds NO client-side interlock
// between --include-descendants and --org-unit-id.
//
// The spec states the server ignores includeDescendants when orgUnitId is
// omitted. That rule is the server's, and the flag's own usage string records
// it; a local refusal would be a second copy of a server rule that can drift
// away from it. So the parameter goes on the wire, and the two flags that were
// never set stay off it.
func TestByOrgUnitOmitsUnsetFlags(t *testing.T) {
	srv, captured := byOrgUnitServer(t, byOrgUnitFixtureUngrouped)

	_, err := runByOrgUnit(t, srv, false,
		"--from", "2026-03-15", "--to", "2026-04-14", "--include-descendants")
	require.NoError(t, err, "--include-descendants without --org-unit-id must not be refused client-side")
	require.Equal(t, 1, captured.calls)
	require.NotNil(t, captured.url)

	q := captured.url.Query()
	assert.Equal(t, "true", q.Get("includeDescendants"))
	for _, key := range []string{"groupBy", "orgUnitId"} {
		_, ok := q[key]
		assert.Falsef(t, ok, "%s must be absent from the query entirely when its flag was never set", key)
	}
}
