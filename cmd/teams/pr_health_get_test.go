package teams

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// TEAM-08 — `revenium teams pr-health get <team-id>`.
//
// Every test in this file carries the TestPrHealthGet prefix deliberately: the
// plan's verification gate runs `-run TestPrHealthGet` and counts PASS lines,
// and a selector that matches nothing exits 0 while proving nothing.
// ---------------------------------------------------------------------------

// requestCounter counts the requests a stub handler observed. Copied in form
// from cmd/jobs/stub_test.go — package teams had no such helper. Increments are
// mutex-guarded: httptest serves each request on its own goroutine, so an
// unguarded counter would trip the race detector and report a data race instead
// of the guard regression the test exists to diagnose.
//
// It lives in this file rather than pr_health_set_test.go because the get tests
// were written first and need it too; both files share the single declaration.
type requestCounter struct {
	mu sync.Mutex
	n  int
}

func (c *requestCounter) inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *requestCounter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// prHealthGetStub starts a hermetic httptest server that serves body for the
// pr-health settings path and counts the requests it observed.
func prHealthGetStub(t *testing.T, body string) (*httptest.Server, *requestCounter) {
	t.Helper()
	counter := &requestCounter{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v2/api/teams/team-1/settings/pr-health", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, counter
}

// wirePrHealthGet points the package-level API client at srv and installs a
// formatter writing into buf, returning the formatter so a test can narrow its
// fields before running the command.
func wirePrHealthGet(srv *httptest.Server, buf *bytes.Buffer, jsonMode bool) *output.Formatter {
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	f := output.NewWithWriter(buf, buf, jsonMode, false)
	cmd.Output = f
	return f
}

// wirePrHealthGetQuiet mirrors wirePrHealthGet but constructs the formatter
// QUIET and non-JSON.
//
// It is a sibling rather than a third parameter on wirePrHealthGet because the
// shipped tests all call that helper and widening its signature would be churn
// this change does not need.
//
// Quiet has to be fixed at CONSTRUCTION: output.NewWithWriter substitutes
// io.Discard for the writer when quiet is true and jsonMode is false, so quiet
// is not reachable by configuring an already-built formatter.
func wirePrHealthGetQuiet(srv *httptest.Server, buf *bytes.Buffer) *output.Formatter {
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	f := output.NewWithWriter(buf, buf, false, true)
	cmd.Output = f
	return f
}

// TestPrHealthGetRendersServerValues asserts the command issues exactly one
// GET to the settings path and renders the server's own threshold values —
// not a client-side table of the schema's example values (D-31-01, D-22-04).
func TestPrHealthGetRendersServerValues(t *testing.T) {
	srv, counter := prHealthGetStub(t, `{"agingDays":7,"rottingDays":9}`)

	var buf bytes.Buffer
	wirePrHealthGet(srv, &buf, false)

	c := newPrHealthGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "agingDays")
	assert.Contains(t, out, "7")
	assert.Contains(t, out, "rottingDays")
	assert.Contains(t, out, "9")
	assert.Equal(t, 1, counter.count(), "exactly one request must reach the server")
}

// TestPrHealthGetCaptionOnlyInTableMode pins T-31-02: the provenance caption is
// an extra stdout line, so it must never land inside a --json document where it
// would corrupt a machine consumer's parse.
//
// Deliberately written as two flat cases rather than two t.Run subtests: the
// plan's verification gate counts `--- PASS: TestPrHealthGet` lines and demands
// an exact number (five when this was written, six since 31-09 added the quiet
// case), and `go test -v` prints an additional indented PASS line per subtest —
// which the gate's substring grep would also count.
func TestPrHealthGetCaptionOnlyInTableMode(t *testing.T) {
	const captionFragment = "the API applies its own defaults"

	// Table mode: the caption is an operator-facing heading and must be present.
	srvTable, _ := prHealthGetStub(t, `{"agingDays":7,"rottingDays":9}`)
	var tableBuf bytes.Buffer
	wirePrHealthGet(srvTable, &tableBuf, false)

	tableCmd := newPrHealthGetCmd()
	tableCmd.SetOut(&tableBuf)
	tableCmd.SetArgs([]string{"team-1"})
	require.NoError(t, tableCmd.Execute())
	assert.Contains(t, tableBuf.String(), captionFragment)

	// JSON mode: no --json consumer may see it (T-31-02).
	srvJSON, _ := prHealthGetStub(t, `{"agingDays":7,"rottingDays":9}`)
	var jsonBuf bytes.Buffer
	wirePrHealthGet(srvJSON, &jsonBuf, true)

	jsonCmd := newPrHealthGetCmd()
	jsonCmd.SetOut(&jsonBuf)
	jsonCmd.SetArgs([]string{"team-1"})
	require.NoError(t, jsonCmd.Execute())
	assert.NotContains(t, jsonBuf.String(), captionFragment)
}

// TestPrHealthGetQuietSuppressesCaption is code review finding WR-03.
//
// The caption exists only to QUALIFY the table — it says the numbers below are
// effective values with the API's defaults already applied. Quiet mode
// suppresses the table (RenderTable returns early), so a caption that survives
// prints a qualifier for something that is not there, which is the flag's
// meaning inverted rather than merely noisy.
//
// The request count is asserted FIRST, so the emptiness assertion cannot pass
// on a command that stopped issuing the GET altogether.
func TestPrHealthGetQuietSuppressesCaption(t *testing.T) {
	srv, counter := prHealthGetStub(t, `{"agingDays":7,"rottingDays":9}`)

	var buf bytes.Buffer
	wirePrHealthGetQuiet(srv, &buf)

	c := newPrHealthGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	assert.Equal(t, 1, counter.count(), "exactly one request must reach the server")
	assert.Empty(t, buf.String(), "--quiet must leave stdout empty (WR-03)")
}

// TestPrHealthGetFieldsNarrowsColumns is D-31-33's requirement: assert that
// --fields actually narrows the rendered columns, not merely that the flag is
// accepted. Phase 30's mitigation gate verified flag existence and the T-30-11
// defect shipped anyway, because the renderer bypassed the filtering entry
// point. Routing through cmd.Output.Render is what this test pins.
func TestPrHealthGetFieldsNarrowsColumns(t *testing.T) {
	srv, _ := prHealthGetStub(t, `{"agingDays":7,"rottingDays":9}`)

	var buf bytes.Buffer
	f := wirePrHealthGet(srv, &buf, false)
	f.SetFields([]string{"Value"})

	c := newPrHealthGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "Value", "the retained column header must still render")
	assert.NotContains(t, out, "Setting", "--fields Value must drop the Setting column")
}

// TestPrHealthGetSkipsLinks asserts HAL navigation noise never becomes a
// settings row.
func TestPrHealthGetSkipsLinks(t *testing.T) {
	srv, _ := prHealthGetStub(t, `{"agingDays":7,"rottingDays":9,"_links":{"self":{"href":"/v2/api/teams/team-1/settings/pr-health"}}}`)

	var buf bytes.Buffer
	wirePrHealthGet(srv, &buf, false)

	c := newPrHealthGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	assert.NotContains(t, buf.String(), "_links")
}

// TestPrHealthGetRegisteredUnderTeams pins the command at its exact path in the
// package-level tree — the same object main.go registers.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a Name()-only
// assertion stays green on a command hung off the wrong parent.
func TestPrHealthGetRegisteredUnderTeams(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"pr-health", "get"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "teams pr-health get", resolved.CommandPath())
}
