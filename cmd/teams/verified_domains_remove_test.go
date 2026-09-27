package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// TEAM-12 — `revenium teams verified-domains remove <team-id> <domain>`.
//
// Every test in this file carries the TestVerifiedDomainsRemove prefix
// deliberately: the plan's verification gate runs `-run TestVerifiedDomainsRemove`
// and counts PASS lines, and a selector that matches nothing exits 0 while
// proving nothing. Every case is FLAT rather than a t.Run subtest, for the
// reason recorded in verified_domains_list_test.go's header.
//
// resource.ConfirmDelete auto-confirms when stdin is not a TTY, which is the
// case under `go test` — so these tests do not need to drive the prompt and do
// not hang. Where a test wants the explicit skip path, it registers and passes
// --yes, which at runtime is a PERSISTENT ROOT flag rather than one declared on
// this command (the delete_test.go convention).
// ---------------------------------------------------------------------------

// verifiedDomainsRemoveRecorder keeps SEPARATE method counters, because two of
// the properties these tests assert are about which requests were NOT made: a
// pre-flight GET, or a second DELETE that client-side de-duplication swallowed.
// It also records the last observed query value, which is the string the server
// matches against stored rows.
type verifiedDomainsRemoveRecorder struct {
	get      requestCounter
	del      requestCounter
	lastPath string
	lastQry  string
}

