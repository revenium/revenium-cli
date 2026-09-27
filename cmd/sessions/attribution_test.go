package sessions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// SESS-01 — `revenium sessions attribution <session-id>`.
//
// Every test here carries the TestSessionsAttribution prefix deliberately: the
// plan's verification gate runs `-run TestSessionsAttribution` and counts PASS
// lines, and a selector that matches nothing exits 0 while proving nothing.
//
// Every test here is also FLAT — no t.Run subtests anywhere, including where
// the plan's prose says "subtests". `go test -v` prints an additional indented
// `--- PASS: Parent/Sub` line per subtest, and the gate's grep matches its
// pattern as a substring anywhere in the line, so each subtest would inflate a
// count the gate pins exactly (eight when this was written, ten since 31-09
// added the two quiet cases). This is the same gate-shape constraint
// 31-01 and 31-02 recorded; do not "tidy" these cases back into subtests.
// ---------------------------------------------------------------------------

// emDash is what an interval with no split renders in the Splits column. Named
// so an assertion reads as the contract rather than as a mystery rune.
const emDash = "—"

// noSplitsSentinel is the exact wording the empty-state branch prints. A
// session with no recorded attribution is ordinary, not an error (D-31-19).
const noSplitsSentinel = "No attribution intervals recorded for this session."

// requestCounter counts the requests a stub handler observed and records their
// methods and paths. Copied in form from cmd/jobs/stub_test.go — package
// sessions had no such helper. Increments are mutex-guarded: httptest serves
// each request on its own goroutine, so an unguarded counter would trip the
// race detector and report a data race instead of the regression the test
// exists to diagnose.
//
// The method/path slices are the local extension: SESS-01's read-only property
// is "exactly one GET and no request of any other method", which a bare count
// cannot express.
type requestCounter struct {
	mu      sync.Mutex
	n       int
	methods []string
	paths   []string
}

func (c *requestCounter) record(method, path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	c.methods = append(c.methods, method)
	c.paths = append(c.paths, path)
}

func (c *requestCounter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func (c *requestCounter) methodsSeen() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.methods...)
}

func (c *requestCounter) pathsSeen() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.paths...)
}

