package jobs

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// factsAppendFlagNames is the single flag `facts append` declares.
var factsAppendFlagNames = []string{"file"}

// resetFactsAppendFlags restores the append command's flags to their unset
// state.
//
// initFacts() constructs newFactsAppendCmd() exactly once, at package
// initialization, so every test in this binary drives the SAME command object
// with the same flag values and the same Changed bits. Without this reset a
// test that supplied --file would leave that path visible to a later test that
// meant to omit it, and the later test would pass or fail according to source
// order rather than according to behaviour. The name differs from
// resetAppendFlags (baselines_append_test.go) because that name is already
// taken in this package.
func resetFactsAppendFlags(t *testing.T) {
	t.Helper()
	c, _, err := Cmd.Find([]string{"types", "facts", "append"})
	require.NoError(t, err)
	for _, name := range factsAppendFlagNames {
		fl := c.Flags().Lookup(name)
		require.NotNil(t, fl, "flag --%s must be registered", name)
		require.NoError(t, fl.Value.Set(fl.DefValue))
		fl.Changed = false
	}
}

// writeTempFactsFile writes body to a file under t's temp dir and returns the
// path. The path is what the error messages must name, so tests capture it.
// Named for facts because writeTempEconomicsFile is already taken in this
// package; facts_doc_test.go uses this same helper.
func writeTempFactsFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "facts.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

// decodeFactEntries decodes a fixture into the array shape the pipeline works
// in, so a test can assert that what reached the wire equals what the operator
// supplied without hand-building the expectation twice.
func decodeFactEntries(t *testing.T, body string) []map[string]interface{} {
	t.Helper()
	var entries []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(body), &entries))
	return entries
}

// twoFactEntries is a two-entry PeriodFactEntry array carrying all ten
// properties on the first entry and the documented-optional ones omitted on
// the second.
//
// The first entry's value is 1234567.89 deliberately: a number at or above 1e6
// is where a re-formatting or rounding bug would show, and this command must
// forward the number it read untouched.
const twoFactEntries = `[
  {
    "periodStart": "2026-01-01T00:00:00Z",
    "periodEnd": "2026-01-31T23:59:59Z",
    "dimensionKey": "region",
    "dimensionValue": "us-east",
    "key": "documents_reviewed",
    "value": 1234567.89,
    "provenance": "MEASURED",
    "recordedBy": "ops@acme.example",
    "source": "warehouse-export",
    "reason": "monthly close"
  },
  {
    "periodStart": "2026-02-01T00:00:00Z",
    "periodEnd": "2026-02-28T23:59:59Z",
    "dimensionKey": "region",
    "dimensionValue": "eu-west",
    "key": "quality_rate",
    "value": 0.955,
    "provenance": "SELF_REPORTED"
  }
]`

// TestFactsAppendFile is JOBS-15's happy path: an array read from a file
// reaches the wire once, at the escaped path, with teamId on the query string
// and the body byte-equal to what was supplied.
func TestFactsAppendFile(t *testing.T) {
	resetFactsAppendFlags(t)

	var counter requestCounter
	var got []map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/jobs/types/acme-review/facts", r.URL.Path)
		assert.Equal(t, "team-28", r.URL.Query().Get("teamId"),
			"teamId is required by the endpoint and must reach the wire")
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, twoFactEntries)
	require.NoError(t, execJobs(&buf,
		"types", "facts", "append", "acme-review", "--file", path))

	// The decoded body, not merely its length: the array is forwarded
	// verbatim, so any key renaming or value coercion shows up here.
	require.Len(t, got, 2)
	assert.Equal(t, decodeFactEntries(t, twoFactEntries), got)
	assert.Equal(t, 1, counter.count(),
		"exactly one POST per invocation — there is no per-entry loop")
	assert.Contains(t, buf.String(), "Appended 2 period facts to job type acme-review.")
}

// TestFactsAppendHandlesEmpty201 is this phase's highest-value assertion.
//
// POST /v2/api/jobs/types/{type}/facts declares a 201 with NO content block at
// all, and internal/api.Client.Do decodes the response body whenever its
// result argument is non-nil. Passing a decode target therefore turns every
// successful append into "failed to decode response: EOF" — a permanent write
// reported to the operator as a failure.
//
// The stub below writes the status line and NOTHING else: no content-type
// header, no body, not even an empty JSON document. That is the point. A stub
// that wrote `{}` after the 201 would keep the defective form green and this
// test would prove nothing.
func TestFactsAppendHandlesEmpty201(t *testing.T) {
	resetFactsAppendFlags(t)

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, twoFactEntries)
	require.NoError(t, execJobs(&buf,
		"types", "facts", "append", "acme-review", "--file", path),
		"a 201 carrying no body is the documented success and must not surface as a decode error")

	assert.Equal(t, 1, counter.count())
	assert.Contains(t, buf.String(), "Appended 2 period facts",
		"success must be reported from what was sent — the server returned nothing to report")
}

