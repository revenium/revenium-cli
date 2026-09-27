package jobs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// outcomeMetricsFlagNames is the single flag `outcome-metrics` declares.
var outcomeMetricsFlagNames = []string{"file"}

// resetOutcomeMetricsFlags restores the command's flags to their unset state.
//
// newOutcomeMetricsCmd() is constructed exactly once, by this package's
// initializer, so every test in this binary drives the SAME command object
// with the same flag values and the same Changed bits. Without this reset a
// test that supplied --file would leave that path visible to a later test that
// meant to omit it, and the later test would pass or fail according to source
// order rather than according to behaviour.
//
// The name differs from resetAppendFlags (baselines_append_test.go) and from
// resetFactsAppendFlags (facts_append_test.go) because both of those names are
// already taken in this package.
func resetOutcomeMetricsFlags(t *testing.T) {
	t.Helper()
	c, _, err := Cmd.Find([]string{"outcome-metrics"})
	require.NoError(t, err)
	require.Equal(t, "outcome-metrics <agenticJobId>", c.Use,
		"the reset must resolve the outcome-metrics command itself, not a partial match on its parent")
	for _, name := range outcomeMetricsFlagNames {
		fl := c.Flags().Lookup(name)
		require.NotNil(t, fl, "flag --%s must be registered", name)
		require.NoError(t, fl.Value.Set(fl.DefValue))
		fl.Changed = false
	}
}

// twoOutcomeMetricEntries is a two-entry OutcomeMetricEntry array.
//
// The shape is PeriodFactEntry minus the period window and the dimension pair,
// plus recordedAt — the one field that tells this entry kind from the other,
// and therefore the one whose survival across the round trip is worth pinning.
//
// The first entry's value is 1234567.89 deliberately: a number at or above 1e6
// is where a re-formatting or rounding bug would show, and this command must
// forward the number it read untouched.
const twoOutcomeMetricEntries = `[
  {
    "key": "documents_reviewed",
    "value": 1234567.89,
    "recordedAt": "2026-02-01T09:15:00Z",
    "provenance": "MEASURED",
    "recordedBy": "ops@acme.example",
    "source": "warehouse-export"
  },
  {
    "key": "quality_rate",
    "value": 0.955,
    "recordedAt": "2026-02-02T11:30:45Z",
    "provenance": "SELF_REPORTED"
  }
]`

// TestOutcomeMetricsFile is JOBS-16's happy path: an array read from a file
// reaches the wire once, at the escaped path, with teamId on the query string
// and the body byte-equal to what was supplied.
func TestOutcomeMetricsFile(t *testing.T) {
	resetOutcomeMetricsFlags(t)

	var counter requestCounter
	var got []map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1/outcome/metrics", r.URL.Path)
		assert.Equal(t, "team-28", r.URL.Query().Get("teamId"),
			"teamId is required by the endpoint and must reach the wire")
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, twoOutcomeMetricEntries)
	require.NoError(t, execJobs(&buf,
		"outcome-metrics", "loan-app-1", "--file", path))

	// The decoded body, not merely its length: the array is forwarded
	// verbatim, so any key renaming or value coercion shows up here.
	require.Len(t, got, 2)
	assert.Equal(t, decodeFactEntries(t, twoOutcomeMetricEntries), got)
	assert.Equal(t, 1, counter.count(),
		"exactly one POST per invocation — there is no per-entry loop")
	assert.Contains(t, buf.String(), "Appended 2 outcome metrics to job loan-app-1.")
}

// TestOutcomeMetricsStdin pins the pipe path and, with it, the field that
// distinguishes this entry shape from the period-fact shape.
//
// recordedAt is not a key the CLI knows about: the document travels as
// []map[string]interface{} end to end and is never decoded into a typed
// struct, which is precisely what stops a key the CLI does not model from
// being dropped on the way to the wire. Asserting on the DECODED value rather
// than on a substring of the raw body is what makes that claim, rather than
// the weaker claim that the bytes happened to appear somewhere.
func TestOutcomeMetricsStdin(t *testing.T) {
	resetOutcomeMetricsFlags(t)

	var counter requestCounter
	var got []map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.Equal(t, "/v2/api/jobs/loan-app-1/outcome/metrics", r.URL.Path)
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	// The reader is injected through the command, never through the process's
	// standard input — the deviation readFactEntries inherits from
	// readEconomicsInput and the reason it takes a reader at all.
	require.NoError(t, execJobsWithStdin(&buf, strings.NewReader(twoOutcomeMetricEntries),
		"outcome-metrics", "loan-app-1", "--file", "-"))

	require.Len(t, got, 2)
	assert.Equal(t, "2026-02-01T09:15:00Z", got[0]["recordedAt"],
		"recordedAt must arrive exactly as it was written")
	assert.Equal(t, "2026-02-02T11:30:45Z", got[1]["recordedAt"])

	// No key present in the source may be missing from what reached the wire.
	want := decodeFactEntries(t, twoOutcomeMetricEntries)
	for i, entry := range want {
		for key, value := range entry {
			require.Contains(t, got[i], key,
				"entry %d lost the key %q between the pipe and the wire", i, key)
			assert.Equal(t, value, got[i][key], "entry %d key %q", i, key)
		}
	}
	assert.Equal(t, 1, counter.count())
}