// attributionStub starts a hermetic httptest server that serves body for every
// request and records what it observed.
//
// Response bodies handed to this stub are hand-built to match the dev spec's
// CollectionModel/SessionAttributionResource shapes. They are never a decode of
// the cached OpenAPI document and never a frozen testdata fixture — D-27-12's
// posture, because a fixture that drifts silently proves only that the CLI
// still agrees with a snapshot.
func attributionStub(t *testing.T, body string) (*httptest.Server, *requestCounter) {
	t.Helper()
	rec := &requestCounter{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r.Method, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method, "sessions attribution is read-only")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// withRealCmd prepares the package-level Cmd singleton for direct execution
// inside a test and registers the cleanup that undoes it.
//
// Copied from cmd/skills/cmd_test.go, and it exists for the reason recorded
// there: cobra's ExecuteC redirects to Root() whenever the receiver has a
// parent, so args have to be set on the package-level Cmd. Cmd is shared state
// across every test in this package, so SilenceErrors, SilenceUsage and the
// parsed args are all restored on cleanup.
func withRealCmd(t *testing.T) {
	t.Helper()
	prevSilenceErrors := Cmd.SilenceErrors
	prevSilenceUsage := Cmd.SilenceUsage
	Cmd.SilenceErrors = true
	Cmd.SilenceUsage = true
	t.Cleanup(func() {
		Cmd.SilenceErrors = prevSilenceErrors
		Cmd.SilenceUsage = prevSilenceUsage
		Cmd.SetArgs(nil)
	})
}

// runAttribution drives the package-level Cmd — the same object main.go
// registers — rather than a bare newAttributionCmd(), so every test exercises
// the registration too.
//
// configure runs against the formatter before the command does, which is how a
// test narrows --fields without needing a real flag parse.
func runAttribution(t *testing.T, srv *httptest.Server, buf *bytes.Buffer, jsonMode bool, configure func(*output.Formatter), args ...string) error {
	t.Helper()
	withRealCmd(t)

	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	f := output.NewWithWriter(buf, buf, jsonMode, false)
	if configure != nil {
		configure(f)
	}
	cmd.Output = f

	Cmd.SetOut(buf)
	Cmd.SetErr(buf)
	Cmd.SetArgs(append([]string{"attribution"}, args...))
	return Cmd.Execute()
}

// runAttributionQuiet mirrors runAttribution but constructs the formatter QUIET
// and non-JSON.
//
// It is a sibling rather than a widened runAttribution: eight shipped tests
// call that helper, and its `configure` callback cannot set quiet anyway —
// output.NewWithWriter substitutes io.Discard for the writer at CONSTRUCTION
// when quiet is true and jsonMode is false, so quiet is unreachable on an
// already-built formatter.
func runAttributionQuiet(t *testing.T, srv *httptest.Server, buf *bytes.Buffer, args ...string) error {
	t.Helper()
	withRealCmd(t)

	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(buf, buf, false, true)

	Cmd.SetOut(buf)
	Cmd.SetErr(buf)
	Cmd.SetArgs(append([]string{"attribution"}, args...))
	return Cmd.Execute()
}

// ansiRE matches the SGR escapes lipgloss emits around table borders and
// headers.
var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// tableCells returns the rendered table as trimmed cells — index 0 is the
// header row, index 1 the first DATA row.
//
// The ANSI strip is load-bearing, not cosmetic: output.NewWithWriter
// deliberately skips the colorprofile wrapper that strips escapes in
// production, so a test that wants a CELL rather than a substring has to strip
// them itself. Cell-level assertions are what let "the first data row IS
// element 0" and "Splits reads exactly 3" be identities rather than
// substring coincidences.
func tableCells(out string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(ansiRE.ReplaceAllString(out, ""), "\n") {
		if !strings.Contains(line, "│") {
			continue
		}
		parts := strings.Split(line, "│")
		if len(parts) < 3 {
			continue
		}
		parts = parts[1 : len(parts)-1] // drop the empty fields outside the borders
		cells := make([]string, len(parts))
		for i, p := range parts {
			cells[i] = strings.TrimSpace(p)
		}
		rows = append(rows, cells)
	}
	return rows
}

// TestSessionsAttributionRendersIntervals is the wire assertion: one GET to the
// session's attribution path, no request of any other method, and the server's
// ticket ids in the output.
func TestSessionsAttributionRendersIntervals(t *testing.T) {
	srv, rec := attributionStub(t, `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha"},
		{"effectiveFrom":"2026-01-01T00:00:00Z","ticketId":"TICKET-B","ticketTitle":"Beta"}
	]},"_links":{"self":{"href":"/v2/api/sessions/sess-1/attribution"}}}`)

	var buf bytes.Buffer
	require.NoError(t, runAttribution(t, srv, &buf, false, nil, "sess-1"))

	out := buf.String()
	assert.Contains(t, out, "TICKET-A")
	assert.Contains(t, out, "TICKET-B")
	assert.Contains(t, out, "Alpha")
	assert.Equal(t, 1, rec.count(), "exactly one request must reach the server")
	assert.Equal(t, []string{http.MethodGet}, rec.methodsSeen(), "a read issues no request of any other method")
	assert.Equal(t, []string{"/v2/api/sessions/sess-1/attribution"}, rec.pathsSeen())
	assert.NotContains(t, out, "_links", "HAL navigation noise must not become a row")
}

// TestSessionsAttributionPreservesServerOrder is the SC4 assertion, and it is
// deliberately an IDENTITY against the fixture rather than a sort result
// (D-31-16).
//
// The three-interval fixture is in the order the server contract states
// (current interval first, effectiveFrom descending) — which is exactly the
// order a lexicographic ASCENDING sort would reverse. So if anyone ever adds a
// client-side sort, this test turns red rather than silently agreeing.
//
// The second case is the adjacency edge: two intervals sharing an identical
// effectiveFrom keep the server's relative order, because nothing reorders
// them.
func TestSessionsAttributionPreservesServerOrder(t *testing.T) {
	srv, _ := attributionStub(t, `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-03-01T00:00:00Z","ticketId":"TICKET-CURRENT","ticketTitle":"Current"},
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-MIDDLE","ticketTitle":"Middle"},
		{"effectiveFrom":"2026-01-01T00:00:00Z","ticketId":"TICKET-OLDEST","ticketTitle":"Oldest"}
	]}}`)

	var buf bytes.Buffer
	require.NoError(t, runAttribution(t, srv, &buf, false, nil, "sess-1"))

	out := buf.String()
	rows := tableCells(out)
	require.GreaterOrEqual(t, len(rows), 4, "header plus three data rows")
	assert.Equal(t, "TICKET-CURRENT", rows[1][1], "the rendered first data row IS _embedded.objectList[0]")
	assert.Equal(t, "TICKET-MIDDLE", rows[2][1])
	assert.Equal(t, "TICKET-OLDEST", rows[3][1])
	assert.Less(t, strings.Index(out, "TICKET-CURRENT"), strings.Index(out, "TICKET-MIDDLE"),
		"element 0 must appear before element 1 in the rendered output")

	// Adjacency edge, flat second case: identical effectiveFrom, fixture order.
	srvTie, _ := attributionStub(t, `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-FIRST","ticketTitle":"First"},
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-SECOND","ticketTitle":"Second"}
	]}}`)

	var tieBuf bytes.Buffer
	require.NoError(t, runAttribution(t, srvTie, &tieBuf, false, nil, "sess-1"))

	tieRows := tableCells(tieBuf.String())
	require.GreaterOrEqual(t, len(tieRows), 3)
	assert.Equal(t, "TICKET-FIRST", tieRows[1][1], "a tie keeps the server's relative order")
	assert.Equal(t, "TICKET-SECOND", tieRows[2][1])
}

// TestSessionsAttributionSplitsRenderAsCount pins D-31-18: the nested array is
// never flattened into a cell. The count tells an operator THAT an interval is
// split so they know to reach for --json; absent, null and empty all read the
// same because they mean the same thing.
func TestSessionsAttributionSplitsRenderAsCount(t *testing.T) {
	// Present: a three-element splits array renders exactly "3", and no split
	// member id leaks into the table.
	srv, _ := attributionStub(t, `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha","splits":[
			{"ticketId":"SPLIT-ONE","weight":0.5,"ticketTitle":"One"},
			{"ticketId":"SPLIT-TWO","weight":0.3,"ticketTitle":"Two"},
			{"ticketId":"SPLIT-THREE","weight":0.2,"ticketTitle":"Three"}
		]}
	]}}`)

	var buf bytes.Buffer
	require.NoError(t, runAttribution(t, srv, &buf, false, nil, "sess-1"))

	out := buf.String()
	rows := tableCells(out)
	require.GreaterOrEqual(t, len(rows), 2)
	assert.Equal(t, "3", rows[1][3], "Splits renders the element count")
	assert.NotContains(t, out, "SPLIT-ONE", "a nested array is never flattened into a table cell")
	assert.NotContains(t, out, "SPLIT-TWO")
	assert.NotContains(t, out, "SPLIT-THREE")
	assert.NotContains(t, out, "0.5", "the CLI performs no arithmetic on the weights")

	// Absent key, explicit null, and empty array are three flat cases with one
	// shared expectation: the em dash.
	for _, body := range []string{
		`{"_embedded":{"objectList":[{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha"}]}}`,
		`{"_embedded":{"objectList":[{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha","splits":null}]}}`,
		`{"_embedded":{"objectList":[{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha","splits":[]}]}}`,
	} {
		caseSrv, _ := attributionStub(t, body)
		var caseBuf bytes.Buffer
		require.NoError(t, runAttribution(t, caseSrv, &caseBuf, false, nil, "sess-1"))

		caseRows := tableCells(caseBuf.String())
		require.GreaterOrEqual(t, len(caseRows), 2)
		assert.Equal(t, emDash, caseRows[1][3], "no split renders an em dash: %s", body)
	}
}

// TestSessionsAttributionEmptyList pins D-31-19: an absent or empty
// _embedded.objectList is a real case, not an error. Under --json it is a
// TYPED empty array so a machine consumer's parse still succeeds; otherwise a
// sentence, and no header row at all.
func TestSessionsAttributionEmptyList(t *testing.T) {
	const absentEmbedded = `{"_links":{"self":{"href":"/v2/api/sessions/sess-1/attribution"}}}`
	const emptyList = `{"_embedded":{"objectList":[]},"_links":{}}`

	// Table mode: the sentence, and no table.
	for _, body := range []string{absentEmbedded, emptyList} {
		srv, rec := attributionStub(t, body)
		var buf bytes.Buffer
		require.NoError(t, runAttribution(t, srv, &buf, false, nil, "sess-1"))

		out := buf.String()
		assert.Contains(t, out, noSplitsSentinel, "body: %s", body)
		assert.NotContains(t, out, "Effective From", "an empty list renders no header row: %s", body)
		assert.Equal(t, 1, rec.count(), "the empty case is still one ordinary GET")
	}

	// JSON mode: a typed empty array, not null and not a bare object.
	for _, body := range []string{absentEmbedded, emptyList} {
		srv, _ := attributionStub(t, body)
		var buf bytes.Buffer
		require.NoError(t, runAttribution(t, srv, &buf, true, nil, "sess-1"))

		assert.Equal(t, "[]", strings.TrimSpace(buf.String()), "body: %s", body)
	}
}

// TestSessionsAttributionNarrowColumns pins D-31-17 and T-31-12: the rendered
// column set is the INTERSECTION every credential plane receives, and it is not
// a function of which keys the response happened to carry.
//
// A management-plane caller (WRITE-scoped key or JWT) receives apiKeyId,
// createdBy, modifiedBy and subscriberEmail; a METERING-scoped key receives
// only ticketId, ticketTitle and effectiveFrom, with the rest omitted from the
// response entirely rather than nulled. Both must render the same four columns
// — otherwise a narrower caller sees four permanently blank columns that read
// as data loss, or two operators comparing terminals see different columns with
// no way to know why.
func TestSessionsAttributionNarrowColumns(t *testing.T) {
	expectedHeaders := []string{"Effective From", "Ticket", "Title", "Splits"}

	const managementPlane = `{"_embedded":{"objectList":[{
		"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha",
		"apiKeyId":"key-9","createdBy":"creator-1","modifiedBy":"modifier-2",
		"subscriberEmail":"ops@example.com"}]}}`
	const meteringPlane = `{"_embedded":{"objectList":[{
		"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha"}]}}`

	srvMgmt, _ := attributionStub(t, managementPlane)
	var mgmtBuf bytes.Buffer
	require.NoError(t, runAttribution(t, srvMgmt, &mgmtBuf, false, nil, "sess-1"))

	mgmtOut := mgmtBuf.String()
	mgmtRows := tableCells(mgmtOut)
	require.GreaterOrEqual(t, len(mgmtRows), 2)
	assert.Equal(t, expectedHeaders, mgmtRows[0], "the management plane sees the intersection columns")
	assert.NotContains(t, mgmtOut, "ops@example.com", "provenance is --json's job, not the table's")
	assert.NotContains(t, mgmtOut, "key-9")
	assert.NotContains(t, mgmtOut, "creator-1")
	assert.NotContains(t, mgmtOut, "modifier-2")
	assert.Contains(t, mgmtOut, "(use --json for full detail)")

	srvMeter, _ := attributionStub(t, meteringPlane)
	var meterBuf bytes.Buffer
	require.NoError(t, runAttribution(t, srvMeter, &meterBuf, false, nil, "sess-1"))

	meterRows := tableCells(meterBuf.String())
	require.GreaterOrEqual(t, len(meterRows), 2)
	assert.Equal(t, expectedHeaders, meterRows[0], "the metering plane sees the SAME columns, not four blank ones")

	// --json is lossless: the provenance the table omits is still there.
	srvJSON, _ := attributionStub(t, managementPlane)
	var jsonBuf bytes.Buffer
	require.NoError(t, runAttribution(t, srvJSON, &jsonBuf, true, nil, "sess-1"))

	jsonOut := jsonBuf.String()
	assert.Contains(t, jsonOut, "subscriberEmail")
	assert.Contains(t, jsonOut, "ops@example.com")
	assert.Contains(t, jsonOut, "apiKeyId")
	assert.NotContains(t, jsonOut, "(use --json for full detail)", "the hint must never land in a JSON document")
}

// TestSessionsAttributionQuietSuppressesHint is code review finding WR-03.
//
// The hint points at a flag that reveals MORE of a table, and quiet mode has
// already suppressed the table (RenderTable returns early). A hint that
// survives is therefore a pointer to nothing, on a stream a script is entitled
// to read as empty.
//
// The request count is asserted FIRST, so the emptiness assertion cannot pass
// on a command that stopped issuing the GET altogether.
func TestSessionsAttributionQuietSuppressesHint(t *testing.T) {
	srv, rec := attributionStub(t, `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha"},
		{"effectiveFrom":"2026-01-01T00:00:00Z","ticketId":"TICKET-B","ticketTitle":"Beta"}
	]}}`)

	var buf bytes.Buffer
	require.NoError(t, runAttributionQuiet(t, srv, &buf, "sess-1"))

	assert.Equal(t, 1, rec.count(), "exactly one request must reach the server")
	assert.Empty(t, buf.String(), "--quiet must leave stdout empty (WR-03)")
}

// TestSessionsAttributionQuietSuppressesEmptyStateSentence is the empty-response
// half of the same property: --quiet suppresses the empty-state sentence too,
// so both branches of the command leave stdout empty rather than only the one
// that renders a table.
func TestSessionsAttributionQuietSuppressesEmptyStateSentence(t *testing.T) {
	srv, rec := attributionStub(t, `{"_embedded":{"objectList":[]},"_links":{}}`)

	var buf bytes.Buffer
	require.NoError(t, runAttributionQuiet(t, srv, &buf, "sess-1"))

	assert.Equal(t, 1, rec.count(), "exactly one request must reach the server")
	assert.Empty(t, buf.String(), "--quiet must leave stdout empty (WR-03)")
}

// TestSessionsAttributionFieldsNarrowsColumns is D-31-33's requirement, and it
// asserts --fields actually NARROWS the columns rather than merely being
// accepted. Phase 30's mitigation gate verified flag existence and the T-30-11
// defect shipped anyway, because the renderer bypassed the filtering entry
// point. Routing through cmd.Output.Render is what this test pins.
func TestSessionsAttributionFieldsNarrowsColumns(t *testing.T) {
	srv, _ := attributionStub(t, `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha"}
	]}}`)

	var buf bytes.Buffer
	require.NoError(t, runAttribution(t, srv, &buf, false, func(f *output.Formatter) {
		f.SetFields([]string{"Ticket"})
	}, "sess-1"))

	out := buf.String()
	rows := tableCells(out)
	require.GreaterOrEqual(t, len(rows), 2)
	assert.Equal(t, []string{"Ticket"}, rows[0], "--fields Ticket must leave exactly one column")
	assert.NotContains(t, out, "Effective From", "--fields Ticket must drop the Effective From column")
}

// TestSessionsAttributionNoTeamFlagAndNoGuard pins D-31-20. The GET declares
// teamId as required:false and Client.Do already injects it from the resolved
// Client.TeamID for every non-bearer platform request, so a per-command flag
// would duplicate the root's persistent --team-id and a client-side guard would
// refuse a command the server accepts.
func TestSessionsAttributionNoTeamFlagAndNoGuard(t *testing.T) {
	assert.Nil(t, newAttributionCmd().Flags().Lookup("team-id"),
		"no per-command --team-id: the root's persistent flag already carries it")

	srv, rec := attributionStub(t, `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha"}
	]}}`)

	// runAttribution wires api.NewClient with an EMPTY TeamID — the exact shape
	// a requireTeam()-style guard would refuse.
	var buf bytes.Buffer
	require.NoError(t, runAttribution(t, srv, &buf, false, nil, "sess-1"),
		"an empty TeamID must not be refused client-side")
	assert.Equal(t, 1, rec.count(), "the request must actually be issued, not guarded away")
	assert.Contains(t, buf.String(), "TICKET-A")
}

// TestSessionsAttributionRegisteredOnCmd pins the command at its exact path in
// the package-level tree — the same object main.go registers — and pins the
// package's scope: one read verb, no write verb.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a Name()-only
// assertion stays green on a command hung off the wrong parent.
func TestSessionsAttributionRegisteredOnCmd(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"attribution"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "sessions attribution", resolved.CommandPath())

	// The scope assertion (T-31-14): no write verb may appear alongside the
	// read, or the recorded deliberate skip on POST
	// /v2/api/sessions/{sessionId}/attribution would be retired by accident.
	//
	// cobra's own `help` and `completion` are filtered out rather than counted:
	// ExecuteC injects both into whichever command it treats as root, so any
	// earlier test in this file that ran Cmd.Execute() has already added them.
	// A raw len(Cmd.Commands()) == 1 would therefore pass or fail on test
	// ORDER, which is not the property being asserted.
	var declared []string
	for _, sub := range Cmd.Commands() {
		if sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		declared = append(declared, sub.Name())
	}
	assert.Equal(t, []string{"attribution"}, declared, "cmd/sessions must gain no verb alongside the read")
}

