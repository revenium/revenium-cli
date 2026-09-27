package jobs

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	rerrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
)

// economicsRequestBody is a JobTypeEconomicsRequest carrying all SEVEN request
// properties (including overheadCurrency, correction C1) and declaring exactly
// what stub_test.go's economicsResourceBody declares.
//
// Diffed against that resource it loses nothing, so it is the non-destructive
// fixture: every rung that must proceed without --yes uses it.
const economicsRequestBody = `{
  "unitMetricKey": "documents-reviewed",
  "unitLabel": "documents reviewed",
  "metrics": [
    {"key":"documents-reviewed","type":"COUNT","direction":"HIGHER_IS_BETTER","aggregation":"SUM","resolution":"PER_JOB"}
  ],
  "dimensions": [
    {"key":"region","allowedValues":["us-east","eu-west"]}
  ],
  "monetization": {
    "metricKey":"documents-reviewed",
    "valuePerUnit":12.5,
    "currency":"USD",
    "category":"COST_AVOIDED",
    "basis":"REALIZED"
  },
  "overheadPerUnit": 1234567,
  "overheadCurrency": "USD"
}`

// economicsDestructiveBody drops the "documents-reviewed" metric that
// economicsResourceBody declares and changes nothing else.
//
// Exactly ONE loss, deliberately: the refusal message is asserted to name that
// metric by key, and a fixture losing three things at once would let a message
// naming only the first of them pass.
const economicsDestructiveBody = `{
  "unitMetricKey": "documents-reviewed",
  "unitLabel": "documents reviewed",
  "metrics": [
    {"key":"review-accuracy","type":"PERCENT","direction":"HIGHER_IS_BETTER","aggregation":"AVG","resolution":"PERIOD"}
  ],
  "dimensions": [
    {"key":"region","allowedValues":["us-east","eu-west"]}
  ],
  "monetization": {
    "metricKey":"documents-reviewed",
    "valuePerUnit":12.5,
    "currency":"USD",
    "category":"COST_AVOIDED",
    "basis":"REALIZED"
  },
  "overheadPerUnit": 1234567,
  "overheadCurrency": "USD"
}`

// economicsSetPath is the single path all four request kinds must use.
const economicsSetPath = "/v2/api/jobs/types/acme-review/economics"

// --- harness -----------------------------------------------------------------

// economicsSetStub answers the preflight GET and the PUT, counting each
// SEPARATELY and recording what the PUT received.
//
// Two counters, not one, because the distinguishing fact in half of these tests
// is *which* request was or was not issued. A refusal that still issued its
// preflight GET is correct; a refusal that issued the PUT is the exact defect
// the guard exists to prevent. A single combined counter cannot tell those
// apart, and "the request count is 1" would be satisfied by either.
type economicsSetStub struct {
	t *testing.T

	getStatus int
	getBody   string
	putStatus int
	putBody   string

	gets requestCounter
	puts requestCounter

	mu        sync.Mutex
	putMethod string
	putPath   string
	putTeamID string
	putSeen   map[string]interface{}
}

func (s *economicsSetStub) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.gets.inc()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(s.getStatus)
			fmt.Fprint(w, s.getBody)
		case http.MethodPut:
			s.puts.inc()
			var got map[string]interface{}
			assert.NoError(s.t, json.NewDecoder(r.Body).Decode(&got))
			s.mu.Lock()
			s.putMethod = r.Method
			s.putPath = r.URL.Path
			s.putTeamID = r.URL.Query().Get("teamId")
			s.putSeen = got
			s.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(s.putStatus)
			fmt.Fprint(w, s.putBody)
		default:
			s.t.Errorf("unexpected %s request to %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// received returns the decoded body the PUT carried, or nil if no PUT arrived.
func (s *economicsSetStub) received() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putSeen
}

