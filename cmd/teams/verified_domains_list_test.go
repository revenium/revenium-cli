package teams

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// TEAM-12 — `revenium teams verified-domains list <team-id>`.
//
// Every test in this file carries the TestVerifiedDomainsList prefix
// deliberately: the plan's verification gate runs `-run TestVerifiedDomainsList`
// and counts PASS lines, and a selector that matches nothing exits 0 while
// proving nothing.
//
// Every case is written FLAT rather than as a t.Run subtest. `go test -v`
// prints an additional indented `--- PASS:` line per subtest, and the gate's
// grep matches that substring anywhere in the line — so a subtest wrapper makes
// the plan's own exact-count gate unsatisfiable. 31-01 and 31-02 both recorded
// this; do not "tidy" these cases back into subtests.
// ---------------------------------------------------------------------------

// verifiedDomainsListStub starts a hermetic httptest server that serves body
// for the verified-domains settings path and counts the requests it observed.
// The requestCounter type is declared once for package teams in
// pr_health_get_test.go; it is reused here, not redeclared.
func verifiedDomainsListStub(t *testing.T, body string) (*httptest.Server, *requestCounter) {
	t.Helper()
	counter := &requestCounter{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v2/api/teams/team-1/settings/verified-domains", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, counter
}

// wireVerifiedDomains points the package-level API client at srv and installs a
// formatter writing into buf, returning the formatter so a test can narrow its
// fields before running the command.
func wireVerifiedDomains(srv *httptest.Server, buf *bytes.Buffer, jsonMode bool) *output.Formatter {
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	f := output.NewWithWriter(buf, buf, jsonMode, false)
	cmd.Output = f
	return f
}

// wireVerifiedDomainsQuiet mirrors wireVerifiedDomains but constructs the
// formatter QUIET and non-JSON.
//
// It is a sibling rather than a third parameter on wireVerifiedDomains because
// five shipped tests across two files call that helper, and widening its
// signature would be churn this change does not need.
//
// Quiet has to be fixed at CONSTRUCTION: output.NewWithWriter substitutes
// io.Discard for the writer when quiet is true and jsonMode is false, so quiet
// is not reachable through a post-construction configure callback on an
// already-built formatter.
func wireVerifiedDomainsQuiet(srv *httptest.Server, buf *bytes.Buffer) *output.Formatter {
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	f := output.NewWithWriter(buf, buf, false, true)
	cmd.Output = f
	return f
}

// TestVerifiedDomainsListRendersRows asserts the command issues exactly one GET
// to the settings path and renders all THREE columns of each row — including
// the two properties the operator cannot set (D-31-13).
func TestVerifiedDomainsListRendersRows(t *testing.T) {
	srv, counter := verifiedDomainsListStub(t, `[
		{"domain":"acme.example","source":"ADMIN","joinPolicy":"REQUEST"},
		{"domain":"beta.example","source":"SYSTEM","joinPolicy":"AUTOMATIC"}
	]`)

	var buf bytes.Buffer
	wireVerifiedDomains(srv, &buf, false)

	c := newVerifiedDomainsListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "acme.example")
	assert.Contains(t, out, "ADMIN")
	assert.Contains(t, out, "REQUEST")
	assert.Contains(t, out, "beta.example")
	assert.Contains(t, out, "SYSTEM")
	assert.Contains(t, out, "AUTOMATIC")
	assert.Equal(t, 1, counter.count(), "exactly one request must reach the server")
}

// TestVerifiedDomainsListEmpty pins the empty state (D-31-19) for both bodies
// the server can legally return — `[]` and `null` — in both output modes.
//
// Four flat cases, no subtests (see the file header).
func TestVerifiedDomainsListEmpty(t *testing.T) {
	const sentence = "No verified domains found for this team."

	for _, body := range []string{`[]`, `null`} {
		// Table mode: the sentence, and no header row — an operator must not be
		// shown an empty table that looks like a rendering failure.
		srvTable, _ := verifiedDomainsListStub(t, body)
		var tableBuf bytes.Buffer
		wireVerifiedDomains(srvTable, &tableBuf, false)

		tableCmd := newVerifiedDomainsListCmd()
		tableCmd.SetOut(&tableBuf)
		tableCmd.SetArgs([]string{"team-1"})
		require.NoError(t, tableCmd.Execute(), "body %s in table mode", body)

		tableOut := tableBuf.String()
		assert.Contains(t, tableOut, sentence, "body %s in table mode", body)
		assert.NotContains(t, tableOut, "Join Policy", "body %s must render no header row", body)

		// JSON mode: a TYPED empty array, so a machine consumer's parse still
		// succeeds rather than meeting a prose sentence on stdout.
		srvJSON, _ := verifiedDomainsListStub(t, body)
		var jsonBuf bytes.Buffer
		wireVerifiedDomains(srvJSON, &jsonBuf, true)

		jsonCmd := newVerifiedDomainsListCmd()
		jsonCmd.SetOut(&jsonBuf)
		jsonCmd.SetArgs([]string{"team-1"})
		require.NoError(t, jsonCmd.Execute(), "body %s in JSON mode", body)

		jsonOut := strings.TrimSpace(jsonBuf.String())
		assert.Equal(t, "[]", jsonOut, "body %s in JSON mode must emit a typed empty array", body)
		assert.NotContains(t, jsonOut, sentence, "body %s must keep the sentence out of the document", body)
	}
}

