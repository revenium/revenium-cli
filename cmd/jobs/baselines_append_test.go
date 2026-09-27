package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// appendFlagNames are the nine BaselineRequest flags, in schema order.
var appendFlagNames = []string{
	"cost-per-unit", "minutes-per-unit", "quality-rate", "hourly-rate",
	"currency", "provenance", "declared-by", "evidence-url", "effective-from",
}

// resetAppendFlags restores the append command's flags to their unset state.
//
// initBaselines() constructs newBaselinesAppendCmd() exactly once, at package
// initialization, so every test in this binary drives the SAME command object
// with the same flag values and the same Changed bits. Without this reset a
// test that omits --provenance would still observe Changed("provenance") ==
// true from a test that ran earlier, and
// TestBaselinesAppendProvenanceOptional would pass or fail according to source
// order rather than according to behaviour. Executing through the real
// registration path is worth this bookkeeping — it is what makes these tests
// exercise the command an operator actually reaches.
func resetAppendFlags(t *testing.T) {
	t.Helper()
	c, _, err := Cmd.Find([]string{"types", "baselines", "append"})
	require.NoError(t, err)
	for _, name := range appendFlagNames {
		fl := c.Flags().Lookup(name)
		require.NotNil(t, fl, "flag --%s must be registered", name)
		require.NoError(t, fl.Value.Set(fl.DefValue))
		fl.Changed = false
	}
}

// TestBaselinesAppend is JOBS-14's happy path: all nine flags supplied, and
// the stub asserts each one arrived on the wire with the value given.
func TestBaselinesAppend(t *testing.T) {
	resetAppendFlags(t)

	var got map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/jobs/types/acme-review/baselines", r.URL.Path)
		assert.Equal(t, "team-27", r.URL.Query().Get("teamId"),
			"teamId is required by the endpoint and must reach the wire")
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":7}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--cost-per-unit", "4.25",
		"--minutes-per-unit", "18",
		"--quality-rate", "0.955",
		"--hourly-rate", "85.5",
		"--currency", "USD",
		"--provenance", "MEASURED",
		"--declared-by", "ops@acme.example",
		"--evidence-url", "https://acme.example/evidence/7",
		"--effective-from", "2026-01-07T00:00:00Z",
	))

	// All nine BaselineRequest fields, by their schema key names — the flag
	// names are kebab-case and the wire keys are camelCase, so a typo in the
	// mapping is invisible without asserting the decoded body.
	require.NotNil(t, got)
	assert.Equal(t, 4.25, got["costPerUnit"])
	assert.Equal(t, 18.0, got["minutesPerUnit"])
	assert.Equal(t, 0.955, got["qualityRate"])
	assert.Equal(t, 85.5, got["hourlyRate"])
	assert.Equal(t, "USD", got["currency"])
	assert.Equal(t, "MEASURED", got["provenance"])
	assert.Equal(t, "ops@acme.example", got["declaredBy"])
	assert.Equal(t, "https://acme.example/evidence/7", got["evidenceUrl"])
	assert.Equal(t, "2026-01-07T00:00:00Z", got["effectiveFrom"])
	assert.Len(t, got, 9, "exactly the nine supplied fields, no invented keys")
}

// TestBaselinesAppend201 pins that the spec's 201 is an ordinary success.
// internal/api.Client.Do decodes any status below 400, so a handler that
// treated 2xx-but-not-200 as an error would be a regression here.
func TestBaselinesAppend201(t *testing.T) {
	resetAppendFlags(t)

	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":7,"costPerUnit":4.25,"currency":"USD",
			"provenance":"MEASURED","declaredBy":"ops@acme.example",
			"effectiveFrom":"2026-01-07T00:00:00Z"}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--cost-per-unit", "4.25",
		"--effective-from", "2026-01-07T00:00:00Z",
	), "a 201 is the documented success status and must not surface as an error")

	rows := tableRows(t, buf.String())
	require.Len(t, rows, 2, "one header row plus the appended version")
	assert.Equal(t, "7", rows[1][0], "the appended version must be shown back to the operator")
}

// TestBaselinesAppendProvenanceOptional proves the two documented server-side
// defaults are left to the server. Both fields are optional and nullable, so a
// client-side requirement would refuse input the API accepts (Pitfall 3).
//
// Mutation pin: removing the flag-was-supplied guard around the provenance
// check must turn this test red.
func TestBaselinesAppendProvenanceOptional(t *testing.T) {
	resetAppendFlags(t)

	var got map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":8}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--cost-per-unit", "4.25",
		"--effective-from", "2026-01-07T00:00:00Z",
	), "omitting an optional field must not be an error")

	require.NotNil(t, got)
	assert.NotContains(t, got, "provenance",
		`omitted means absent: the server documents "Defaults to CUSTOMER_DECLARED when omitted"`)
	assert.NotContains(t, got, "declaredBy",
		`omitted means absent: the server documents "Defaults to the calling principal when omitted"`)
}