// resetEconomicsSetFlags restores `economics set`'s local --file flag to its
// unset state.
//
// initEconomics() constructs newEconomicsSetCmd() exactly ONCE, at package
// initialization, so every test in this binary drives the same command object
// with the same flag variable and the same Changed bit. Without the
// `fl.Changed = false` line below, a test that omits --file would still observe
// Changed("file") == true from a test that ran earlier — cobra's required-flag
// validation would pass, the stale path from the previous test would be read,
// and a document nobody in this test asked for would be PUT.
// TestEconomicsSetRequiresFile is red without that line.
func resetEconomicsSetFlags(t *testing.T) {
	t.Helper()

	c, _, err := Cmd.Find([]string{"types", "economics", "set"})
	require.NoError(t, err)
	fl := c.Flags().Lookup("file")
	require.NotNil(t, fl, "flag --file must be registered")
	require.NoError(t, fl.Value.Set(fl.DefValue))
	fl.Changed = false

	// The two root flags are package-level variables in cmd and live for the
	// whole test binary, so they are cleared here as well as restored by
	// setRootFlag's cleanup — belt and braces, because a leaked --dry-run would
	// silently turn every later mutating test in cmd/jobs into a no-op that
	// still passed.
	setRootFlag(t, "dry-run", "false")
	setRootFlag(t, "yes", "false")
}

// setRootFlag sets one of the ROOT command's persistent flags for the duration
// of the test and restores it afterwards.
//
// --yes and --dry-run are declared on rootCmd (cmd/root.go:220-221). In this
// test binary jobs.Cmd has no parent — main.go's registration never runs — so
// neither flag can be passed as an argument to the command under test; cobra
// would reject it as unknown. Setting the pflag.Value writes through to the
// very variable cmd.YesMode() and cmd.DryRun() read, so both the accessor and
// the flag-to-variable binding are exercised rather than bypassed.
func setRootFlag(t *testing.T, name, value string) {
	t.Helper()
	fl := cmd.Root().PersistentFlags().Lookup(name)
	require.NotNil(t, fl, "root persistent flag --%s must exist", name)

	prevValue := fl.Value.String()
	prevChanged := fl.Changed
	require.NoError(t, fl.Value.Set(value))
	fl.Changed = value != fl.DefValue

	t.Cleanup(func() {
		_ = fl.Value.Set(prevValue)
		fl.Changed = prevChanged
	})
}

// withNonTTYStdin points os.Stdin at a pipe for the duration of the test.
//
// The refusal rung turns on term.IsTerminal(os.Stdin.Fd()), and a pipe is never
// a terminal. Making that a FACT of the test rather than an accident of how
// `go test` happened to be invoked is what keeps the phase's most important
// guard test deterministic: run from an interactive shell on a toolchain that
// forwards the terminal to the test binary, the same assertion would otherwise
// fall through to the interactive prompt rung and block on a read.
func withNonTTYStdin(t *testing.T) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	prev := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = prev
		_ = w.Close()
		_ = r.Close()
	})
}

// execEconomicsSet runs args through the package-level Cmd with stdout and
// stderr captured SEPARATELY.
//
// stub_test.go's execJobs points both at one writer, which suits its callers
// but would let a stderr assertion here pass against text that actually landed
// on stdout. "--json stdout stays machine-clean while the loss summary still
// reaches the operator" is one of the properties these tests exist to pin, and
// it is unobservable through a single buffer.
func execEconomicsSet(out, errOut io.Writer, in io.Reader, args ...string) error {
	Cmd.SetOut(out)
	Cmd.SetErr(errOut)
	Cmd.SetIn(in)
	defer Cmd.SetIn(nil)
	Cmd.SetArgs(args)
	return Cmd.Execute()
}

// setupEconomicsSet wires the stub, the client and the formatter, and returns
// the stdout and stderr buffers.
func setupEconomicsSet(t *testing.T, s *economicsSetStub) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	s.t = t
	resetEconomicsSetFlags(t)

	srv := newStubServer(t, s.handler())
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(out, errOut, false, false)
	return out, errOut
}

// writeEconomicsFile writes contents to a temp file and returns its path.
func writeEconomicsFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "economics.json")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