// TestVerifiedDomainsListQuietSuppressesEmptyStateSentence is code review
// finding WR-03: --quiet is documented as suppressing non-error output, so a
// sentence that survives it inverts the flag's meaning for the script reading
// the stream.
//
// The request count is asserted FIRST, so the emptiness assertion cannot pass
// on a command that stopped issuing the GET altogether.
func TestVerifiedDomainsListQuietSuppressesEmptyStateSentence(t *testing.T) {
	srv, counter := verifiedDomainsListStub(t, `[]`)

	var buf bytes.Buffer
	wireVerifiedDomainsQuiet(srv, &buf)

	c := newVerifiedDomainsListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	assert.Equal(t, 1, counter.count(), "exactly one request must reach the server")
	assert.Empty(t, buf.String(), "--quiet must leave stdout empty (WR-03)")
}

// TestVerifiedDomainsListSortsByDomain asserts row order is a property of our
// output rather than an assumption about a contract the spec never states
// (D-27-08), and that the sort is STABLE.
//
// The stability case is the half a plain sort.Slice would not guarantee: two
// rows sharing a domain must keep the server's relative order between them.
func TestVerifiedDomainsListSortsByDomain(t *testing.T) {
	// Reverse order from the server; alphabetical order out.
	srv, _ := verifiedDomainsListStub(t, `[
		{"domain":"zulu.example","source":"ADMIN","joinPolicy":"REQUEST"},
		{"domain":"alpha.example","source":"ADMIN","joinPolicy":"REQUEST"}
	]`)

	var buf bytes.Buffer
	wireVerifiedDomains(srv, &buf, false)

	c := newVerifiedDomainsListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Less(t, strings.Index(out, "alpha.example"), strings.Index(out, "zulu.example"),
		"rows must render in domain order regardless of the order the server sent them")

	// Stability: identical sort keys, distinguishable by a second column. The
	// server sent SECOND before FIRST; SliceStable must preserve that.
	srvStable, _ := verifiedDomainsListStub(t, `[
		{"domain":"tie.example","source":"SECOND","joinPolicy":"REQUEST"},
		{"domain":"tie.example","source":"FIRST","joinPolicy":"REQUEST"}
	]`)

	var stableBuf bytes.Buffer
	wireVerifiedDomains(srvStable, &stableBuf, false)

	stableCmd := newVerifiedDomainsListCmd()
	stableCmd.SetOut(&stableBuf)
	stableCmd.SetArgs([]string{"team-1"})
	require.NoError(t, stableCmd.Execute())

	stableOut := stableBuf.String()
	assert.Less(t, strings.Index(stableOut, "SECOND"), strings.Index(stableOut, "FIRST"),
		"rows comparing equal on the sort key must keep the server's relative order")
}

// TestVerifiedDomainsListFieldsNarrowsColumns is D-31-33's requirement: assert
// that --fields actually NARROWS the rendered columns, not merely that the flag
// is accepted. Phase 30's mitigation gate verified flag existence and the
// T-30-11 defect shipped anyway, because the renderer bypassed the filtering
// entry point. Routing through cmd.Output.Render is what this test pins.
func TestVerifiedDomainsListFieldsNarrowsColumns(t *testing.T) {
	srv, _ := verifiedDomainsListStub(t, `[{"domain":"acme.example","source":"ADMIN","joinPolicy":"REQUEST"}]`)

	var buf bytes.Buffer
	f := wireVerifiedDomains(srv, &buf, false)
	f.SetFields([]string{"Domain"})

	c := newVerifiedDomainsListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "Domain", "the retained column header must still render")
	assert.NotContains(t, out, "Join Policy", "--fields Domain must drop the Join Policy column")
}

// TestVerifiedDomainsListRegisteredUnderTeams pins the command at its exact path
// in the package-level tree — the same object main.go registers.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a Name()-only
// assertion stays green on a command hung off the wrong parent.
func TestVerifiedDomainsListRegisteredUnderTeams(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"verified-domains", "list"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "teams verified-domains list", resolved.CommandPath())
}
