package teams

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// TEAM-10 — `revenium teams attribution-identity-policy get <team-id>`.
//
// Every test in this file carries the TestAttributionIdentityPolicyGet prefix
// deliberately: the plan's verification gate runs
// `-run TestAttributionIdentityPolicyGet` and counts PASS lines, and a selector
// that matches nothing exits 0 while proving nothing.
//
// All cases are written FLAT rather than as t.Run subtests. `go test -v` prints
// an additional indented `--- PASS:` line per subtest, and the gate's grep
// counts substring matches anywhere in the line — so a subtest wrapper would
// make the gate's own count unsatisfiable (31-01 deviation 1). Do not "tidy"
// these back into subtests.
//
// The mutex-guarded requestCounter these tests use is declared once for package
// teams in pr_health_get_test.go (31-01). It is deliberately NOT redeclared
// here — Go forbids a second declaration in the same package.
// ---------------------------------------------------------------------------

// attributionIdentityPolicyGetStub starts a hermetic httptest server that serves
// body for the attribution-identity-policy settings path, asserts the method and
// path, and counts the requests it observed.
func attributionIdentityPolicyGetStub(t *testing.T, body string) (*httptest.Server, *requestCounter) {
	t.Helper()
	counter := &requestCounter{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v2/api/teams/team-1/settings/attribution-identity-policy", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, counter
}

// wireAttributionIdentityPolicy points the package-level API client at srv and
// installs a formatter writing into outBuf, returning the formatter so a test
// can narrow its fields before running the command.
func wireAttributionIdentityPolicy(srv *httptest.Server, outBuf *bytes.Buffer, jsonMode bool) *output.Formatter {
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	f := output.NewWithWriter(outBuf, outBuf, jsonMode, false)
	cmd.Output = f
	return f
}

// TestAttributionIdentityPolicyGetRendersServerValue asserts the command issues
// exactly one GET to the settings path and renders the server's own policy
// value.
func TestAttributionIdentityPolicyGetRendersServerValue(t *testing.T) {
	srv, counter := attributionIdentityPolicyGetStub(t, `{"policy":"ALLOW_SELF_ASSERTED_UNVERIFIED"}`)

	var buf bytes.Buffer
	wireAttributionIdentityPolicy(srv, &buf, false)

	c := newAttributionIdentityPolicyGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "policy")
	assert.Contains(t, out, "ALLOW_SELF_ASSERTED_UNVERIFIED")
	assert.Equal(t, 1, counter.count(), "exactly one request must reach the server")
}

// TestAttributionIdentityPolicyGetRendersBlankVerbatim is the mutation-killing
// test for D-31-02 / T-31-09.
//
// It asserts the ABSENCE of a client-side blank-to-strict fallback rather than
// the presence of a feature, which is the only shape that stays red if someone
// later "helps" by substituting the strict value for a blank one. The server
// already returns an unset policy as the strict value, so a blank response is a
// genuine contract break and must stay visible as one instead of being masked
// behind a value the CLI invented.
func TestAttributionIdentityPolicyGetRendersBlankVerbatim(t *testing.T) {
	srv, counter := attributionIdentityPolicyGetStub(t, `{"policy":""}`)

	var buf bytes.Buffer
	wireAttributionIdentityPolicy(srv, &buf, false)

	c := newAttributionIdentityPolicyGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	assert.NotContains(t, buf.String(), "VERIFIED_DOMAIN_ONLY",
		"a blank policy must render blank; substituting the strict value would mask a server contract break")
	assert.Equal(t, 1, counter.count())
}

// TestAttributionIdentityPolicyGetFieldsNarrowsColumns is D-31-33's
// requirement: assert that --fields actually narrows the rendered columns, not
// merely that the flag is accepted. Phase 30's mitigation gate verified flag
// existence and the T-30-11 defect shipped anyway, because the renderer
// bypassed the filtering entry point. Routing through cmd.Output.Render is what
// this test pins.
func TestAttributionIdentityPolicyGetFieldsNarrowsColumns(t *testing.T) {
	srv, _ := attributionIdentityPolicyGetStub(t, `{"policy":"ALLOW_SELF_ASSERTED_UNVERIFIED"}`)

	var buf bytes.Buffer
	f := wireAttributionIdentityPolicy(srv, &buf, false)
	f.SetFields([]string{"Value"})

	c := newAttributionIdentityPolicyGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "Value", "the retained column header must still render")
	assert.NotContains(t, out, "Setting", "--fields Value must drop the Setting column")
}

// TestAttributionIdentityPolicyGetPrintsNoNotice pins D-31-06's read half: the
// enforcement notice belongs on `set`, immediately before the write, and NOT on
// every read. A stderr line on every read trains operators to ignore stderr, at
// which point the notice that matters is the one they miss.
func TestAttributionIdentityPolicyGetPrintsNoNotice(t *testing.T) {
	srv, _ := attributionIdentityPolicyGetStub(t, `{"policy":"ALLOW_SELF_ASSERTED_UNVERIFIED"}`)

	var outBuf, errBuf bytes.Buffer
	wireAttributionIdentityPolicy(srv, &outBuf, false)

	c := newAttributionIdentityPolicyGetCmd()
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs([]string{"team-1"})
	require.NoError(t, c.Execute())

	assert.Empty(t, errBuf.String(), "get must write nothing to the command's error writer")
	assert.NotEmpty(t, outBuf.String(), "the table must still have been rendered to stdout")
}

// TestAttributionIdentityPolicyGetRegisteredUnderTeams pins the command at its
// exact path in the package-level tree — the same object main.go registers.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a Name()-only
// assertion stays green on a command hung off the wrong parent.
func TestAttributionIdentityPolicyGetRegisteredUnderTeams(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"attribution-identity-policy", "get"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "teams attribution-identity-policy get", resolved.CommandPath())
}