// TestBaselinesAppendRejectsFactProvenance pins T-27-11. SELF_REPORTED belongs
// to Phase 28's four-value outcome-fact enum, not to a baseline, and the
// server's own 422 is a generic "Semantic validation failed" — so catching it
// here is what tells the operator what was actually wrong.
func TestBaselinesAppendRejectsFactProvenance(t *testing.T) {
	resetAppendFlags(t)

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":9}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	err := execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--provenance", "SELF_REPORTED",
		"--effective-from", "2026-01-07T00:00:00Z",
	)
	require.Error(t, err)

	// The error must name every legal value — an operator who mistypes should
	// not have to open the API spec to learn what was expected.
	assert.Contains(t, err.Error(), "CUSTOMER_DECLARED")
	assert.Contains(t, err.Error(), "MEASURED")
	assert.Contains(t, err.Error(), "SIGNED_OFF")

	// The load-bearing assertion. require.Error alone cannot tell a
	// client-side rejection from a request that reached the server and failed
	// there; the counter is what proves the check ran BEFORE the wire.
	assert.Equal(t, 0, counter.count(),
		"an invalid --provenance must be refused before any request is issued")
}

// TestBaselinesAppendAcceptsOutOfRangeRate pins D-27-11's refusal to invent a
// constraint. BaselineRequest declares no 0-1 bound on qualityRate, so the
// value goes to the wire unchanged and the API is the authority.
func TestBaselinesAppendAcceptsOutOfRangeRate(t *testing.T) {
	resetAppendFlags(t)

	var got map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":10}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--quality-rate", "1.5",
		"--effective-from", "2026-01-07T00:00:00Z",
	), "no client-side range check may refuse a value the schema permits")

	require.NotNil(t, got)
	assert.Equal(t, 1.5, got["qualityRate"], "the value must reach the wire unchanged")
}

// TestBaselinesAppendPreservesPrior is JOBS-14 SC4 made observable: the
// immutability the spec promises is demonstrated through the CLI rather than
// assumed. The stub accumulates what it is posted, so list-append-list shows
// the prior version still present and still carrying its original values.
func TestBaselinesAppendPreservesPrior(t *testing.T) {
	resetAppendFlags(t)

	var mu sync.Mutex
	stored := []map[string]interface{}{{
		"version":       1,
		"costPerUnit":   5.0,
		"currency":      "USD",
		"provenance":    "CUSTOMER_DECLARED",
		"declaredBy":    "v1@acme.example",
		"effectiveFrom": "2026-01-01T00:00:00Z",
	}}

	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			assert.NoError(t, json.NewEncoder(w).Encode(stored))
		case http.MethodPost:
			var body map[string]interface{}
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
				return
			}
			// The server derives version as the 1-based position in
			// declaration order; it never overwrites an existing entry.
			body["version"] = len(stored) + 1
			stored = append(stored, body)
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(body))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	cmd.APIClient = stubClient(srv.URL, "team-27")

	// Before.
	var before bytes.Buffer
	cmd.Output = output.NewWithWriter(&before, &before, false, false)
	require.NoError(t, execJobs(&before, "types", "baselines", "list", "acme-review"))
	beforeRows := tableRows(t, before.String())
	require.Len(t, beforeRows, 2)
	assert.Equal(t, "1", beforeRows[1][0])
	assert.Equal(t, "$5.00", beforeRows[1][1])

	// Append.
	var appended bytes.Buffer
	cmd.Output = output.NewWithWriter(&appended, &appended, false, false)
	require.NoError(t, execJobs(&appended,
		"types", "baselines", "append", "acme-review",
		"--cost-per-unit", "3.25",
		"--currency", "USD",
		"--declared-by", "v2@acme.example",
		"--effective-from", "2026-02-01T00:00:00Z",
	))

	// After.
	var after bytes.Buffer
	cmd.Output = output.NewWithWriter(&after, &after, false, false)
	require.NoError(t, execJobs(&after, "types", "baselines", "list", "acme-review"))
	afterRows := tableRows(t, after.String())
	require.Len(t, afterRows, 3, "both versions must be listed — appending removes nothing")

	declaredByCol := -1
	for i, h := range afterRows[0] {
		if h == "Declared By" {
			declaredByCol = i
		}
	}
	require.NotEqual(t, -1, declaredByCol)

	// Newest first: the appended version is row 1.
	assert.Equal(t, "2", afterRows[1][0])
	assert.Equal(t, "$3.25", afterRows[1][1])
	assert.Equal(t, "v2@acme.example", afterRows[1][declaredByCol])

	// The whole point: version 1 is still readable and still says what it
	// said before the append.
	assert.Equal(t, "1", afterRows[2][0])
	assert.Equal(t, "$5.00", afterRows[2][1],
		"the prior version must keep its original values — baselines are immutable")
	assert.Equal(t, "v1@acme.example", afterRows[2][declaredByCol])
}

