package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// Every fixture in this file is written from the VERBATIM schema in
// 30-RESEARCH.md § The API Contract → 1, using the spec's own example values —
// NOT from the rendering code. Writing a fixture from the code is Pitfall 2:
// the table then renders whatever the code already reads, the test is green,
// and the field names may not appear in any schema at all. That is the defect
// vcs_prs.go:45-46 shipped, and response drift is invisible to every gate in
// this repo.

// seatsFixture is one fully-populated day at the spec's example values.
const seatsFixture = `{
	"days": [
		{
			"date": "2026-08-22",
			"seatsPaid": 1500,
			"seatsUsed": 900,
			"pendingInvites": 12,
			"dailyActive": 310,
			"weeklyActive": 640,
			"monthlyActive": 900
		}
	]
}`

// seatsFixtureTwoDays supplies the days OUT of date order deliberately. The
// spec states no ordering for days, so the two output modes must diverge here:
// the table sorts ascending, --json stays the server's document in the server's
// order.
const seatsFixtureTwoDays = `{
	"days": [
		{
			"date": "2026-08-22",
			"seatsPaid": 1500,
			"seatsUsed": 900,
			"pendingInvites": 12,
			"dailyActive": 310,
			"weeklyActive": 640,
			"monthlyActive": 900
		},
		{
			"date": "2026-08-21",
			"seatsPaid": 1400,
			"seatsUsed": 880,
			"pendingInvites": 11,
			"dailyActive": 305,
			"weeklyActive": 620,
			"monthlyActive": 880
		}
	]
}`

// seatsFixtureWithheld carries the three states that must stay distinguishable:
// seatsPaid explicitly JSON null, pendingInvites absent from the object
// entirely, and dailyActive a measured zero. The spec says withheld counts
// arrive absent rather than zero, so the first two are "the vendor did not tell
// us" and the third is "we counted nobody".
const seatsFixtureWithheld = `{
	"days": [
		{
			"date": "2026-08-22",
			"seatsPaid": null,
			"seatsUsed": 9,
			"dailyActive": 0,
			"weeklyActive": 7,
			"monthlyActive": 9
		}
	]
}`

// seatsFixtureEmpty is the response an organization with no Claude Enterprise
// credential gets. The spec states it is an empty list, not an error.
const seatsFixtureEmpty = `{"days": []}`

// seatsServer returns an httptest server serving body, plus a pointer to the
// count of requests it received. The counter is what proves the two refusal
// tests refuse BEFORE the request rather than after it.
func seatsServer(t *testing.T, body string) (*httptest.Server, *int) {
	t.Helper()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

// TestSeatsRequestPath pins the request half of the slice: the exact path, the
// endpoint's own fromDate/toDate wire keys, the absence of the startDate/endDate
// spelling every OTHER date-ranged endpoint in this repo uses, and that exactly
// one request leaves the process.
//
// The negative half (startDate/endDate empty) is the load-bearing one. A command
// that sent the wrong spelling would still reach a live server, still get a
// response shape this test never inspects, and fail only as a 400 that reads
// like a date-format problem (30-RESEARCH.md § The API Contract → 1).
func TestSeatsRequestPath(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "/v2/api/billing/seats", r.URL.Path)
		assert.Equal(t, "2026-08-01", r.URL.Query().Get("fromDate"))
		assert.Equal(t, "2026-08-22", r.URL.Query().Get("toDate"))
		assert.Empty(t, r.URL.Query().Get("startDate"),
			"this endpoint spells the range fromDate/toDate — startDate must never be sent")
		assert.Empty(t, r.URL.Query().Get("endDate"),
			"this endpoint spells the range fromDate/toDate — endDate must never be sent")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, seatsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	// The third positional is the team id. A non-empty value is required here:
	// requireTeam() refuses before any HTTP request when the client resolved no
	// team, so an empty one would make this test assert a refusal rather than a
	// request path.
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newSeatsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--from", "2026-08-01", "--to", "2026-08-22"})
	require.NoError(t, c.Execute())

	assert.Equal(t, 1, calls, "the command must issue exactly one GET")
}