// TestOutcomeMetricsHandlesEmpty201 mirrors this phase's highest-value
// assertion, on the second of the two endpoints that share the contract.
//
// POST /v2/api/jobs/{agenticJobId}/outcome/metrics declares a 201 with NO
// content block at all, and internal/api.Client.Do decodes the response body
// whenever its result argument is non-nil. Passing a decode target therefore
// turns every successful append into "failed to decode response: EOF" — a
// permanent write reported to the operator as a failure.
//
// The stub below sets the status and does nothing further: no content-type
// header, no body, not even an empty JSON document. That is the point. A stub
// that emitted `{}` after the 201 would keep the defective form green and this
// test would prove nothing.
func TestOutcomeMetricsHandlesEmpty201(t *testing.T) {
	resetOutcomeMetricsFlags(t)

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, twoOutcomeMetricEntries)
	require.NoError(t, execJobs(&buf,
		"outcome-metrics", "loan-app-1", "--file", path),
		"a 201 carrying no body is the documented success and must not surface as a decode error")

	assert.Equal(t, 1, counter.count())
	assert.Contains(t, buf.String(), "Appended 2 outcome metrics",
		"success must be reported from what was sent — the server returned nothing to report")
}

// TestOutcomeMetricsValidation is the command-level half of D-28-06: the SAME
// validator that refuses these entries for `facts append` refuses them here,
// with the same indexed message, and refuses them BEFORE any request exists.
//
// The load-bearing assertion in every case is the request counter reading
// exactly 0 — an equality assertion, not a zero-value one. require.Error alone
// would pass with the validation gate deleted, because a malformed request
// fails too, and an unrelated error satisfying the same assertion is not
// evidence of a client-side refusal.
func TestOutcomeMetricsValidation(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		wants []string
	}{
		{
			name: "bad provenance at index 1",
			body: `[
			  {"key":"documents_reviewed","value":1,"recordedAt":"2026-02-01T00:00:00Z","provenance":"MEASURED"},
			  {"key":"documents_reviewed","value":2,"recordedAt":"2026-02-01T00:00:00Z","provenance":"MEASUED"}
			]`,
			wants: []string{"entry [1].provenance"},
		},
		{
			name:  "quality_rate above the bound at index 0",
			body:  `[{"key":"quality_rate","value":1.5,"recordedAt":"2026-02-01T00:00:00Z"}]`,
			wants: []string{"entry [0].value"},
		},
		{
			name: "both failures reported together",
			body: `[
			  {"key":"quality_rate","value":1.5,"recordedAt":"2026-02-01T00:00:00Z"},
			  {"key":"documents_reviewed","value":2,"recordedAt":"2026-02-01T00:00:00Z","provenance":"SIGNED_OFF"}
			]`,
			wants: []string{"entry [0].value", "entry [1].provenance"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetOutcomeMetricsFlags(t)

			var counter requestCounter
			srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
				counter.inc()
				w.WriteHeader(http.StatusCreated)
			})

			var buf bytes.Buffer
			cmd.APIClient = stubClient(srv.URL, "team-28")
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			path := writeTempFactsFile(t, tc.body)
			err := execJobs(&buf, "outcome-metrics", "loan-app-1", "--file", path)
			require.Error(t, err)
			for _, want := range tc.wants {
				assert.Contains(t, err.Error(), want,
					"the refusal must name the entry's position in the operator's file")
			}
			assert.Equal(t, 0, counter.count(),
				"a refused entry means no request object was ever built — an appended metric cannot be withdrawn")
		})
	}
}

