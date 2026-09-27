package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOutcomeUpdate is the happy-path test for
// `revenium jobs outcome-update <id> --reason <value>`. It pins the
// LOAD-BEARING contract that the request method is PATCH (not POST) to a
// path ending `/outcome` — the same path as the existing POST `outcome`
// command but a distinct verb (RES-07).
func TestOutcomeUpdate(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "corrected value"})
	err := c.Execute()

	require.NoError(t, err)

	assert.Equal(t, "PATCH", receivedMethod, "outcome-update must use PATCH, not POST")
	assert.Equal(t, "/v2/api/jobs/loan-app-1/outcome", receivedPath)
	assert.Equal(t, "corrected value", receivedBody["reason"])

	assert.Contains(t, buf.String(), "JMwX9g4")
	assert.Contains(t, buf.String(), "Process Loan")
}

// TestOutcomeUpdateOptionalFieldsGated asserts that omitted optional flags do
// NOT appear in the PATCH body, mirroring outcome_test.go's gating pattern.
func TestOutcomeUpdateOptionalFieldsGated(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "corrected value"})
	err := c.Execute()

	require.NoError(t, err)

	assert.Equal(t, "corrected value", receivedBody["reason"])

	_, hasExecutionStatus := receivedBody["executionStatus"]
	assert.False(t, hasExecutionStatus, "executionStatus must NOT be sent when --execution-status omitted")
	_, hasType := receivedBody["outcomeType"]
	assert.False(t, hasType, "outcomeType must NOT be sent when --outcome-type omitted")
	_, hasValue := receivedBody["outcomeValue"]
	assert.False(t, hasValue, "outcomeValue must NOT be sent when --outcome-value omitted")
	_, hasCurrency := receivedBody["outcomeCurrency"]
	assert.False(t, hasCurrency, "outcomeCurrency must NOT be sent when --outcome-currency omitted")
	_, hasMetadata := receivedBody["metadata"]
	assert.False(t, hasMetadata, "metadata must NOT be sent when --metadata omitted")
}

// TestOutcomeUpdateMissingReason confirms that omitting --reason surfaces a
// Cobra "required flag(s)" error BEFORE any HTTP call.
func TestOutcomeUpdateMissingReason(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1"}) // no --reason

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reason",
		"missing-flag error should mention 'reason' by name")
}

// TestOutcomeUpdateConflict pins the DISTINCT 409 copy (Pitfall 2 regression
// guard): outcome-update's 409 means "concurrent update conflict, retry" —
// NOT "already reported... immutable" like the existing outcome command's
// 409. The rendered error must contain "concurrent" and must NOT contain
// "immutable".
func TestOutcomeUpdateConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"timestamp":"2026-05-12T00:00:00Z","status":409,"error":"Conflict","message":"Concurrent update","path":"/v2/api/jobs/loan-app-1/outcome"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "retry"})
	err := c.Execute()

	require.Error(t, err)

	// The FULL substring, not just the word "concurrent", because the full
	// substring is what an external consumer depends on.
	//
	// The Hermes SDK detects an optimistic-concurrency rejection from this
	// command by matching this literal sentence fragment on stderr. It has
	// nothing else to match: the message deliberately contains neither "409"
	// nor "conflict" (Pitfall 2 keeps this copy distinct from outcome.go's),
	// and no structured status survives to the consumer either —
	// outcome_update.go:182 returns a bare fmt.Errorf, so the *rerrors.APIError
	// is dropped from the error chain and the --output json envelope reports
	// "status": 0 rather than 409 (measured against a live 409).
	//
	// A reword to, say, "concurrent modification detected" would keep the
	// broader "concurrent" assertion below green while silently breaking that
	// consumer. Pinning the literal fragment makes such a reword fail loudly
	// here, where a maintainer can decide whether to coordinate the change.
	assert.Contains(t, err.Error(), "concurrent outcome update detected",
		"the Hermes SDK matches this literal substring on stderr to detect an "+
			"optimistic-concurrency rejection; reword it only in coordination with that consumer")

	assert.Contains(t, err.Error(), "concurrent")
	// A separate contract from the fragment above: Pitfall 2 forbids reusing
	// outcome.go's "already reported... immutable" copy on this command's 409.
	assert.NotContains(t, err.Error(), "immutable",
		"outcome-update's 409 must NOT reuse outcome.go's 'immutable' copy (Pitfall 2)")
	assert.Contains(t, err.Error(), "loan-app-1")
}

// TestOutcomeUpdateUnprocessable pins the 422 copy: "no outcome has been
// reported yet" — distinct from the 409 copy above and from outcome.go's
// 409 entirely.
func TestOutcomeUpdateUnprocessable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"timestamp":"2026-05-12T00:00:00Z","status":422,"error":"Unprocessable Entity","message":"No outcome reported","path":"/v2/api/jobs/loan-app-1/outcome"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "first update"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no outcome has been reported yet")
	assert.Contains(t, err.Error(), "revenium jobs outcome loan-app-1")
}