// assertSevenRequestProperties checks the decoded PUT body against every
// JobTypeEconomicsRequest property, by its WIRE key.
func assertSevenRequestProperties(t *testing.T, got map[string]interface{}) {
	t.Helper()
	require.NotNil(t, got, "the PUT must have carried a body")

	assert.Equal(t, "documents-reviewed", got["unitMetricKey"])
	assert.Equal(t, "documents reviewed", got["unitLabel"])
	assert.Equal(t, 1234567.0, got["overheadPerUnit"])
	assert.Equal(t, "USD", got["overheadCurrency"])

	metrics, ok := got["metrics"].([]interface{})
	require.True(t, ok, "metrics must survive as an array")
	require.Len(t, metrics, 1)
	metric, ok := metrics[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "documents-reviewed", metric["key"])
	assert.Equal(t, "COUNT", metric["type"])

	dimensions, ok := got["dimensions"].([]interface{})
	require.True(t, ok, "dimensions must survive as an array")
	require.Len(t, dimensions, 1)

	monetization, ok := got["monetization"].(map[string]interface{})
	require.True(t, ok, "monetization must survive as an object")
	assert.Equal(t, "COST_AVOIDED", monetization["category"])
	assert.Equal(t, "REALIZED", monetization["basis"])

	assert.Len(t, got, 7, "exactly the seven JobTypeEconomicsRequest properties, no more")
}

// --- tests -------------------------------------------------------------------

// TestEconomicsSetFile is JOBS-12's happy path from a file: a Request-shaped
// document that drops nothing is PUT to the escaped path with teamId on the
// query, and the seven properties arrive with the values supplied.
func TestEconomicsSetFile(t *testing.T) {
	stub := &economicsSetStub{
		getStatus: http.StatusOK,
		getBody:   economicsResourceBody,
		putStatus: http.StatusOK,
		putBody:   economicsResourceBody,
	}
	out, errOut := setupEconomicsSet(t, stub)
	path := writeEconomicsFile(t, economicsRequestBody)

	require.NoError(t, execEconomicsSet(out, errOut, nil,
		"types", "economics", "set", "acme-review", "--file", path))

	assert.Equal(t, 1, stub.gets.count(), "exactly one preflight GET")
	assert.Equal(t, 1, stub.puts.count(), "exactly one PUT")
	assert.Equal(t, http.MethodPut, stub.putMethod)
	assert.Equal(t, economicsSetPath, stub.putPath)
	assert.Equal(t, "team-27", stub.putTeamID,
		"teamId is required by the endpoint and must reach the wire")

	assertSevenRequestProperties(t, stub.received())

	// SC2's "reading it back returns what was written", observable from the
	// command itself: the PUT response is rendered, not swallowed.
	assert.Contains(t, out.String(), "Contract")
	assert.Empty(t, errOut.String(),
		"a Request-shaped document drops nothing, so there is no note to print")
}

// TestEconomicsSetStdin is the same write driven through `--file -`, with the
// document supplied on the command's input reader.
func TestEconomicsSetStdin(t *testing.T) {
	stub := &economicsSetStub{
		getStatus: http.StatusOK,
		getBody:   economicsResourceBody,
		putStatus: http.StatusOK,
		putBody:   economicsResourceBody,
	}
	out, errOut := setupEconomicsSet(t, stub)

	require.NoError(t, execEconomicsSet(out, errOut, strings.NewReader(economicsRequestBody),
		"types", "economics", "set", "acme-review", "--file", "-"))

	assert.Equal(t, 1, stub.gets.count())
	assert.Equal(t, 1, stub.puts.count())
	assert.Equal(t, economicsSetPath, stub.putPath)
	assertSevenRequestProperties(t, stub.received())
}

// TestEconomicsSetStripsReadOnly is the D-27-03 round-trip: a document in the
// Resource shape `economics get --json` emits is accepted, and the two
// read-only keys never reach the wire.
//
// The load-bearing assertion is on the RECEIVED REQUEST BODY, not on stderr.
// The strip and the note are separately implemented, so a stderr-only
// assertion stays green with the stripping removed — which is the whole defect.
func TestEconomicsSetStripsReadOnly(t *testing.T) {
	stub := &economicsSetStub{
		getStatus: http.StatusOK,
		getBody:   economicsResourceBody,
		putStatus: http.StatusOK,
		putBody:   economicsResourceBody,
	}
	out, errOut := setupEconomicsSet(t, stub)
	path := writeEconomicsFile(t, economicsResourceBody)

	require.NoError(t, execEconomicsSet(out, errOut, nil,
		"types", "economics", "set", "acme-review", "--file", path))

	assert.Equal(t, 1, stub.puts.count())

	got := stub.received()
	require.NotNil(t, got)
	assert.NotContains(t, got, "jobType",
		"jobType is read-only and must never reach the request body")
	assert.NotContains(t, got, "currentBaseline",
		"currentBaseline is read-only and must never reach the request body")
	assertSevenRequestProperties(t, got)

	// The note is load-bearing per D-27-03 and belongs on stderr so --json
	// stdout stays machine-clean.
	note := errOut.String()
	assert.Contains(t, note, "jobType")
	assert.Contains(t, note, "currentBaseline")
	assert.Contains(t, note, "append-only",
		"the operator must be told the baseline did not move")
	assert.NotContains(t, out.String(), "read-only key",
		"the note goes to stderr, never to stdout")
}