// TestSeatsTableRendersCensus asserts BILL-04's own success criterion: one row
// per UTC day carrying seats assigned, daily/weekly/30-day active people, and
// unaccepted invitations.
func TestSeatsTableRendersCensus(t *testing.T) {
	srv, calls := seatsServer(t, seatsFixtureTwoDays)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newSeatsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--from", "2026-08-21", "--to", "2026-08-22"})
	require.NoError(t, c.Execute())
	require.Equal(t, 1, *calls)

	out := buf.String()
	assert.Contains(t, out, "Monthly Active", "the header row must name the 30-day figure")
	for _, want := range []string{
		"2026-08-21", "1400", "305", "620", "880", "11",
		"2026-08-22", "1500", "310", "640", "900", "12",
	} {
		assert.Contains(t, out, want, "the table must carry the server's value %q", want)
	}
}

// TestSeatsWithheldCountsRenderDash is the T-30-03 assertion: a figure the
// vendor withheld and a figure measured as zero must be DIFFERENT strings in
// the rendered table.
//
// Asserting only that a marker appears somewhere would stay green on a render
// that turned every count into a marker, so this asserts both halves: two
// markers for the two withheld cells, and a standalone zero for the measured
// one. The zero is matched on word boundaries so a digit inside the date
// (2026-08-22) cannot satisfy it.
func TestSeatsWithheldCountsRenderDash(t *testing.T) {
	srv, calls := seatsServer(t, seatsFixtureWithheld)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newSeatsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--from", "2026-08-22", "--to", "2026-08-22"})
	require.NoError(t, c.Execute())
	require.Equal(t, 1, *calls)

	out := buf.String()
	assert.GreaterOrEqual(t, strings.Count(out, undeclared), 2,
		"seatsPaid (null) and pendingInvites (absent) must both render as the undeclared marker, not as 0")
	assert.Regexp(t, regexp.MustCompile(`\b0\b`), out,
		"dailyActive was measured as zero and must still render as 0, not as the undeclared marker")
}

// TestSeatsEmptyDays covers the spec's own statement that an organization with
// no Claude Enterprise credential returns an empty list rather than an error —
// so this is a success path in BOTH output modes, not a failure and not an
// empty JSON document.
//
// Deliberately NOT written as two t.Run subtests: 30-01-PLAN.md's verify gate
// counts `--- PASS: TestSeats` lines and requires exactly seven, and go test
// emits an additional indented PASS line per subtest.
func TestSeatsEmptyDays(t *testing.T) {
	// Table mode: the sentence, not an empty table.
	tableSrv, tableCalls := seatsServer(t, seatsFixtureEmpty)

	var tableBuf bytes.Buffer
	cmd.APIClient = api.NewClient(tableSrv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&tableBuf, &tableBuf, false, false)

	tableCmd := newSeatsCmd()
	tableCmd.SetOut(&tableBuf)
	tableCmd.SetArgs([]string{"--from", "2026-08-01", "--to", "2026-08-22"})
	require.NoError(t, tableCmd.Execute(), "an empty census is a success, not an error")
	require.Equal(t, 1, *tableCalls)
	assert.Contains(t, tableBuf.String(), "No seat utilization data found")

	// JSON mode: still the server's own document, not a suppressed one.
	jsonSrv, jsonCalls := seatsServer(t, seatsFixtureEmpty)

	var jsonBuf bytes.Buffer
	cmd.APIClient = api.NewClient(jsonSrv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&jsonBuf, &jsonBuf, true, false)

	jsonCmd := newSeatsCmd()
	jsonCmd.SetOut(&jsonBuf)
	jsonCmd.SetArgs([]string{"--from", "2026-08-01", "--to", "2026-08-22"})
	require.NoError(t, jsonCmd.Execute())
	require.Equal(t, 1, *jsonCalls)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &parsed),
		"--json must still emit the server's object for an empty census")
	_, ok := parsed["days"]
	assert.True(t, ok, "the emitted document must still carry the days key")
}

