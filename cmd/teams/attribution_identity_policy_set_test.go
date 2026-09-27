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

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// TEAM-11 — `revenium teams attribution-identity-policy set <team-id> --policy V`.
//
// Every test carries the TestAttributionIdentityPolicySet prefix: the plan's
// gate runs `-run TestAttributionIdentityPolicySet` and counts PASS lines, and a
// selector matching nothing exits 0 while proving nothing. For the same reason
// no test here uses t.Run — `go test -v` prints an extra indented PASS line per
// subtest, which the gate's substring grep would also count.
//
// requestCounter (mutex-guarded, and the mutex is load-bearing under -race) is
// declared once in pr_health_get_test.go and shared across package teams.
// ---------------------------------------------------------------------------

// noticeFragment is the substring the enforcement notice must carry. Asserting a
// fragment rather than the whole sentence keeps the tests from breaking on a
// wording tweak while still failing if the notice is dropped, silenced, or made
// conditional.
const noticeFragment = "not currently enforced"

// attributionIdentityPolicySetStub is a hermetic stub for the settings path with
// SEPARATE per-method counters. The GET counter makes "no read before the write"
// directly assertable; the total counter is what proves a client-side refusal
// cost zero HTTP rather than merely produced an error.
type attributionIdentityPolicySetStub struct {
	srv          *httptest.Server
	gets         requestCounter
	puts         requestCounter
	total        requestCounter
	receivedBody map[string]interface{}
	receivedPath string
}

func newAttributionIdentityPolicySetStub(t *testing.T, responseBody string) *attributionIdentityPolicySetStub {
	t.Helper()
	stub := &attributionIdentityPolicySetStub{}
	stub.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.total.inc()
		switch r.Method {
		case http.MethodGet:
			stub.gets.inc()
		case http.MethodPut:
			stub.puts.inc()
		}
		stub.receivedPath = r.URL.Path
		if body, err := io.ReadAll(r.Body); err == nil && len(body) > 0 {
			_ = json.Unmarshal(body, &stub.receivedBody)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, responseBody)
	}))
	t.Cleanup(stub.srv.Close)
	return stub
}