// TestEconomicsSetRefusesDestructive is the phase's principal safety control.
//
// A document that drops a declared metric, with no --yes and nobody watching,
// must refuse — naming --yes and the metric that would be lost — and must issue
// NO PUT. The PUT counter is the load-bearing assertion: in Phase 27 plan 01 a
// deleted guard still satisfied require.Error, because the unguarded request
// reached the stub and failed on a decode. Only the counter caught it.
//
// MUTATION (recorded in the SUMMARY): replacing the body of
// confirmDestructiveReplace with a call into internal/resource.ConfirmDelete
// turns this red on the PUT counter, because that helper returns proceed when
// stdin is not a TTY.
func TestEconomicsSetRefusesDestructive(t *testing.T) {
	t.Run("a file path, refused via the TTY probe", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusOK,
			getBody:   economicsResourceBody,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)
		withNonTTYStdin(t)
		path := writeEconomicsFile(t, economicsDestructiveBody)

		err := execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path)

		// The counters come FIRST, and they are assert rather than require,
		// so a mutation that lets the write through reports the PUT count as
		// the failing assertion. Leading with require.Error would abort the
		// subtest before the counter ever ran, and "an error was expected"
		// says nothing about whether a contract was destroyed.
		assert.Equal(t, 1, stub.gets.count(), "the preflight GET is expected and correct")
		assert.Equal(t, 0, stub.puts.count(),
			"a destructive replace with no --yes and no TTY must issue no PUT")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "--yes",
			"the refusal must name the flag that unblocks it")
		assert.Contains(t, err.Error(), "documents-reviewed",
			"the refusal must name the metric that would be lost, not just a count")
	})

	t.Run("--file -, refused explicitly rather than by inference", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusOK,
			getBody:   economicsResourceBody,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)

		// No withNonTTYStdin here, deliberately: the `-` rung must refuse on
		// its own, without help from the TTY probe. stdin was already consumed
		// by the document, so there is nothing left to prompt on.
		err := execEconomicsSet(out, errOut, strings.NewReader(economicsDestructiveBody),
			"types", "economics", "set", "acme-review", "--file", "-")

		assert.Equal(t, 0, stub.puts.count(), "--file - must never write without --yes")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "--yes")
		assert.Contains(t, err.Error(), "documents-reviewed")
	})
}

// TestEconomicsSetYesProceeds pins the other side of the same rung: explicit
// consent is the path that writes.
func TestEconomicsSetYesProceeds(t *testing.T) {
	stub := &economicsSetStub{
		getStatus: http.StatusOK,
		getBody:   economicsResourceBody,
		putStatus: http.StatusOK,
		putBody:   economicsResourceBody,
	}
	out, errOut := setupEconomicsSet(t, stub)
	withNonTTYStdin(t)
	setRootFlag(t, "yes", "true")
	path := writeEconomicsFile(t, economicsDestructiveBody)

	require.NoError(t, execEconomicsSet(out, errOut, nil,
		"types", "economics", "set", "acme-review", "--file", path))

	assert.Equal(t, 1, stub.puts.count(), "--yes is the path that proceeds")
	assert.Contains(t, errOut.String(), "documents-reviewed",
		"--yes means `I accept this`, not `do not tell me` — the loss is still recorded")
}