// TestOutcomeUpdateDryRun pins the dry-run output contract by invoking
// dryrun.Render DIRECTLY with the exact path/action/resource/body shape that
// outcome_update.go's RunE will pass in the dry-run branch — mirroring
// outcome_test.go's TestOutcomeDryRun pattern. This decouples the contract
// assertion from the cmd.DryRun() global-flag toggle.
func TestOutcomeUpdateDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact path outcome_update.go's RunE constructs via
	// fmt.Sprintf("/v2/api/jobs/%s/outcome", url.PathEscape("loan-app-1")).
	path := "/v2/api/jobs/loan-app-1/outcome"

	// The exact body outcome_update.go's RunE builds for a reason-only call
	// with no optional flags set.
	body := map[string]interface{}{"reason": "preview"}

	err := dryrun.Render(out, "outcome-update", "job", path, body)

	require.NoError(t, err)
	rendered := buf.String()

	assert.Contains(t, rendered, "Dry run: outcome-update job")
	assert.Contains(t, rendered, "/v2/api/jobs/loan-app-1/outcome")
	assert.Contains(t, rendered, "reason")
	assert.Contains(t, rendered, "No changes were made.")
}

// ---------------------------------------------------------------------------
// --expected-entity-version (JOBS-19)
//
// Four tests, each named for the single assertion it owns. They share the
// separate-stdout/stderr-buffer shape from the cmd/teams precedent
// (attribution_identity_policy_set_test.go): a single shared buffer cannot tell
// a channel violation from correct behaviour, which is the whole point of the
// stdout-absent assertion below.
//
// The five pre-existing outcome_update tests above are deliberately UNMODIFIED.
// They are the corroborating control for the byte-identical-for-existing-callers
// claim: if the new key or the new advisory leaked into an unflagged run, they
// would be the ones to notice.
// ---------------------------------------------------------------------------

// expectedEntityVersionNoticeFragment is the substring the advisory must carry.
// Asserting a fragment rather than the whole sentence keeps these tests from
// breaking on a wording tweak while still failing if the advisory is dropped,
// made unconditional, or routed to the wrong stream. It is deliberately long
// enough that no unrelated line — including the rendered job table — can
// satisfy it by accident.
const expectedEntityVersionNoticeFragment = "accepts this value and discards it silently"

// outcomeUpdateStub is a hermetic PATCH stub that captures the RAW request body
// bytes as well as the decoded map. The raw bytes are what prove the JSON wire
// TYPE; a decoded map alone cannot distinguish 7 from "7" without a type
// assertion, and cannot distinguish 7 from 7e+06 at all.
type outcomeUpdateStub struct {
	srv     *httptest.Server
	rawBody string
	body    map[string]interface{}
}