// wireAttributionIdentityPolicySet points the package-level client at the stub
// and installs a formatter writing into outBuf — deliberately NOT into the
// command's error buffer, so a notice that leaked through the output formatter
// would land in stdout where TestAttributionIdentityPolicySetNoticeGoesToStderr
// can see it.
func wireAttributionIdentityPolicySet(stub *attributionIdentityPolicySetStub, outBuf *bytes.Buffer, jsonMode bool) {
	cmd.APIClient = api.NewClient(stub.srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(outBuf, outBuf, jsonMode, false)
}

// execAttributionIdentityPolicySet runs the set command with args, capturing
// stdout and stderr in SEPARATE buffers — the separation is the whole point of
// D-31-06's channel claim.
func execAttributionIdentityPolicySet(outBuf, errBuf *bytes.Buffer, args ...string) error {
	c := newAttributionIdentityPolicySetCmd()
	c.SetOut(outBuf)
	c.SetErr(errBuf)
	c.SilenceUsage = true
	c.SetArgs(args)
	return c.Execute()
}

// TestAttributionIdentityPolicySetSendsPolicy is the happy path: one PUT whose
// body carries exactly the single `policy` key, no GET before it, and the PUT's
// own 200 re-rendered through the shared renderer (D-31-05).
func TestAttributionIdentityPolicySetSendsPolicy(t *testing.T) {
	stub := newAttributionIdentityPolicySetStub(t, `{"policy":"VERIFIED_DOMAIN_ONLY"}`)

	var outBuf, errBuf bytes.Buffer
	wireAttributionIdentityPolicySet(stub, &outBuf, false)

	require.NoError(t, execAttributionIdentityPolicySet(&outBuf, &errBuf,
		"team-1", "--policy", "VERIFIED_DOMAIN_ONLY"))

	assert.Equal(t, "/v2/api/teams/team-1/settings/attribution-identity-policy", stub.receivedPath)
	assert.Equal(t, 1, stub.puts.count(), "exactly one PUT")
	assert.Equal(t, 0, stub.gets.count(), "the command must never read before it writes")

	require.Len(t, stub.receivedBody, 1, "the body carries exactly the one required property")
	assert.Equal(t, "VERIFIED_DOMAIN_ONLY", stub.receivedBody["policy"])

	assert.Contains(t, outBuf.String(), "policy")
	assert.Contains(t, outBuf.String(), "VERIFIED_DOMAIN_ONLY")
}

// TestAttributionIdentityPolicySetNoticeIsUnconditional pins D-31-07.
//
// The tempting optimisation is to warn only on the strict value — the one the
// platform currently ignores — and stay quiet on the permissive one, which
// matches current behaviour anyway. That encodes a client-side belief about
// which server-side gate is live right now, and becomes silently wrong in the
// OTHER direction the moment the gate is activated. This test is what kills it.
//
// Written as a loop rather than t.Run subtests for the PASS-count reason above.
func TestAttributionIdentityPolicySetNoticeIsUnconditional(t *testing.T) {
	for _, policy := range []string{"VERIFIED_DOMAIN_ONLY", "ALLOW_SELF_ASSERTED_UNVERIFIED"} {
		stub := newAttributionIdentityPolicySetStub(t, fmt.Sprintf(`{"policy":%q}`, policy))

		var outBuf, errBuf bytes.Buffer
		wireAttributionIdentityPolicySet(stub, &outBuf, false)

		require.NoError(t, execAttributionIdentityPolicySet(&outBuf, &errBuf, "team-1", "--policy", policy))

		assert.Contains(t, errBuf.String(), noticeFragment,
			"the notice must be printed for %s too — conditioning it on the value is the failure mode", policy)
		assert.Equal(t, 1, stub.puts.count(), "the notice must not replace the write")
	}
}

// TestAttributionIdentityPolicySetNoticeGoesToStderr pins T-31-07: the notice is
// an extra line, so it must never land inside a --json document where it would
// corrupt a machine consumer's parse. The mechanical reason it cannot is that it
// goes to the cobra command's error writer and never through cmd.Output, whose
// writer is stdout (internal/output/output.go).
//
// The JSON-mode half is the one that matters: table mode tolerates stray lines,
// a parsed document does not.
func TestAttributionIdentityPolicySetNoticeGoesToStderr(t *testing.T) {
	// Table mode: present on stderr, absent from stdout.
	tableStub := newAttributionIdentityPolicySetStub(t, `{"policy":"VERIFIED_DOMAIN_ONLY"}`)
	var tableOut, tableErr bytes.Buffer
	wireAttributionIdentityPolicySet(tableStub, &tableOut, false)

	require.NoError(t, execAttributionIdentityPolicySet(&tableOut, &tableErr,
		"team-1", "--policy", "VERIFIED_DOMAIN_ONLY"))

	assert.Contains(t, tableErr.String(), noticeFragment)
	assert.NotContains(t, tableOut.String(), noticeFragment,
		"the notice must not reach stdout")

	// JSON mode: stdout must still be a clean document.
	jsonStub := newAttributionIdentityPolicySetStub(t, `{"policy":"VERIFIED_DOMAIN_ONLY"}`)
	var jsonOut, jsonErr bytes.Buffer
	wireAttributionIdentityPolicySet(jsonStub, &jsonOut, true)

	require.NoError(t, execAttributionIdentityPolicySet(&jsonOut, &jsonErr,
		"team-1", "--policy", "VERIFIED_DOMAIN_ONLY"))

	assert.Contains(t, jsonErr.String(), noticeFragment)
	assert.NotContains(t, jsonOut.String(), noticeFragment,
		"a --json consumer's document must be intact")

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonOut.Bytes(), &decoded),
		"stdout must parse as a single JSON document")
}