// execJobsWithStdin runs args through the package-level Cmd with in attached as
// the command's input reader, which is what `--file -` reads through.
//
// The reader is attached to Cmd rather than to os.Stdin so the pipe path is
// exercised without process-level plumbing — the deviation readFactEntries
// inherits from readEconomicsInput and the reason it takes a reader at all.
func execJobsWithStdin(out io.Writer, in io.Reader, args ...string) error {
	Cmd.SetIn(in)
	defer Cmd.SetIn(nil)
	return execJobs(out, args...)
}

// TestFactsAppendStdin: `--file -` reads the injected reader and posts what it
// read, exactly once.
func TestFactsAppendStdin(t *testing.T) {
	resetFactsAppendFlags(t)

	var counter requestCounter
	var got []map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.Equal(t, "/v2/api/jobs/types/acme-review/facts", r.URL.Path)
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobsWithStdin(&buf, strings.NewReader(twoFactEntries),
		"types", "facts", "append", "acme-review", "--file", "-"))

	assert.Equal(t, decodeFactEntries(t, twoFactEntries), got)
	assert.Equal(t, 1, counter.count())
}

// TestFactsAppendRejectsProvenance is the command-level half of SC2. The
// validator's own message is pinned in facts_doc_test.go; what this test adds
// is the only assertion that can prove "before any HTTP request is made" —
// the stub's counter reading ZERO.
//
// An equality assertion, not a zero-value one: require.Error alone passes with
// the validation gate deleted, because a team-less or malformed request fails
// too, and an unrelated error satisfying the same assertion is not evidence.
func TestFactsAppendRejectsProvenance(t *testing.T) {
	resetFactsAppendFlags(t)

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, `[
	  {"key":"documents_reviewed","value":1,"provenance":"MEASURED"},
	  {"key":"documents_reviewed","value":2,"provenance":"DERIVED"},
	  {"key":"documents_reviewed","value":3,"provenance":"ATTESTED"},
	  {"key":"documents_reviewed","value":4,"provenance":"MEASUED"}
	]`)

	err := execJobs(&buf, "types", "facts", "append", "acme-review", "--file", path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry [3].provenance")
	assert.Equal(t, 0, counter.count(),
		"a refused entry means no request object was ever built — an appended fact cannot be withdrawn")
}

// TestFactsAppendRateBound is the command-level half of the rate bound. Same
// contract as above: the counter, not the error, is what proves the refusal
// happened before the wire.
func TestFactsAppendRateBound(t *testing.T) {
	resetFactsAppendFlags(t)

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, `[{"key":"quality_rate","value":1.5}]`)

	err := execJobs(&buf, "types", "facts", "append", "acme-review", "--file", path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry [0].value")
	assert.Equal(t, 0, counter.count())
}

// TestFactsAppendProvenanceOptional pins that an omitted provenance reaches the
// wire as an ABSENT key, not as an empty string or a null the CLI invented.
//
// The schema documents a server-side default ("Defaults to SELF_REPORTED when
// omitted"), and only an absent key lets the server apply it.
func TestFactsAppendProvenanceOptional(t *testing.T) {
	resetFactsAppendFlags(t)

	var counter requestCounter
	var got []map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, `[
	  {"key":"documents_reviewed","value":10},
	  {"key":"documents_reviewed","value":11}
	]`)

	require.NoError(t, execJobs(&buf,
		"types", "facts", "append", "acme-review", "--file", path))

	require.Len(t, got, 2)
	for i, entry := range got {
		assert.NotContains(t, entry, "provenance",
			"entry %d must reach the wire with no provenance key at all", i)
	}
	assert.Equal(t, 1, counter.count())
}