// TestBaselinesAppendDryRunIssuesNoPost pins the repo-wide --dry-run contract
// on the one write in this phase that cannot be undone.
//
// --dry-run is a ROOT persistent flag (cmd/root.go), so cobra accepts it on
// this command whether or not anybody reads it, and README promises it previews
// "any mutation ... without executing it". A baseline version is permanent —
// the API offers no amend and no delete — so a --dry-run that still POSTs
// creates a permanent version and reports success.
//
// The load-bearing assertion is the counter. require.NoError alone cannot tell
// a refused request from one that was issued and happened to succeed, and the
// rendered success table is itself produced from the stub's response, so
// asserting on output shape without the counter would pass under the very
// mutation this test exists to catch.
func TestBaselinesAppendDryRunIssuesNoPost(t *testing.T) {
	resetAppendFlags(t)
	setRootFlag(t, "dry-run", "true")

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":42,"declaredBy":"never-appended@acme.example"}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--cost-per-unit", "4.25",
		"--provenance", "MEASURED",
		"--effective-from", "2026-01-07T00:00:00Z",
	), "a dry run is a question, not an attempt: it exits 0")

	assert.Equal(t, 0, counter.count(),
		"--dry-run must issue no POST — a baseline version is permanent and cannot be withdrawn")

	out := buf.String()
	assert.Contains(t, out, "Dry run: append job type baseline",
		"the operator must be told which mutation was previewed")
	assert.Contains(t, out, "/v2/api/jobs/types/acme-review/baselines")
	assert.Contains(t, out, "No changes were made.")
	assert.NotContains(t, out, "never-appended@acme.example",
		"nothing from the server may be rendered — no request was issued")
}

// TestBaselinesAppendRequiresEffectiveFrom pins G-1 / D11.
//
// The dev host rejects an omitted effectiveFrom with HTTP 400 carrying the
// bare body `effectiveFrom` — confirmed by a live call on 2026-09-05, which is
// what finally answered a question open since plan 27-03. BaselineRequest
// declares no `required` key, so the field's required-ness was not inferable
// from the schema and no stub could settle it.
//
// The guard exists so the operator gets a message naming the flag rather than
// a relayed server error, which is the standard requireTeam() sets for
// --team-id under SC5. Every other flag on this command is gated on Changed();
// this one deliberately is not, because absence is the condition it refuses.
func TestBaselinesAppendRequiresEffectiveFrom(t *testing.T) {
	resetAppendFlags(t)

	var counter requestCounter
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":10}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	err := execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--cost-per-unit", "4.25",
	)
	require.Error(t, err)

	// Name the flag, not the JSON field: the operator types `--effective-from`,
	// and the server's own `effectiveFrom` is exactly the unhelpful text this
	// guard exists to replace.
	assert.Contains(t, err.Error(), "--effective-from",
		"the error must name the flag the operator has to add")

	// The load-bearing assertion. require.Error alone cannot distinguish a
	// client-side refusal from a request that reached the server and was
	// rejected there — which is precisely the behaviour being fixed, and would
	// keep this test green under the mutation it exists to catch.
	assert.Equal(t, 0, counter.count(),
		"a missing --effective-from must be refused before any request is issued")
}

// TestBaselinesAppendAcceptsSuppliedEffectiveFrom is the companion that stops
// the G-1 guard from being satisfiable by refusing everything.
func TestBaselinesAppendAcceptsSuppliedEffectiveFrom(t *testing.T) {
	resetAppendFlags(t)

	var counter requestCounter
	var got map[string]interface{}
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		counter.inc()
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"version":11}`)
	})

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf,
		"types", "baselines", "append", "acme-review",
		"--cost-per-unit", "4.25",
		"--effective-from", "2026-01-07T00:00:00Z",
	), "a supplied --effective-from must proceed")

	assert.Equal(t, 1, counter.count(), "the request must reach the server")
	assert.Equal(t, "2026-01-07T00:00:00Z", got["effectiveFrom"])
}