func newOutcomeUpdateStub(t *testing.T) *outcomeUpdateStub {
	t.Helper()
	s := &outcomeUpdateStub{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		s.rawBody = string(raw)
		_ = json.Unmarshal(raw, &s.body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":"SUCCESS"}`)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

// execOutcomeUpdate wires the stub and the two SEPARATE buffers, then runs the
// command. cmd.Output's writer is stdout — that is the mechanical reason the
// stdout-absent assertion is meaningful, and the thing the M6 mutation breaks.
func execOutcomeUpdate(t *testing.T, s *outcomeUpdateStub, stdout, stderr *bytes.Buffer, args ...string) error {
	t.Helper()
	cmd.APIClient = api.NewClient(s.srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(stdout, stderr, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(stdout)
	c.SetErr(stderr)
	// A failure must surface as an error, not as a usage dump into the buffers
	// the assertions read.
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs(args)
	return c.Execute()
}

// TestOutcomeUpdateExpectedEntityVersionFlagSpelling owns the load-bearing
// external contract: the flag's NAME. A downstream consumer (the Hermes SDK)
// probes for the literal two-dash spelling `--expected-entity-version` in this
// command's help output and keeps its optimistic-concurrency path inert until it
// finds it. A rename is therefore a silent breaking change to that consumer,
// which is why this assertion pins the spelling in both places it is published:
// the flag set and the rendered help text.
func TestOutcomeUpdateExpectedEntityVersionFlagSpelling(t *testing.T) {
	c := newOutcomeUpdateCmd()

	f := c.Flags().Lookup("expected-entity-version")
	require.NotNil(t, f, "the flag must be registered at exactly this spelling")
	assert.Equal(t, "int64", f.Value.Type(),
		"an int64 flag makes cobra reject a non-integer before any request is formed")

	assert.Contains(t, c.UsageString(), "--expected-entity-version",
		"the literal two-dash spelling must appear in the rendered help text")
}

// TestOutcomeUpdateOmitsExpectedEntityVersionWhenUnset owns the
// existing-callers-unchanged claim (T-Q19-04): with the flag omitted the PATCH
// body must carry no expectedEntityVersion key AT ALL — not the key with a zero
// value — and no advisory may reach any stream.
func TestOutcomeUpdateOmitsExpectedEntityVersionWhenUnset(t *testing.T) {
	s := newOutcomeUpdateStub(t)
	var stdout, stderr bytes.Buffer

	require.NoError(t, execOutcomeUpdate(t, s, &stdout, &stderr,
		"loan-app-1", "--reason", "corrected value"))

	_, hasKey := s.body["expectedEntityVersion"]
	assert.False(t, hasKey,
		"expectedEntityVersion must NOT be sent when --expected-entity-version is omitted")
	assert.NotContains(t, s.rawBody, "expectedEntityVersion",
		"the key must be absent from the raw body, not merely decode to a zero value")

	assert.NotContains(t, stderr.String(), expectedEntityVersionNoticeFragment,
		"the advisory must NOT print when the operator never asked for optimistic concurrency")
	assert.NotContains(t, stdout.String(), expectedEntityVersionNoticeFragment)
}

// TestOutcomeUpdateSendsExpectedEntityVersionAsNumber owns three separate
// contracts of the flagged path: the wire TYPE (a JSON number, not a string and
// not scientific notation), the advisory's PRESENCE, and its CHANNEL.
//
// The channel half pins T-Q19-03: an advisory on stdout would be interleaved
// into a --json document and break every machine consumer's parse. It reaches
// stderr because it is printed with c.ErrOrStderr() and never through
// cmd.Output, whose writer is stdout (internal/output/output.go).
func TestOutcomeUpdateSendsExpectedEntityVersionAsNumber(t *testing.T) {
	s := newOutcomeUpdateStub(t)
	var stdout, stderr bytes.Buffer

	require.NoError(t, execOutcomeUpdate(t, s, &stdout, &stderr,
		"loan-app-1", "--reason", "corrected value", "--expected-entity-version", "7"))

	// (a) The raw bytes: an unquoted JSON number, adjacent to its key.
	assert.Contains(t, s.rawBody, `"expectedEntityVersion":7`,
		"the value must serialise as a bare JSON number, not a quoted string")

	// (b) The decoded dynamic type: encoding/json decodes every JSON number
	// into float64, so float64 here proves "number" and a string would prove
	// the opposite.
	raw, hasKey := s.body["expectedEntityVersion"]
	require.True(t, hasKey, "expectedEntityVersion must be sent when the flag is passed")
	assert.IsType(t, float64(0), raw,
		"a string here means the value was stringified on the way out")
	assert.Equal(t, float64(7), raw)

	// (c) The advisory is present on stderr.
	assert.Contains(t, stderr.String(), expectedEntityVersionNoticeFragment,
		"the advisory must print whenever the flag is supplied")

	// (d) ...and absent from stdout, where it would corrupt a --json parse.
	assert.NotContains(t, stdout.String(), expectedEntityVersionNoticeFragment,
		"the advisory must not reach stdout")
}

// TestOutcomeUpdateAdvisoryAvoidsConsumerMatcherTokens owns the durable half of
// T-Q19-02. The downstream Hermes SDK classifies this command's failures with
// `(^|[^0-9])409($|[^0-9])` OR `[Cc]onflict` over captured output. The advisory
// prints on FAILING runs too, so either token in its text would make an auth,
// network, validation or 500 failure be misreported to the operator as a
// stale-version rejection — our own output turning an occasional consumer
// misfire into a systematic one.
//
// The assertion targets the constant's runtime VALUE and the flag's Usage
// string, never the source file's bytes: outcome_update.go's comments
// legitimately discuss the 409 arm, so a grep over the source would be a
// self-invalidating gate. The wording of the advisory may be improved freely;
// what may not happen is a reword that reintroduces a trigger token silently.
func TestOutcomeUpdateAdvisoryAvoidsConsumerMatcherTokens(t *testing.T) {
	usage := newOutcomeUpdateCmd().Flags().Lookup("expected-entity-version").Usage

	for name, s := range map[string]string{
		"advisory constant": expectedEntityVersionNotDeclaredNotice,
		"flag usage string": usage,
	} {
		assert.NotContains(t, s, "409",
			"%s must not contain 409 — it would match the consumer's failure matcher", name)
		assert.NotContains(t, strings.ToLower(s), "conflict",
			"%s must not contain 'conflict' in any case — it would match the consumer's failure matcher", name)
	}
}