// TestEconomicsSetDryRun pins T-27-02: --dry-run previews and never writes.
//
// MUTATION (recorded in the SUMMARY): deleting the early return in the dry-run
// branch turns the non-destructive subtest red on the PUT counter, and the
// destructive subtest red on its exit status.
//
// The two subtests are BOTH necessary and neither is redundant. With a
// destructive document, deleting the early return falls through to the refusal
// rung, which still issues no PUT — so the destructive case alone would report
// only "an unexpected error", and would leave "a dry run of a harmless edit
// writes nothing" completely unpinned. That is the case where a lost early
// return actually destroys data silently, and it is the non-destructive
// subtest that catches it.
func TestEconomicsSetDryRun(t *testing.T) {
	t.Run("a destructive replace previews and exits 0", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusOK,
			getBody:   economicsResourceBody,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)
		withNonTTYStdin(t)
		setRootFlag(t, "dry-run", "true")
		path := writeEconomicsFile(t, economicsDestructiveBody)

		err := execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path)

		assert.Equal(t, 1, stub.gets.count())
		assert.Equal(t, 0, stub.puts.count(), "--dry-run must issue no PUT")
		require.NoError(t, err,
			"a dry run of a destructive replace exits 0 — it is a question, not an attempt")

		// Legibility, not merely presence. Asserting only that stdout contains
		// "metrics" would pass against dryrun.Render's Go map syntax and prove
		// nothing (RESEARCH Pitfall 8), so the assertion is on the INDENTED
		// JSON shape a human can actually read.
		body := out.String()
		assert.Contains(t, body, "\"metrics\": [")
		assert.Contains(t, body, "\"unitMetricKey\": \"documents-reviewed\"")
		assert.Contains(t, body, "No changes were made.")

		// The operator must learn from the preview that the real run needs
		// --yes.
		assert.Contains(t, errOut.String(), "DESTRUCTIVE")
		assert.Contains(t, errOut.String(), "documents-reviewed")
	})

	// WR-04. dryrun.Render prints `  Body: %v` whenever body != nil
	// (internal/dryrun/dryrun.go), so passing the document to it after already
	// printing the document reproduced it a second time in Go map syntax —
	// non-deterministic key order and, for overheadPerUnit, the scientific
	// notation money()/num() and three separate tests exist to prevent. It
	// resurfaced in the ONE output surface an operator reads before committing
	// a replace.
	//
	// The assertions below are on exact, distinctive text. The original test
	// asserted only that stdout contained `"metrics": [` and "No changes were
	// made.", both satisfied by the first copy, so the duplicate was unpinned
	// in either direction.
	t.Run("the resolved body is printed once, and never as a Go map", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusOK,
			getBody:   economicsResourceBody,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)
		withNonTTYStdin(t)
		setRootFlag(t, "dry-run", "true")
		path := writeEconomicsFile(t, economicsRequestBody)

		require.NoError(t, execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path))

		body := out.String()
		assert.Contains(t, body, `"overheadPerUnit": 1234567`,
			"the readable, indented JSON copy is the one the operator gets")
		assert.NotContains(t, body, "1.234567e+06",
			"scientific notation must not reach the surface an operator reads before a replace")
		assert.NotContains(t, body, "Body: map[",
			"dryrun.Render must not re-print the document in Go map syntax")
		assert.Equal(t, 1, strings.Count(body, "overheadPerUnit"),
			"the resolved body is printed exactly once")
		assert.Contains(t, body, "No changes were made.",
			"the shared footer is still what closes the preview")
	})

	// The JSON path has no earlier copy to duplicate, so the body must still
	// reach dryrun.Render there. Fixing the duplicate by dropping the body
	// unconditionally would silently empty `--dry-run --json`, and no assertion
	// above would notice.
	t.Run("--dry-run --json still carries the resolved body", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusOK,
			getBody:   economicsResourceBody,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)
		cmd.Output = output.NewWithWriter(out, errOut, true, false)
		withNonTTYStdin(t)
		setRootFlag(t, "dry-run", "true")
		path := writeEconomicsFile(t, economicsRequestBody)

		require.NoError(t, execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path))

		assert.Equal(t, 0, stub.puts.count())

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(out.Bytes(), &payload),
			"--json stdout must stay machine-parseable")
		assert.Equal(t, true, payload["dry_run"])
		requestBody, ok := payload["body"].(map[string]interface{})
		require.True(t, ok, "the resolved body must still be carried under --json")
		assert.Equal(t, "documents-reviewed", requestBody["unitMetricKey"])
	})

	t.Run("a non-destructive replace still writes nothing", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusOK,
			getBody:   economicsResourceBody,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)
		withNonTTYStdin(t)
		setRootFlag(t, "dry-run", "true")
		path := writeEconomicsFile(t, economicsRequestBody)

		require.NoError(t, execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path))

		assert.Equal(t, 1, stub.gets.count())
		assert.Equal(t, 0, stub.puts.count(),
			"--dry-run writes nothing even when nothing would be lost — "+
				"no rung below it may be reached")
		assert.Contains(t, errOut.String(), "not destructive",
			"the preview must say the real run would need no --yes")
	})
}