// TestSessionsAttributionJSONShapeIsStableAcrossStates is the SESS-01
// CROSS-STATE SHAPE CONTRACT: `--json` emits a JSON ARRAY of interval objects
// in every response state, so one consumer script parses both.
//
// It exists because 31-VERIFICATION.md's truth 8b and 31-REVIEW.md's
// second-pass CR-01 (Critical) both record the two `--json` exits returning
// DIFFERENT top-level types: the empty exit rendered a bare `[]` while the
// populated exit rendered the untouched CollectionModel wrapper OBJECT. A
// script could not parse both — `jq '.[]'` iterated `_embedded`/`_links` on the
// populated case and the intervals on the empty one, and
// `jq '._embedded.objectList | length'` failed on the empty case with
// `Cannot index array with string "_embedded"`.
//
// The whole point of this test is the SINGLE DECLARED GO TYPE used for both
// states: every fixture below unmarshals into `[]map[string]interface{}`, and
// none of them may unmarshal into `map[string]interface{}`. Neither shipped
// test makes that assertion — TestSessionsAttributionEmptyList pins the empty
// exit's literal `[]`, and TestSessionsAttributionNarrowColumns pins SUBSTRINGS
// of the populated exit, which are true of either shape. If a future edit
// widens the declared type per case, or branches on the response, the contract
// has been deleted rather than tested.
//
// Flat by construction — no t.Run — per this file's header rule.
func TestSessionsAttributionJSONShapeIsStableAcrossStates(t *testing.T) {
	// Hand-built to the dev spec's CollectionModel / SessionAttributionResource
	// shapes (D-27-12's posture): never a decode of the cached OpenAPI document
	// and never a frozen testdata file.
	const emptyBody = `{"_embedded":{"objectList":[]},"_links":{}}`
	const singleBody = `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha"}
	]},"_links":{"self":{"href":"/v2/api/sessions/sess-1/attribution"}}}`
	// Ordered descending, as the server's contract states — which is the order a
	// lexicographic ASCENDING pass would reverse.
	const managementPlaneBody = `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-A","ticketTitle":"Alpha",
		 "apiKeyId":"key-9","createdBy":"creator-1","modifiedBy":"modifier-2",
		 "subscriberEmail":"ops@example.com"},
		{"effectiveFrom":"2026-01-01T00:00:00Z","ticketId":"TICKET-B","ticketTitle":"Beta"}
	]},"_links":{"self":{"href":"/v2/api/sessions/sess-1/attribution"}}}`
	// The adjacency edge: two intervals whose effectiveFrom values are the
	// identical string.
	const tieBody = `{"_embedded":{"objectList":[
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-EQ1","ticketTitle":"First"},
		{"effectiveFrom":"2026-02-01T00:00:00Z","ticketId":"TICKET-EQ2","ticketTitle":"Second"}
	]},"_links":{}}`

	for _, tc := range []struct {
		name string
		body string
		want int
	}{
		{"empty", emptyBody, 0},
		{"single interval", singleBody, 1},
		{"management plane, two intervals", managementPlaneBody, 2},
		{"equal effectiveFrom", tieBody, 2},
	} {
		srv, _ := attributionStub(t, tc.body)
		var buf bytes.Buffer
		require.NoError(t, runAttribution(t, srv, &buf, true, nil, "sess-1"), "case %s", tc.name)

		// THE assertion: one declared type, every state.
		var intervals []map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &intervals),
			"case %s: --json must parse into the same declared Go type in every response state", tc.name)
		assert.Len(t, intervals, tc.want, "case %s: one element per server interval", tc.name)

		// And the positive of the same property: no exit is a top-level object.
		var asObject map[string]interface{}
		assert.Error(t, json.Unmarshal(buf.Bytes(), &asObject),
			"case %s: no --json exit of this command may be a top-level JSON object", tc.name)
	}

	// The per-case value assertions the element count alone cannot carry. Each
	// re-runs its fixture through the same helpers.

	// Single: element 0 is the server's interval, not a synthesized one.
	srvSingle, _ := attributionStub(t, singleBody)
	var singleBuf bytes.Buffer
	require.NoError(t, runAttribution(t, srvSingle, &singleBuf, true, nil, "sess-1"))
	var single []map[string]interface{}
	require.NoError(t, json.Unmarshal(singleBuf.Bytes(), &single))
	require.Len(t, single, 1)
	assert.Equal(t, "TICKET-A", single[0]["ticketId"])

	// Management plane: server order survives into JSON, and so does every
	// per-item provenance field. This is the measured proof that dropping the
	// HAL wrapper drops nothing D-31-17's losslessness argument depends on.
	srvMgmt, _ := attributionStub(t, managementPlaneBody)
	var mgmtBuf bytes.Buffer
	require.NoError(t, runAttribution(t, srvMgmt, &mgmtBuf, true, nil, "sess-1"))
	var mgmt []map[string]interface{}
	require.NoError(t, json.Unmarshal(mgmtBuf.Bytes(), &mgmt))
	require.Len(t, mgmt, 2)
	assert.Equal(t, "TICKET-A", mgmt[0]["ticketId"], "element 0 of the array IS element 0 of _embedded.objectList")
	assert.Equal(t, "key-9", mgmt[0]["apiKeyId"], "unwrapping the envelope drops no per-item provenance")
	assert.Equal(t, "creator-1", mgmt[0]["createdBy"])
	assert.Equal(t, "modifier-2", mgmt[0]["modifiedBy"])
	assert.Equal(t, "ops@example.com", mgmt[0]["subscriberEmail"])

	// The envelope's removal is deliberate rather than incidental.
	mgmtOut := mgmtBuf.String()
	assert.NotContains(t, mgmtOut, "_embedded", "the HAL collection envelope is unwrapped, not republished")
	assert.NotContains(t, mgmtOut, "_links", "the collection-level HAL links are unwrapped, not republished")

	// Adjacency: byte-identical effectiveFrom values stay two separate elements
	// in the server's relative order — not merged, not deduplicated.
	srvTie, _ := attributionStub(t, tieBody)
	var tieBuf bytes.Buffer
	require.NoError(t, runAttribution(t, srvTie, &tieBuf, true, nil, "sess-1"))
	var tie []map[string]interface{}
	require.NoError(t, json.Unmarshal(tieBuf.Bytes(), &tie))
	require.Len(t, tie, 2)
	assert.Equal(t, "TICKET-EQ1", tie[0]["ticketId"], "a tie keeps the server's relative order")
	assert.Equal(t, "TICKET-EQ2", tie[1]["ticketId"])
}