// TestAttributionIdentityPolicySetRejectsUnknownPolicy pins D-31-08.
//
// The counter assertion PRECEDES require.Error deliberately (the cmd/jobs
// economics-set precedent): a test that only asserts an error came back still
// passes when the guard is deleted and the server 400s. The zero request count
// is the assertion that names the actual property — refused at zero HTTP cost.
func TestAttributionIdentityPolicySetRejectsUnknownPolicy(t *testing.T) {
	stub := newAttributionIdentityPolicySetStub(t, `{}`)

	var outBuf, errBuf bytes.Buffer
	wireAttributionIdentityPolicySet(stub, &outBuf, false)

	err := execAttributionIdentityPolicySet(&outBuf, &errBuf, "team-1", "--policy", "NOPE")

	assert.Equal(t, 0, stub.total.count(), "no request of any method may be sent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "NOPE")
	assert.Contains(t, err.Error(), "VERIFIED_DOMAIN_ONLY")
	assert.Contains(t, err.Error(), "ALLOW_SELF_ASSERTED_UNVERIFIED")
}

// TestAttributionIdentityPolicySetRejectsCaseVariant pins the adjacency edge:
// the comparison is EXACT against the schema's declared values. Normalising the
// input first would accept a casing the schema never declared and the server
// never promised to take — the CLI would be inventing an accepted input.
func TestAttributionIdentityPolicySetRejectsCaseVariant(t *testing.T) {
	stub := newAttributionIdentityPolicySetStub(t, `{}`)

	var outBuf, errBuf bytes.Buffer
	wireAttributionIdentityPolicySet(stub, &outBuf, false)

	err := execAttributionIdentityPolicySet(&outBuf, &errBuf, "team-1", "--policy", "verified_domain_only")

	assert.Equal(t, 0, stub.total.count(), "no request of any method may be sent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "VERIFIED_DOMAIN_ONLY")
	assert.Contains(t, err.Error(), "ALLOW_SELF_ASSERTED_UNVERIFIED")
}

// TestAttributionIdentityPolicySetRequiresPolicyFlag covers the two shapes of an
// under-specified call, flat rather than as subtests: the flag omitted entirely,
// and the flag passed explicitly empty. There is no legal empty body for a
// resource whose only property is required, so both must be refused before any
// request — and each is now refused by a DIFFERENT mechanism, which is what the
// two cases assert separately.
//
// The comment this replaces recorded cobra's PreRunE-before-ValidateRequiredFlags
// ordering as a known limitation and stated the enum guard reported BOTH cases.
// After 31-08 only the second half is true (code review IN-01). The PreRunE
// closure yields when `--policy` was never Changed, so cobra's own
// `required flag(s) "policy" not set` reports the omitted case: the
// MarkFlagRequired call has gone from a backstop this test could only pin with a
// grep gate to the mechanism an operator actually reads.
//
// The explicitly-empty case stays the enum guard's, and must: cobra considers a
// flag passed as "" to be supplied, so its validation would let an empty body
// through. That is the one shape only the enum guard can see, which is why the
// negative assertion below (the omitted case must NOT name the enum values) is
// the discriminator between the two mechanisms rather than a stylistic check.
func TestAttributionIdentityPolicySetRequiresPolicyFlag(t *testing.T) {
	// Omitted entirely — cobra's required-flag validation.
	omittedStub := newAttributionIdentityPolicySetStub(t, `{}`)
	var omittedOut, omittedErr bytes.Buffer
	wireAttributionIdentityPolicySet(omittedStub, &omittedOut, false)

	omittedErrResult := execAttributionIdentityPolicySet(&omittedOut, &omittedErr, "team-1")

	assert.Equal(t, 0, omittedStub.total.count(), "no request of any method may be sent")
	require.Error(t, omittedErrResult)
	assert.Contains(t, omittedErrResult.Error(), "required flag",
		"an omitted --policy must be reported by cobra, naming the missing flag")
	assert.NotContains(t, omittedErrResult.Error(), "VERIFIED_DOMAIN_ONLY",
		"the enum guard must no longer report a flag the operator never passed")

	// Passed explicitly empty — the enum guard, which cobra cannot replace here.
	emptyStub := newAttributionIdentityPolicySetStub(t, `{}`)
	var emptyOut, emptyErr bytes.Buffer
	wireAttributionIdentityPolicySet(emptyStub, &emptyOut, false)

	emptyErrResult := execAttributionIdentityPolicySet(&emptyOut, &emptyErr, "team-1", "--policy", "")

	assert.Equal(t, 0, emptyStub.total.count(), "an empty policy must not produce a request")
	require.Error(t, emptyErrResult)
	assert.Contains(t, emptyErrResult.Error(), "VERIFIED_DOMAIN_ONLY")
	assert.Contains(t, emptyErrResult.Error(), "ALLOW_SELF_ASSERTED_UNVERIFIED")
}

// TestAttributionIdentityPolicySetDryRun pins the dry-run render contract by
// invoking dryrun.Render directly with the exact verb, resource label, path and
// body shape the RunE passes when the dry-run gate is taken. cmd.DryRun() has no
// exported setter (cmd/organizations/create_test.go precedent), so the branch
// itself is a single `if`; that the mutating annotation and the gate are BOTH
// present is pinned by the plan's grep gate — Phase 27's Critical was a command
// that declared itself mutating and then performed the real write anyway.
func TestAttributionIdentityPolicySetDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	body := map[string]interface{}{"policy": "VERIFIED_DOMAIN_ONLY"}

	err := dryrun.Render(out, "update", "attribution identity policy",
		"/v2/api/teams/team-1/settings/attribution-identity-policy", body)
	require.NoError(t, err)

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: update attribution identity policy")
	assert.Contains(t, rendered, "/v2/api/teams/team-1/settings/attribution-identity-policy")
	assert.Contains(t, rendered, "No changes were made.")
}

// TestAttributionIdentityPolicySetRegisteredUnderTeams pins the command at its
// exact path in the package-level tree — the same object main.go registers.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a Name()-only
// assertion stays green on a command hung off the wrong parent.
func TestAttributionIdentityPolicySetRegisteredUnderTeams(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"attribution-identity-policy", "set"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "teams attribution-identity-policy set", resolved.CommandPath())
}