// TestFactsAppendPreservesOrderAndDuplicates pins that the CLI is a courier.
//
// Two entries carrying identical key, period and dimension are two
// measurements, and the endpoint is append-only: collapsing them into one
// would discard something the operator recorded. Order matters for the same
// reason every refusal message names an index — a reordered array makes
// "entry [2]" point at the wrong line of the operator's file.
//
// The 1234567.89 value is the precision half: the document travels as
// map[string]interface{} end to end, so any rounding, unit conversion or
// re-formatting the CLI applied of its own accord would show up here.
func TestFactsAppendPreservesOrderAndDuplicates(t *testing.T) {
	resetFactsAppendFlags(t)

	const fourEntriesTwoIdentical = `[
	  {"periodStart":"2026-01-01T00:00:00Z","key":"documents_reviewed","value":1234567.89},
	  {"periodStart":"2026-02-01T00:00:00Z","dimensionKey":"region","dimensionValue":"us-east","key":"quality_rate","value":0.5},
	  {"periodStart":"2026-02-01T00:00:00Z","dimensionKey":"region","dimensionValue":"us-east","key":"quality_rate","value":0.5},
	  {"periodStart":"2026-03-01T00:00:00Z","key":"documents_reviewed","value":3}
	]`

	var counter requestCounter
	var got []map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, fourEntriesTwoIdentical)
	require.NoError(t, execJobs(&buf,
		"types", "facts", "append", "acme-review", "--file", path))

	want := decodeFactEntries(t, fourEntriesTwoIdentical)
	require.Len(t, got, 4, "nothing may be merged, deduplicated or dropped")
	assert.Equal(t, want[1], got[1])
	assert.Equal(t, want[2], got[2])
	assert.Equal(t, got[1], got[2], "the duplicate pair must both survive")
	assert.Equal(t, want[0], got[0], "the array must reach the wire in source order")
	assert.Equal(t, 1234567.89, got[0]["value"],
		"the number must arrive as it was written — no rounding, no re-formatting")
	assert.Equal(t, 1, counter.count(),
		"one POST per invocation regardless of entry count — there is no per-entry loop")
}

// TestFactsAppendDryRunIssuesNoPost is the guard that stops a preview from
// performing the write it previews.
//
// --dry-run is a ROOT persistent flag, so cobra accepts it on this command
// whether or not anybody reads it, README promises it previews any mutation
// without executing it, and cmd/schema.go publishes this command's `mutating`
// annotation to automation. An appended fact is permanent and the API offers
// no deletion, so an unread --dry-run is a guard failing open on an
// irreversible write.
//
// The load-bearing assertion is the counter. Neither exit 0 nor the
// "No changes were made." footer can tell a refused request from one that was
// issued and happened to succeed — the renderer prints the footer either way.
func TestFactsAppendDryRunIssuesNoPost(t *testing.T) {
	resetFactsAppendFlags(t)
	setRootFlag(t, "dry-run", "true")

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, twoFactEntries)
	require.NoError(t, execJobs(&buf,
		"types", "facts", "append", "acme-review", "--file", path),
		"a dry run is a question, not an attempt: it exits 0")

	assert.Equal(t, 0, counter.count(),
		"--dry-run must issue no POST — an appended fact is permanent and cannot be withdrawn")

	out := buf.String()
	assert.Contains(t, out, "Dry run: append job type period facts")
	assert.Contains(t, out, "/v2/api/jobs/types/acme-review/facts",
		"the operator must be told the resolved path")
	assert.Contains(t, out, "2 entries",
		"and the parsed entry count")
	assert.Contains(t, out, "(parsed; provenance and quality_rate checked)",
		"the preview must name the two checks that ran, not claim the entries were validated")
	assert.NotContains(t, out, "parsed and validated",
		"required-ness, shape and declared-ness are NOT checked here — an entry with no key, "+
			"an undeclared metric, or no fields at all passes this path, so a bare \"validated\" "+
			"green-lights a permanent append the server will reject")
	assert.Contains(t, out, "No changes were made.")
}

// TestFactsAppendDryRunBeatsYes pins D-27-06's ordering: --dry-run is checked
// before anything else and wins over --yes. An operator who typed both asked a
// question, and answering it with a permanent write is the worst reading of
// the two flags.
func TestFactsAppendDryRunBeatsYes(t *testing.T) {
	resetFactsAppendFlags(t)
	setRootFlag(t, "dry-run", "true")
	setRootFlag(t, "yes", "true")

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, twoFactEntries)
	require.NoError(t, execJobs(&buf,
		"types", "facts", "append", "acme-review", "--file", path))

	assert.Equal(t, 0, counter.count(),
		"--yes skips a confirmation; it does not turn a preview into a write")
}
