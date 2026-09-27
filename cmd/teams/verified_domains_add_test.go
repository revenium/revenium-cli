package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// TEAM-12 — `revenium teams verified-domains add <team-id> <domain>`.
//
// Every test in this file carries the TestVerifiedDomainsAdd prefix
// deliberately: the plan's verification gate runs `-run TestVerifiedDomainsAdd`
// and counts PASS lines, and a selector that matches nothing exits 0 while
// proving nothing.
//
// Every case is FLAT rather than a t.Run subtest, for the reason recorded in
// verified_domains_list_test.go's header: `go test -v` prints an extra indented
// `--- PASS:` line per subtest, which the gate's substring grep also counts.
// ---------------------------------------------------------------------------

// verifiedDomainsAddStub starts a hermetic httptest server for the
// verified-domains settings path. It records the last decoded request body and
// keeps SEPARATE method counters, because two of the properties these tests
// assert are about which requests were NOT made — a pre-flight GET, or a second
// PUT that client-side de-duplication swallowed.
//
// The requestCounter type is declared once for package teams in
// pr_health_get_test.go; it is reused here, not redeclared.
type verifiedDomainsAddRecorder struct {
	get  requestCounter
	put  requestCounter
	body map[string]interface{}
	raw  string
}

func verifiedDomainsAddStub(t *testing.T, response string) (*httptest.Server, *verifiedDomainsAddRecorder) {
	t.Helper()
	rec := &verifiedDomainsAddRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rec.get.inc()
		case http.MethodPut:
			rec.put.inc()
		}
		assert.Equal(t, "/v2/api/teams/team-1/settings/verified-domains", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		rec.raw = string(raw)
		rec.body = nil
		_ = json.Unmarshal(raw, &rec.body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, response)
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// TestVerifiedDomainsAddSendsDomainVerbatim is D-31-12's assertion: the
// operator's string reaches the server byte-for-byte, with no client-side
// lowercasing, trimming or normalisation.
//
// The mixed-case case is the one that matters. VerifiedDomainResource.domain is
// documented as a lowercase verified domain — a statement about what the server
// STORES — while VerifiedDomainRequest.domain declares no pattern and no
// format. Normalising here would be the CLI guessing at a transform it cannot
// observe, and on the sibling `remove` path that guess becomes a delete that
// silently misses.
func TestVerifiedDomainsAddSendsDomainVerbatim(t *testing.T) {
	for _, domain := range []string{"acme.example", "ACME.Example"} {
		srv, rec := verifiedDomainsAddStub(t, `{"domain":"acme.example","source":"ADMIN","joinPolicy":"REQUEST"}`)

		var buf bytes.Buffer
		wireVerifiedDomains(srv, &buf, false)

		c := newVerifiedDomainsAddCmd()
		c.SetOut(&buf)
		c.SetArgs([]string{"team-1", domain})
		require.NoError(t, c.Execute(), "domain %q", domain)

		require.NotNil(t, rec.body, "domain %q: the request body must be valid JSON", domain)
		assert.Len(t, rec.body, 1, "domain %q: the request declares exactly one property", domain)
		assert.Equal(t, domain, rec.body["domain"],
			"domain %q must reach the server byte-for-byte (D-31-12)", domain)
		assert.Equal(t, 1, rec.put.count(), "domain %q: exactly one PUT", domain)
	}
}

// TestVerifiedDomainsAddSurfacesServerAssignedFields is D-31-13: the two
// properties the operator had no say in are rendered, not dropped. An operator
// who verifies a domain must not have to discover later that the platform
// attached a join policy to it.
func TestVerifiedDomainsAddSurfacesServerAssignedFields(t *testing.T) {
	srv, _ := verifiedDomainsAddStub(t, `{"domain":"acme.example","source":"ADMIN","joinPolicy":"REQUEST"}`)

	var buf bytes.Buffer
	wireVerifiedDomains(srv, &buf, false)

	c := newVerifiedDomainsAddCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1", "acme.example"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "acme.example")
	assert.Contains(t, out, "ADMIN", "the server-assigned source must be surfaced")
	assert.Contains(t, out, "REQUEST", "the server-assigned join policy must be surfaced")
}

// TestVerifiedDomainsAddHasNoSourceOrJoinPolicyFlags asserts the ABSENCE of the
// two flags it would be natural to add. VerifiedDomainRequest declares one
// property; flags for the other two would build a request the server ignores,
// and would tell an operator they chose something they did not.
func TestVerifiedDomainsAddHasNoSourceOrJoinPolicyFlags(t *testing.T) {
	c := newVerifiedDomainsAddCmd()
	assert.Nil(t, c.Flags().Lookup("source"), "--source must not exist: the request schema has no such property")
	assert.Nil(t, c.Flags().Lookup("join-policy"), "--join-policy must not exist: the request schema has no such property")

	srv, rec := verifiedDomainsAddStub(t, `{"domain":"acme.example","source":"ADMIN","joinPolicy":"REQUEST"}`)

	var buf bytes.Buffer
	wireVerifiedDomains(srv, &buf, false)

	run := newVerifiedDomainsAddCmd()
	run.SetOut(&buf)
	run.SetArgs([]string{"team-1", "acme.example"})
	require.NoError(t, run.Execute())

	require.NotNil(t, rec.body)
	assert.Len(t, rec.body, 1, "no server-assigned property may be added to the body")
}

// TestVerifiedDomainsAddIsRepeatable pins the idempotency posture: re-adding an
// existing domain sends the same PUT again and relays the server's answer.
// There is no pre-flight GET and no client-side de-duplication — which is what
// the zero GET counter and the PUT counter of 2 assert. An error-presence
// assertion could not tell the two apart.
func TestVerifiedDomainsAddIsRepeatable(t *testing.T) {
	srv, rec := verifiedDomainsAddStub(t, `{"domain":"acme.example","source":"ADMIN","joinPolicy":"REQUEST"}`)

	for i := 0; i < 2; i++ {
		var buf bytes.Buffer
		wireVerifiedDomains(srv, &buf, false)

		c := newVerifiedDomainsAddCmd()
		c.SetOut(&buf)
		c.SetArgs([]string{"team-1", "acme.example"})
		require.NoError(t, c.Execute(), "run %d", i+1)
	}

	assert.Equal(t, 2, rec.put.count(), "each invocation sends its own PUT — no client-side de-duplication")
	assert.Equal(t, 0, rec.get.count(), "no pre-flight read may sit between the operator's intent and the write")
}

// TestVerifiedDomainsAddDryRun pins the dry-run output contract by invoking
// dryrun.Render directly with the exact verb, resource label, path and body
// shape that verified_domains_add.go's RunE passes when cmd.DryRun() is true.
// cmd.DryRun() has no exported setter (see cmd/organizations/create_test.go
// precedent), so the branch itself is a trivial single `if`, pinned separately
// by the plan's `grep -c 'cmd.DryRun()' == 1` gate.
func TestVerifiedDomainsAddDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	body := map[string]interface{}{"domain": "acme.example"}

	require.NoError(t, dryrun.Render(out, "add", "verified domain", "/v2/api/teams/team-1/settings/verified-domains", body))

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: add verified domain")
	assert.Contains(t, rendered, "/v2/api/teams/team-1/settings/verified-domains")
	assert.Contains(t, rendered, "No changes were made.")
}

// TestVerifiedDomainsAddRegisteredUnderTeams pins the command at its exact path
// in the package-level tree — the same object main.go registers. CommandPath()
// is the load-bearing half: Cmd.Find returns the deepest partial match it could
// reach and reports no error of its own.
func TestVerifiedDomainsAddRegisteredUnderTeams(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"verified-domains", "add"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "teams verified-domains add", resolved.CommandPath())
}