// TestSeatsJSONIsVerbatim proves the D-30-03 column decision is a TABLE
// decision and not data loss: seatsUsed has no column, but it survives
// untouched under --json.
//
// It also pins the ordering contract from both sides. The fixture supplies the
// days out of date order; --json must emit them in the server's order, and the
// table must emit them ascending. A sort placed above the output-mode branch
// would pass the table half and silently break the JSON half.
//
// Deliberately NOT written as t.Run subtests, for the reason
// TestSeatsEmptyDays records.
func TestSeatsJSONIsVerbatim(t *testing.T) {
	// JSON mode: the server's field set AND the server's row order.
	jsonSrv, jsonCalls := seatsServer(t, seatsFixtureTwoDays)

	var jsonBuf bytes.Buffer
	cmd.APIClient = api.NewClient(jsonSrv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&jsonBuf, &jsonBuf, true, false)

	jsonCmd := newSeatsCmd()
	jsonCmd.SetOut(&jsonBuf)
	jsonCmd.SetArgs([]string{"--from", "2026-08-21", "--to", "2026-08-22"})
	require.NoError(t, jsonCmd.Execute())
	require.Equal(t, 1, *jsonCalls)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &parsed))
	days, ok := parsed["days"].([]interface{})
	require.True(t, ok)
	require.Len(t, days, 2)

	first, ok := days[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(900), first["seatsUsed"],
		"seatsUsed has no table column (D-30-03) but must survive verbatim under --json")
	assert.Equal(t, "2026-08-22", first["date"],
		"--json must emit the days in the SERVER's order — the fixture supplies 08-22 first")

	// Table mode, same fixture: sorted ascending. Running both halves in one
	// test is what proves the sort lives BELOW the output-mode branch — a sort
	// placed above it would pass this half and silently break the one above.
	tableSrv, tableCalls := seatsServer(t, seatsFixtureTwoDays)

	var tableBuf bytes.Buffer
	cmd.APIClient = api.NewClient(tableSrv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&tableBuf, &tableBuf, false, false)

	tableCmd := newSeatsCmd()
	tableCmd.SetOut(&tableBuf)
	tableCmd.SetArgs([]string{"--from", "2026-08-21", "--to", "2026-08-22"})
	require.NoError(t, tableCmd.Execute())
	require.Equal(t, 1, *tableCalls)

	out := tableBuf.String()
	earlier := strings.Index(out, "2026-08-21")
	later := strings.Index(out, "2026-08-22")
	require.NotEqual(t, -1, earlier)
	require.NotEqual(t, -1, later)
	assert.Less(t, earlier, later,
		"table rows must be ordered by date ascending regardless of the order the server sent")
}

// TestSeatsRequiresDateRange asserts the refusal costs zero HTTP calls. Both
// range parameters are required: true in the spec, so a request missing one
// would be rejected by the server anyway — but sending it spends an
// authenticated round trip to learn what cobra already knew.
func TestSeatsRequiresDateRange(t *testing.T) {
	srv, calls := seatsServer(t, seatsFixture)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newSeatsCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--from", "2026-08-01"})
	err := c.Execute()

	require.Error(t, err, "a missing required date must be refused")
	assert.Contains(t, err.Error(), "to", "the error must name the missing flag")
	assert.Equal(t, 0, *calls, "the refusal must precede the request")
}

// TestSeatsRequiresTeam is the D-30-05 pin. The seats operation declares teamId
// required, but the shared client appends it only when one has been resolved —
// so without this guard an unresolved team produces an authenticated request
// that comes back as a server-side error the operator has to decode.
func TestSeatsRequiresTeam(t *testing.T) {
	srv, calls := seatsServer(t, seatsFixture)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newSeatsCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--from", "2026-08-01", "--to", "2026-08-22"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "REVENIUM_TEAM_ID",
		"the message must stand alone and name all three resolution sources — cmd/root.go silences usage and errors")
	assert.Equal(t, 0, *calls, "the refusal must precede the request")
}