// TestEconomicsSetDryRunBeatsYes pins D-27-06's precedence: dry run wins.
//
// MUTATION (recorded in the SUMMARY): moving the dry-run rung below the --yes
// rung turns this red on the PUT counter, while TestEconomicsSetDryRun stays
// green — which is exactly why the two tests are separate.
func TestEconomicsSetDryRunBeatsYes(t *testing.T) {
	stub := &economicsSetStub{
		getStatus: http.StatusOK,
		getBody:   economicsResourceBody,
		putStatus: http.StatusOK,
		putBody:   economicsResourceBody,
	}
	out, errOut := setupEconomicsSet(t, stub)
	withNonTTYStdin(t)
	setRootFlag(t, "dry-run", "true")
	setRootFlag(t, "yes", "true")
	path := writeEconomicsFile(t, economicsDestructiveBody)

	require.NoError(t, execEconomicsSet(out, errOut, nil,
		"types", "economics", "set", "acme-review", "--file", path))

	assert.Equal(t, 0, stub.puts.count(),
		"--dry-run wins over --yes and never writes")
}

// TestEconomicsSetPreflight pins T-27-03: a preflight failure that is not a 404
// blocks the write, and a 404 is the create case.
//
// MUTATION (recorded in the SUMMARY): degrading a non-404 preflight error to a
// warning and proceeding turns the first subtest red on the PUT counter.
func TestEconomicsSetPreflight(t *testing.T) {
	t.Run("a 500 blocks the PUT", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusInternalServerError,
			getBody:   `{"message":"backend unavailable"}`,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)
		path := writeEconomicsFile(t, economicsRequestBody)

		err := execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path)

		// Counters first, for the same reason as in
		// TestEconomicsSetRefusesDestructive: under a degrade-and-proceed
		// mutation the PUT succeeds and err is nil, so a leading require.Error
		// would abort before the assertion that names the actual damage.
		assert.Equal(t, 1, stub.gets.count())
		assert.Equal(t, 0, stub.puts.count(),
			"we cannot know what would be lost, so nothing may be written")

		require.Error(t, err)
		var apiErr *rerrors.APIError
		require.True(t, stderrors.As(err, &apiErr),
			"the preflight failure must surface unchanged, carrying its status")
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	})

	t.Run("a 404 is an empty current contract and creates", func(t *testing.T) {
		stub := &economicsSetStub{
			getStatus: http.StatusNotFound,
			getBody:   `{"message":"Team or job type not found."}`,
			putStatus: http.StatusOK,
			putBody:   economicsResourceBody,
		}
		out, errOut := setupEconomicsSet(t, stub)
		path := writeEconomicsFile(t, economicsRequestBody)

		require.NoError(t, execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path),
			"nothing exists to lose, so `set` creates without --yes")

		assert.Equal(t, 1, stub.gets.count())
		assert.Equal(t, 1, stub.puts.count())
		assertSevenRequestProperties(t, stub.received())
	})
}