// verifiedDomainsRemoveStub serves statuses[i] for the i-th request it sees,
// reusing the last entry once the slice is exhausted. That is what lets the
// repeatability test model a domain that exists on the first call and is gone
// on the second.
func verifiedDomainsRemoveStub(t *testing.T, statuses ...int) (*httptest.Server, *verifiedDomainsRemoveRecorder) {
	t.Helper()
	rec := &verifiedDomainsRemoveRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rec.get.inc()
		case http.MethodDelete:
			rec.del.inc()
		}
		rec.lastPath = r.URL.Path
		rec.lastQry = r.URL.Query().Get("domain")

		idx := rec.get.count() + rec.del.count() - 1
		status := http.StatusNoContent
		if len(statuses) > 0 {
			if idx >= len(statuses) {
				idx = len(statuses) - 1
			}
			status = statuses[idx]
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status >= 400 {
			fmt.Fprint(w, `{"message":"not found"}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// TestVerifiedDomainsRemoveSendsDeleteWithQueryDomain asserts the request shape:
// the domain travels as a required QUERY parameter, not as a path segment or a
// body, and exactly one DELETE is issued with no read before it.
func TestVerifiedDomainsRemoveSendsDeleteWithQueryDomain(t *testing.T) {
	srv, rec := verifiedDomainsRemoveStub(t, http.StatusNoContent)

	var buf bytes.Buffer
	wireVerifiedDomains(srv, &buf, false)

	c := newVerifiedDomainsRemoveCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1", "acme.example", "--yes"})
	require.NoError(t, c.Execute())

	assert.Equal(t, "/v2/api/teams/team-1/settings/verified-domains", rec.lastPath)
	assert.Equal(t, "acme.example", rec.lastQry)
	assert.Equal(t, 1, rec.del.count(), "exactly one DELETE")
	assert.Equal(t, 0, rec.get.count(), "no pre-flight read")
	assert.Contains(t, buf.String(), "Removed verified domain acme.example from team team-1.")
}

// TestVerifiedDomainsRemoveSendsDomainVerbatim is the sharp end of D-31-12.
//
// The `domain` query parameter is REQUIRED and is matched against stored rows
// by a server whose normalisation the CLI cannot observe. A client-side
// lowercase or trim that disagrees with the server's is a delete that removes
// NOTHING while the command reports success — a silent miss on the phase's only
// destructive surface. The operator's string is sent as typed; a 404 is the
// honest outcome.
func TestVerifiedDomainsRemoveSendsDomainVerbatim(t *testing.T) {
	srv, rec := verifiedDomainsRemoveStub(t, http.StatusNoContent)

	var buf bytes.Buffer
	wireVerifiedDomains(srv, &buf, false)

	c := newVerifiedDomainsRemoveCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1", "ACME.Example", "--yes"})
	require.NoError(t, c.Execute())

	assert.Equal(t, "ACME.Example", rec.lastQry,
		"the query value must be byte-for-byte what the operator typed (D-31-12)")
}

// TestVerifiedDomainsRemoveJSONEmitsJSONObject is code review finding WR-04.
//
// This is the SUCCESS path of the only destructive command the phase added, and
// it is reachable non-interactively: resource.ConfirmDelete auto-confirms under
// --json (and on any non-TTY stdin), so a script really does arrive here. A
// prose sentence written onto that stream makes json.Unmarshal fail on the one
// outcome a caller most needs to confirm — the removal succeeded.
//
// The assertion is a real json.Unmarshal rather than a substring check, because
// a substring check passes on a document with a sentence appended after it.
//
// The sibling discipline this copies is cmd/sessions/attribution.go, where the
// full-detail hint sits under an IsJSON guard for exactly this reason, pinned by
// TestSessionsAttributionNarrowColumns' assertion that the hint never lands in a
// JSON document.
func TestVerifiedDomainsRemoveJSONEmitsJSONObject(t *testing.T) {
	srv, rec := verifiedDomainsRemoveStub(t, http.StatusNoContent)

	var buf bytes.Buffer
	wireVerifiedDomains(srv, &buf, true)

	c := newVerifiedDomainsRemoveCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1", "acme.example", "--yes"})
	require.NoError(t, c.Execute())

	assert.Equal(t, 1, rec.del.count(), "exactly one DELETE")
	assert.Equal(t, 0, rec.get.count(), "no pre-flight read")

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc),
		"the whole stdout stream must parse as one JSON document (WR-04)")
	assert.Equal(t, true, doc["removed"])
	assert.Equal(t, "acme.example", doc["domain"])
	assert.Equal(t, "team-1", doc["teamId"])
	assert.NotContains(t, buf.String(), "Removed verified domain",
		"the prose sentence must not share the stream with the document")
}

// TestVerifiedDomainsRemoveIsRepeatable pins the idempotency posture: a second
// remove of an already-removed domain sends the same DELETE again and relays
// the server's 404 unchanged rather than reinterpreting it. There is no
// pre-flight GET, which the zero GET counter asserts.
func TestVerifiedDomainsRemoveIsRepeatable(t *testing.T) {
	srv, rec := verifiedDomainsRemoveStub(t, http.StatusNoContent, http.StatusNotFound)

	var firstBuf bytes.Buffer
	wireVerifiedDomains(srv, &firstBuf, false)
	first := newVerifiedDomainsRemoveCmd()
	first.Flags().Bool("yes", false, "Skip confirmation prompts")
	first.SetOut(&firstBuf)
	first.SetArgs([]string{"team-1", "acme.example", "--yes"})
	require.NoError(t, first.Execute())

	var secondBuf bytes.Buffer
	wireVerifiedDomains(srv, &secondBuf, false)
	second := newVerifiedDomainsRemoveCmd()
	second.Flags().Bool("yes", false, "Skip confirmation prompts")
	second.SetOut(&secondBuf)
	second.SetErr(&secondBuf)
	second.SilenceUsage = true
	second.SilenceErrors = true
	second.SetArgs([]string{"team-1", "acme.example", "--yes"})
	require.Error(t, second.Execute(), "the server's 404 must be relayed, not swallowed")

	assert.Equal(t, 2, rec.del.count(), "each invocation sends its own DELETE")
	assert.Equal(t, 0, rec.get.count(), "no pre-flight read may sit between the operator's intent and the write")
}

// TestVerifiedDomainsRemoveDryRunDoesNotPrompt pins the dry-run output contract
// by invoking dryrun.Render directly with the exact verb, resource label, path
// and nil body that verified_domains_remove.go's RunE passes. The path INCLUDES
// the domain query, because that is the request the operator is being shown.
//
// cmd.DryRun() has no exported setter (see cmd/organizations/create_test.go
// precedent). The other half of this property — that the dry-run branch appears
// strictly BEFORE the resource.ConfirmDelete call, so a dry run never prompts —
// is asserted by source position in the plan's verification gate, which is the
// only place it can be observed without a TTY.
func TestVerifiedDomainsRemoveDryRunDoesNotPrompt(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	path := "/v2/api/teams/team-1/settings/verified-domains?domain=acme.example"
	require.NoError(t, dryrun.Render(out, "remove", "verified domain", path, nil))

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: remove verified domain")
	assert.Contains(t, rendered, path)
	assert.Contains(t, rendered, "domain=acme.example")
	assert.Contains(t, rendered, "No changes were made.")
}

// TestVerifiedDomainsRemoveDryRunGateReturnsBeforeConfirmDelete closes the
// second half of the ordering claim that TestVerifiedDomainsRemoveDryRunDoesNotPrompt's
// comment could previously only assert in prose: that the `cmd.DryRun()`
// branch in verified_domains_remove.go's RunE appears, and returns, BEFORE
// the `resource.ConfirmDelete` call is reached.
//
// Why source inspection rather than an end-to-end `--dry-run` invocation:
// `cmd.DryRun()` reads an unexported package-level var in package `cmd` that
// is bound ONLY to rootCmd's persistent `--dry-run` flag (cmd/root.go). There
// is no exported setter and rootCmd itself is unexported, so no test outside
// package `cmd` — including every test in this file — can flip real dry-run
// state and drive the actual RunE through Execute(). This is a pre-existing,
// repo-wide constraint documented identically at
// cmd/guardrails/budget_rules_delete_test.go and cmd/teams/logo_test.go's
// TestTeamLogoSetDryRun; it is not something introduced by this plan, and it
// is out of reach without adding a test-only setter to cmd/root.go, which is
// an implementation file this test suite must not modify.
//
// What this test DOES verify, mechanically and can fail on: it reads the
// actual RunE source and asserts the byte offset of the `cmd.DryRun()` guard
// occurs strictly before the byte offset of the `resource.ConfirmDelete(`
// call, AND that a `return` statement sits between them (so the guard is a
// short-circuit, not a fallthrough that merely renders a dry-run summary and
// keeps going). A regression that reorders the two, or that drops the
// `return`, flips this test red.
func TestVerifiedDomainsRemoveDryRunGateReturnsBeforeConfirmDelete(t *testing.T) {
	src, err := os.ReadFile("verified_domains_remove.go")
	require.NoError(t, err, "must be able to read the RunE source to pin the ordering invariant")
	text := string(src)

	dryRunIdx := strings.Index(text, "cmd.DryRun()")
	require.Greater(t, dryRunIdx, -1, "expected a cmd.DryRun() guard in verified_domains_remove.go")

	confirmIdx := strings.Index(text, "resource.ConfirmDelete(")
	require.Greater(t, confirmIdx, -1, "expected a resource.ConfirmDelete call in verified_domains_remove.go")

	require.Less(t, dryRunIdx, confirmIdx,
		"the cmd.DryRun() gate must appear before resource.ConfirmDelete so a dry run never reaches the confirmation prompt")

	between := text[dryRunIdx:confirmIdx]
	assert.Contains(t, between, "return",
		"the cmd.DryRun() branch must return before falling through to resource.ConfirmDelete, not merely render a summary and continue")
}

// TestVerifiedDomainsRemoveDryRunRenderIssuesNoRequests is defense in depth
// alongside TestVerifiedDomainsRemoveDryRunDoesNotPrompt: it points the SAME
// dry-run render call at a stub server that fails the test if ANY request
// reaches it, confirming dryrun.Render itself performs no I/O. It does not,
// and cannot, prove the full command's real `--dry-run` flag path is wired to
// this render call without the RunE invocation described above — that is
// covered by TestVerifiedDomainsRemoveDryRunGateReturnsBeforeConfirmDelete.
func TestVerifiedDomainsRemoveDryRunRenderIssuesNoRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request during dry-run render: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)

	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	path := "/v2/api/teams/team-1/settings/verified-domains?domain=acme.example"
	require.NoError(t, dryrun.Render(out, "remove", "verified domain", path, nil))

	assert.Contains(t, buf.String(), "No changes were made.")
}

// TestVerifiedDomainsRemoveRegisteredUnderTeams pins the command at its exact
// path, and makes D-31-10's naming decision assertable: the sub-command set is
// exactly list, add and remove — no `set` and no `delete`.
//
// The count filters cobra's injected `help` and `completion` sub-commands.
// cobra's ExecuteC injects both into whichever command it treats as ROOT, so a
// raw len() in a package whose tests also run commands is order-dependent and
// passes or fails on which test ran first (recorded by 31-03 and 31-04).
func TestVerifiedDomainsRemoveRegisteredUnderTeams(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"verified-domains", "remove"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "teams verified-domains remove", resolved.CommandPath())

	var names []string
	for _, sub := range verifiedDomainsCmd.Commands() {
		if sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		names = append(names, sub.Name())
	}
	assert.Len(t, names, 3, "the group declares exactly three verbs")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "remove")
	assert.NotContains(t, names, "set", "D-31-10: an append is not a set")
	assert.NotContains(t, names, "delete", "D-31-10: the collection verb is remove")
}