// TestOutcomeMetricsRejectsBadInput pins the five refusals the shared reader
// gives, on this command as on the facts command.
//
// SIGNED_OFF above and these five here are the two halves of the same claim:
// one reader and one validator serve both entry kinds, so an input refused for
// one is refused identically for the other. Every case names its source — the
// file path — because "failed to parse" without a source is nearly useless
// when a shell pipeline supplied the document.
func TestOutcomeMetricsRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "empty file", body: ``},
		{name: "whitespace only", body: "  \n\t  \n"},
		{name: "bare null", body: `null`},
		{name: "a JSON object rather than an array", body: `{"key":"documents_reviewed","value":1}`},
		{name: "an empty array", body: `[]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetOutcomeMetricsFlags(t)

			var counter requestCounter
			srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
				counter.inc()
				w.WriteHeader(http.StatusCreated)
			})

			var buf bytes.Buffer
			cmd.APIClient = stubClient(srv.URL, "team-28")
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			path := writeTempFactsFile(t, tc.body)
			err := execJobs(&buf, "outcome-metrics", "loan-app-1", "--file", path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), path,
				"the refusal must name the input source the operator supplied")
			assert.Equal(t, 0, counter.count(),
				"a refused document means no request object was ever built")
		})
	}
}

// TestOutcomeMetricsPreservesOrderAndDuplicates pins that the CLI is a courier.
//
// Two entries carrying identical key and recordedAt are two measurements, and
// the endpoint is append-only: collapsing them into one would discard
// something the operator recorded. Order matters for the same reason every
// refusal message names an index — a reordered array makes "entry [2]" point
// at the wrong line of the operator's file.
func TestOutcomeMetricsPreservesOrderAndDuplicates(t *testing.T) {
	resetOutcomeMetricsFlags(t)

	const fourEntriesTwoIdentical = `[
	  {"key":"documents_reviewed","value":1234567.89,"recordedAt":"2026-01-01T00:00:00Z"},
	  {"key":"quality_rate","value":0.5,"recordedAt":"2026-02-01T00:00:00Z"},
	  {"key":"quality_rate","value":0.5,"recordedAt":"2026-02-01T00:00:00Z"},
	  {"key":"documents_reviewed","value":3,"recordedAt":"2026-03-01T00:00:00Z"}
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
		"outcome-metrics", "loan-app-1", "--file", path))

	want := decodeFactEntries(t, fourEntriesTwoIdentical)
	require.Len(t, got, 4, "nothing may be merged, deduplicated or dropped")
	assert.Equal(t, want, got, "the array must reach the wire in source order")
	assert.Equal(t, got[1], got[2], "the duplicate pair must both survive")
	assert.Equal(t, 1234567.89, got[0]["value"],
		"the number must arrive as it was written — no rounding, no re-formatting")
	assert.Equal(t, 1, counter.count(),
		"one POST per invocation regardless of entry count — there is no per-entry loop")
}

// TestOutcomeMetricsDryRun is the guard that stops a preview from performing
// the write it previews.
//
// --dry-run is a ROOT persistent flag, so cobra accepts it on this command
// whether or not anybody reads it, the README promises it previews any
// mutation without executing it, and cmd/schema.go publishes this command's
// `mutating` annotation to automation. An appended metric is permanent and the
// API offers no deletion, so an unread --dry-run is a guard failing open on an
// irreversible write.
//
// The load-bearing assertion is the counter. Neither exit 0 nor the
// "No changes were made." footer can tell a refused request from one that was
// issued and happened to succeed — the renderer prints the footer either way.
//
// The flag is set through setRootFlag rather than declared locally: a local
// flag of the same name would shadow the root's and silently break the
// accessor the command reads.
func TestOutcomeMetricsDryRun(t *testing.T) {
	resetOutcomeMetricsFlags(t)
	setRootFlag(t, "dry-run", "true")

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	path := writeTempFactsFile(t, twoOutcomeMetricEntries)
	require.NoError(t, execJobs(&buf,
		"outcome-metrics", "loan-app-1", "--file", path),
		"a dry run is a question, not an attempt: it exits 0")

	assert.Equal(t, 0, counter.count(),
		"--dry-run must issue no POST — an appended metric is permanent and cannot be withdrawn")

	out := buf.String()
	assert.Contains(t, out, "Dry run: append job outcome metrics")
	assert.Contains(t, out, "/v2/api/jobs/loan-app-1/outcome/metrics",
		"the operator must be told the resolved path")
	assert.Contains(t, out, "2 entries",
		"and the parsed entry count")
	assert.Contains(t, out, "(parsed; provenance and quality_rate checked)",
		"the preview must name the two checks that ran, not claim the entries were validated")
	assert.NotContains(t, out, "parsed and validated",
		"required-ness, shape and declared-ness are NOT checked here — an entry with no key, "+
			"an undeclared metric, or no fields at all passes this path, so a bare \"validated\" "+
			"green-lights a permanent append the server will reject")
}