// TestEconomicsSetIdempotent pins the resolved probe row: running the same
// document twice is a no-op the second time round.
//
// The PUT is a full replace, so the second run's diff against the contract the
// first run just wrote is empty — non-destructive, no --yes required, and the
// same body on the wire both times.
func TestEconomicsSetIdempotent(t *testing.T) {
	var (
		mu     sync.Mutex
		state  string
		bodies []map[string]interface{}
		gets   requestCounter
		puts   requestCounter
	)

	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gets.inc()
			mu.Lock()
			current := state
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if current == "" {
				// No contract declared yet — the create case.
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"message":"Team or job type not found."}`)
				return
			}
			fmt.Fprint(w, current)
		case http.MethodPut:
			puts.inc()
			raw, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var got map[string]interface{}
			assert.NoError(t, json.Unmarshal(raw, &got))
			mu.Lock()
			state = string(raw)
			bodies = append(bodies, got)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(raw)
		default:
			t.Errorf("unexpected %s request", r.Method)
		}
	})

	path := writeEconomicsFile(t, economicsRequestBody)

	for run := 1; run <= 2; run++ {
		resetEconomicsSetFlags(t)
		withNonTTYStdin(t)
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		cmd.APIClient = stubClient(srv.URL, "team-27")
		cmd.Output = output.NewWithWriter(out, errOut, false, false)

		require.NoError(t, execEconomicsSet(out, errOut, nil,
			"types", "economics", "set", "acme-review", "--file", path),
			"run %d must not require --yes: replacing a contract with itself loses nothing", run)
	}

	assert.Equal(t, 2, gets.count())
	assert.Equal(t, 2, puts.count())
	require.Len(t, bodies, 2)
	assert.Equal(t, bodies[0], bodies[1],
		"the same document produces the same request body on every run")
}

// TestEconomicsSetConflictSurfaces pins the backstop for T-27-17: the preflight
// GET and the PUT are two requests with no compare-and-swap, so a concurrent
// writer can make the diff stale. The API's 409 is the only signal available,
// and it must surface as a non-zero exit rather than as a success.
func TestEconomicsSetConflictSurfaces(t *testing.T) {
	stub := &economicsSetStub{
		getStatus: http.StatusOK,
		getBody:   economicsResourceBody,
		putStatus: http.StatusConflict,
		putBody:   `{"message":"economics contract was modified by another writer"}`,
	}
	out, errOut := setupEconomicsSet(t, stub)
	path := writeEconomicsFile(t, economicsRequestBody)

	err := execEconomicsSet(out, errOut, nil,
		"types", "economics", "set", "acme-review", "--file", path)

	require.Error(t, err, "a 409 must never be reported as a successful write")
	var apiErr *rerrors.APIError
	require.True(t, stderrors.As(err, &apiErr))
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
	assert.Contains(t, err.Error(), "modified by another writer",
		"the API's own message is the only account of what happened")
	assert.NotEqual(t, 0, rerrors.ExitCodeFor(err), "the exit code must be non-zero")
	assert.Equal(t, 1, stub.puts.count())
}

// TestEconomicsSetRequiresFile pins the sticky-flag reset itself.
//
// It is deliberately the LAST test in this file: by the time it runs, several
// earlier tests have supplied --file, so the flag variable holds a real path
// and its Changed bit is set. Without `fl.Changed = false` in
// resetEconomicsSetFlags, cobra's required-flag check would be satisfied by the
// previous test's invocation and this run would happily read and PUT a stale
// document — the exact class of defect plan 27-03 found and fixed with
// resetAppendFlags.
func TestEconomicsSetRequiresFile(t *testing.T) {
	stub := &economicsSetStub{
		getStatus: http.StatusOK,
		getBody:   economicsResourceBody,
		putStatus: http.StatusOK,
		putBody:   economicsResourceBody,
	}
	out, errOut := setupEconomicsSet(t, stub)

	err := execEconomicsSet(out, errOut, nil,
		"types", "economics", "set", "acme-review")

	require.Error(t, err, "--file is required and no earlier test may satisfy it")

	// The assertion is on COBRA'S OWN required-flag message, not merely on the
	// substring "file". Without the Changed reset the command runs with an
	// empty --file and fails inside readEconomicsInput with "failed to read
	// economics document from : open : no such file or directory" — which also
	// contains "file", is also an error, and also issues no request. A loose
	// assertion would stay green through the very regression this test exists
	// to catch, and would report the reset as pinned when it is not.
	assert.Contains(t, err.Error(), `required flag(s) "file" not set`,
		"the required-flag check must fire, which it only does when Changed was reset")
	assert.Equal(t, 0, stub.gets.count(), "no preflight may be issued without a document")
	assert.Equal(t, 0, stub.puts.count(), "no stale document may be written")
}

// TestEconomicsSetEnumValidation pins that `economics set` actually CALLS
// validateEconomicsDocument, and calls it before anything reaches the wire.
//
// This is a different property from the one economics_doc_test.go's six
// TestValidateEconomicsDocumentRejects* cases pin. Those prove the validator is
// correct; none of them proves the command uses it. Delete the
// validateEconomicsDocument call from newEconomicsSetCmd's RunE and all six of
// them stay green while a document carrying an invalid enum is PUT to the API —
// precisely the shape Phase 26 found five times, a control sitting green under
// the mutation it existed to catch.
//
// One subtest per enum, not one shared case: the command calls the validator
// once, but the validator walks metrics before monetization and returns the
// FIRST violation, so a single case would leave five of the six paths
// unexercised through the command. Widening any one list turns exactly its own
// subtest red.
//
// The fixture is economicsRequestBody with ONE token changed, so every case is
// non-destructive against economicsResourceBody. That matters: with a
// destructive fixture, deleting the validation call would fall through to the
// refusal rung, which issues no PUT either — the counter would read 0 under both
// the correct and the mutated code and would pin nothing (27-05's dry-run
// lesson, from the other direction).
//
// The expected legal values are LITERALS. An assertion that read
// metricTypeValues to build its expectation would move with the very slice it
// exists to pin (27-04).
func TestEconomicsSetEnumValidation(t *testing.T) {
	cases := []struct {
		name     string
		from, to string // the exact token replaced in economicsRequestBody
		path     string // the JSON path the message must name, WITH its array index
		legal    string // one legal value the message must list
	}{
		{"metrics[i].type", `"type":"COUNT"`, `"type":"TALLY"`, "metrics[0].type", "DURATION"},
		{"metrics[i].direction", `"direction":"HIGHER_IS_BETTER"`, `"direction":"BIGGER_IS_BETTER"`, "metrics[0].direction", "LOWER_IS_BETTER"},
		{"metrics[i].aggregation", `"aggregation":"SUM"`, `"aggregation":"MEDIAN"`, "metrics[0].aggregation", "AVG"},
		{"metrics[i].resolution", `"resolution":"PER_JOB"`, `"resolution":"PER_PERIOD"`, "metrics[0].resolution", "PERIOD"},
		{"monetization.category", `"category":"COST_AVOIDED"`, `"category":"GOODWILL"`, "monetization.category", "REVENUE"},
		{"monetization.basis", `"basis":"REALIZED"`, `"basis":"PROJECTED"`, "monetization.basis", "EXPECTED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, 1, strings.Count(economicsRequestBody, tc.from),
				"the token %s must appear exactly once, or the fixture edits the wrong field", tc.from)
			body := strings.Replace(economicsRequestBody, tc.from, tc.to, 1)

			stub := &economicsSetStub{
				getStatus: http.StatusOK,
				getBody:   economicsResourceBody,
				putStatus: http.StatusOK,
				putBody:   economicsResourceBody,
			}
			out, errOut := setupEconomicsSet(t, stub)
			path := writeEconomicsFile(t, body)

			err := execEconomicsSet(out, errOut, nil,
				"types", "economics", "set", "acme-review", "--file", path)

			// Counters first, and with assert: under the mutation the command
			// proceeds and writes, and a leading require.Error would abort
			// before the only assertion that names the damage ever ran.
			//
			// ZERO GETs, not merely zero PUTs. Validation runs ahead of the
			// preflight, so an invalid document must not cost a round trip at
			// all — and the GET counter is what distinguishes "the validator
			// refused" from "the ladder refused later".
			assert.Equal(t, 0, stub.gets.count(),
				"an invalid enum must be refused before any request is issued")
			assert.Equal(t, 0, stub.puts.count(),
				"a document with an invalid enum must never be written")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.path,
				"the message must name the offending path, array index included")
			assert.Contains(t, err.Error(), tc.legal,
				"the message must list the legal values, so a mistype does not send the operator to the spec")
		})
	}
}
