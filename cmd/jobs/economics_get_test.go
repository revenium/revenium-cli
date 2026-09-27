package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// economicsSectionsBody is a JobTypeEconomicsResource carrying TWO metrics and
// one dimension with TWO allowed values, so the Metrics and Dimensions sections
// are proven to render every entry rather than only the first.
//
// It lives here rather than in stub_test.go because stub_test.go is shared with
// the sibling plans executing in parallel; the shared economicsResourceBody
// fixture is left untouched.
//
// overheadPerUnit is deliberately 1234567: below 1e6 the scientific-notation
// defect money() exists to prevent does not appear, so a fixture of small round
// numbers would leave the rendering path unpinned.
const economicsSectionsBody = `{
  "jobType": "acme-review",
  "unitMetricKey": "documents-reviewed",
  "unitLabel": "documents reviewed",
  "metrics": [
    {"key":"documents-reviewed","type":"COUNT","direction":"HIGHER_IS_BETTER","aggregation":"SUM","resolution":"PER_JOB"},
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
  "overheadCurrency": "USD",
  "currentBaseline": {
    "version": 3,
    "costPerUnit": 4.25,
    "minutesPerUnit": 18,
    "qualityRate": 0.955,
    "hourlyRate": 85.5,
    "currency": "USD",
    "provenance": "CUSTOMER_DECLARED",
    "declaredBy": "ops@acme.example",
    "evidenceUrl": "https://acme.example/evidence/3",
    "effectiveFrom": "2026-01-01T00:00:00Z",
    "created": "2026-01-02T09:15:00Z"
  }
}`

// economicsEmptyBody is a contract that declares nothing beyond its scalars:
// an EMPTY metrics array, an EMPTY dimensions array, and an explicit JSON null
// currentBaseline. Each of the three is a different fact, and D-27-09 requires
// each to read as a sentence rather than as a blank.
const economicsEmptyBody = `{
  "jobType": "acme-review",
  "unitMetricKey": "documents-reviewed",
  "unitLabel": "documents reviewed",
  "metrics": [],
  "dimensions": [],
  "monetization": null,
  "overheadPerUnit": 12.5,
  "overheadCurrency": "USD",
  "currentBaseline": null
}`

// economicsLargeNumbersBody crosses the scientific-notation threshold in BOTH
// numeric paths — the contract's overheadPerUnit and the baseline's
// costPerUnit.
//
// The values are load-bearing, not incidental. encoding/json decodes every JSON
// number into a float64, and the unformatted no-verb print variant str() uses
// emits scientific notation at or above 1e6. A fixture of 10 or 1.5 would keep
// a broken renderer green and prove nothing — precisely the shape Phase 26
// found five times.
const economicsLargeNumbersBody = `{
  "jobType": "acme-review",
  "unitMetricKey": "documents-reviewed",
  "unitLabel": "documents reviewed",
  "metrics": [
    {"key":"documents-reviewed","type":"COUNT","direction":"HIGHER_IS_BETTER","aggregation":"SUM","resolution":"PER_JOB"}
  ],
  "dimensions": [
    {"key":"region","allowedValues":["us-east"]}
  ],
  "monetization": {
    "metricKey":"documents-reviewed",
    "valuePerUnit":12.5,
    "currency":"USD",
    "category":"COST_AVOIDED",
    "basis":"REALIZED"
  },
  "overheadPerUnit": 12345678901,
  "overheadCurrency": "USD",
  "currentBaseline": {
    "version": 3,
    "costPerUnit": 1234567,
    "minutesPerUnit": 18,
    "qualityRate": 0.955,
    "hourlyRate": 85.5,
    "currency": "USD",
    "provenance": "MEASURED",
    "declaredBy": "ops@acme.example",
    "evidenceUrl": "https://acme.example/evidence/3",
    "effectiveFrom": "2026-01-01T00:00:00Z",
    "created": "2026-01-02T09:15:00Z"
  }
}`

// economicsNullNumericsBody carries EXACTLY TWO explicit JSON nulls in numeric
// positions — overheadPerUnit and the baseline's costPerUnit — and declares
// every other numeric field.
//
// The nulls must be explicit rather than omitted keys: an omitted key and a
// null key take different paths through the decoder even though both must
// render the same, and output.FloatVal returns 0 for both, so without the
// presence branch inside money() an undeclared cost renders as a confident
// $0.00.
const economicsNullNumericsBody = `{
  "jobType": "acme-review",
  "unitMetricKey": "documents-reviewed",
  "unitLabel": "documents reviewed",
  "metrics": [
    {"key":"documents-reviewed","type":"COUNT","direction":"HIGHER_IS_BETTER","aggregation":"SUM","resolution":"PER_JOB"}
  ],
  "dimensions": [
    {"key":"region","allowedValues":["us-east"]}
  ],
  "monetization": {
    "metricKey":"documents-reviewed",
    "valuePerUnit":12.5,
    "currency":"USD",
    "category":"COST_AVOIDED",
    "basis":"REALIZED"
  },
  "overheadPerUnit": null,
  "overheadCurrency": "USD",
  "currentBaseline": {
    "version": 3,
    "costPerUnit": null,
    "minutesPerUnit": 18,
    "qualityRate": 0.955,
    "hourlyRate": 85.5,
    "currency": "USD",
    "provenance": "MEASURED",
    "declaredBy": "ops@acme.example",
    "evidenceUrl": "https://acme.example/evidence/3",
    "effectiveFrom": "2026-01-01T00:00:00Z",
    "created": "2026-01-02T09:15:00Z"
  }
}`

// serveEconomics starts a stub returning body for the economics endpoint and
// points cmd.APIClient at it. The teamId assertion is kept on every path: the
// endpoint requires it and requireTeam() only proves the local guard, not that
// the value reaches the wire.
func serveEconomics(t *testing.T, body string) {
	t.Helper()
	srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v2/api/jobs/types/acme-review/economics", r.URL.Path)
		assert.Equal(t, "team-27", r.URL.Query().Get("teamId"),
			"teamId is required by the endpoint and must reach the wire")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	})
	cmd.APIClient = stubClient(srv.URL, "team-27")
}

// TestEconomicsGet is the end-to-end happy path for
// `revenium jobs types economics get <type>`: registration → arg validation →
// team guard → path escaping → request → four-section render.
//
// It runs through the package-level Cmd, so a registration regression turns
// this test red rather than shipping green.
func TestEconomicsGet(t *testing.T) {
	serveEconomics(t, economicsSectionsBody)

	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	err := execJobs(&buf, "types", "economics", "get", "acme-review")
	require.NoError(t, err)

	out := buf.String()

	// All four D-27-09 section headings, in the fixed order.
	assert.Contains(t, out, "Contract")
	assert.Contains(t, out, "Metrics")
	assert.Contains(t, out, "Dimensions")
	assert.Contains(t, out, "Current Baseline")

	// Contract section, including the flattened monetization object.
	assert.Contains(t, out, "Unit Metric Key")
	assert.Contains(t, out, "documents reviewed")
	assert.Contains(t, out, "COST_AVOIDED")
	assert.Contains(t, out, "REALIZED")
	// The overhead cell is 1234567 in the fixture; str() would render it as
	// "1.234567e+06". This assertion is what keeps the render on money().
	assert.Contains(t, out, "$1,234,567.00")

	// Metrics section: BOTH metric keys, so a render that emitted only the
	// first entry is red rather than green.
	assert.Contains(t, out, "documents-reviewed")
	assert.Contains(t, out, "review-accuracy")
	assert.Contains(t, out, "PERCENT")

	// Dimensions section: BOTH allowed values, joined into one cell.
	assert.Contains(t, out, "region")
	assert.Contains(t, out, "us-east")
	assert.Contains(t, out, "eu-west")

	// Current Baseline section: the numeric cells through num()/money().
	assert.Contains(t, out, "$4.25")
	assert.Contains(t, out, "0.9550")
	assert.Contains(t, out, "CUSTOMER_DECLARED")
}

// TestEconomicsGetEmptyStates pins D-27-09's "empty states are deliberate, not
// blank". An empty section and an absent section are indistinguishable to an
// operator, and only one of them is a fact about the contract.
func TestEconomicsGetEmptyStates(t *testing.T) {
	serveEconomics(t, economicsEmptyBody)

	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	err := execJobs(&buf, "types", "economics", "get", "acme-review")
	require.NoError(t, err)

	out := buf.String()

	// None of the three sections may be silently absent — the heading is what
	// tells the operator the section was considered at all.
	assert.Contains(t, out, "Metrics")
	assert.Contains(t, out, "Dimensions")
	assert.Contains(t, out, "Current Baseline")

	// Three DISTINCT sentences, one naming each undeclared thing.
	assert.Contains(t, out, "No metrics are declared for this job type.")
	assert.Contains(t, out, "No dimensions are declared for this job type.")
	assert.Contains(t, out, "No baseline version has been declared yet.")
	// The baseline sentence names the command that declares one.
	assert.Contains(t, out, "jobs types baselines append")

	// A null monetization is one fact, not five blank rows.
	assert.Contains(t, out, "not declared")
}

// TestEconomicsGetJSON proves the --json passthrough is VERBATIM: the emitted
// document is the server's JobTypeEconomicsResource, not a projection of the
// table rows.
//
// It asserts on DECODED map keys rather than on substrings, following the
// JSON-mode test at the end of cmd/jobs/types_test.go. A substring match would
// pass against a rendered table that merely contained the word "jobType", and
// would not catch a four-section render that emitted four JSON documents.
func TestEconomicsGetJSON(t *testing.T) {
	serveEconomics(t, economicsSectionsBody)

	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	err := execJobs(&buf, "types", "economics", "get", "acme-review")
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result),
		"stdout must be exactly one JSON document; four Render calls would emit four")

	// The two read-only keys and correction C1's seventh request property.
	assert.Equal(t, "acme-review", result["jobType"])
	assert.Equal(t, "USD", result["overheadCurrency"])
	require.Contains(t, result, "currentBaseline")

	baseline, ok := result["currentBaseline"].(map[string]interface{})
	require.True(t, ok, "currentBaseline must survive as a nested object")
	assert.Equal(t, float64(3), baseline["version"])
	assert.Equal(t, 4.25, baseline["costPerUnit"],
		"--json is the full-precision channel; the table's 2-decimal cell is never the value of record")

	// Every metric entry, not just the ones the table happens to show.
	metrics, ok := result["metrics"].([]interface{})
	require.True(t, ok)
	require.Len(t, metrics, 2)
	first, ok := metrics[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "documents-reviewed", first["key"])

	// The nested monetization object passes through unflattened — the table
	// flattens it, the document must not.
	monetization, ok := result["monetization"].(map[string]interface{})
	require.True(t, ok, "monetization must not be flattened into the document")
	assert.Equal(t, 12.5, monetization["valuePerUnit"])
}

// TestEconomicsGetLargeNumber pins JOBS-14's precision trap: a cost an operator
// acts on must never be rendered in scientific notation.
//
// encoding/json decodes every JSON number into a float64, and the unformatted
// no-verb print variant str() uses emits 1.234567e+06 for one at or above 1e6.
// This test is red under the exact mutation it exists to catch — swapping a
// money() call back to str().
//
// --json remains the full-precision channel; these table cells are a
// two-decimal presentation and are never the value of record.
func TestEconomicsGetLargeNumber(t *testing.T) {
	serveEconomics(t, economicsLargeNumbersBody)

	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	err := execJobs(&buf, "types", "economics", "get", "acme-review")
	require.NoError(t, err)

	out := buf.String()

	// Positive: the exact comma-grouped, two-decimal strings.
	assert.Contains(t, out, "$1,234,567.00",
		"the baseline's costPerUnit must render comma-grouped, not in scientific notation")
	assert.Contains(t, out, "$12,345,678,901.00",
		"the contract's overheadPerUnit must render comma-grouped, not in scientific notation")

	// Negative: name the regression rather than merely detecting it.
	assert.NotContains(t, out, "e+06",
		"1234567 rendered as 1.234567e+06 — a numeric cell escaped money()/num()")
	assert.NotContains(t, out, "e+10",
		"12345678901 rendered as 1.2345678901e+10 — a numeric cell escaped money()/num()")
}

// TestEconomicsGetNullNumerics pins the other half of JOBS-14's precision
// trap: "not declared" and "declared as zero" are different facts about a
// contract and must be visibly different in the output.
//
// output.FloatVal returns 0 for a JSON null, so without the presence branch
// inside money() an undeclared cost renders as a confident $0.00 — a
// wrong-but-plausible number an operator would act on (T-27-08).
func TestEconomicsGetNullNumerics(t *testing.T) {
	serveEconomics(t, economicsNullNumericsBody)

	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	err := execJobs(&buf, "types", "economics", "get", "acme-review")
	require.NoError(t, err)

	out := buf.String()

	// The fixture nulls exactly two numeric fields and declares every other
	// one, so exactly two cells may carry the undeclared placeholder. Counting
	// is what keeps this from passing on an em dash that leaked somewhere else.
	assert.Equal(t, 2, strings.Count(out, undeclared),
		"exactly the two JSON-null numerics — overheadPerUnit and costPerUnit — render as %q", undeclared)

	// Negative: the confident zero is the defect, named.
	assert.NotContains(t, out, "$0.00",
		"a JSON-null numeric rendered as $0.00 — the presence branch in money() is gone")

	// The declared numerics still render, so the test cannot pass by rendering
	// nothing at all.
	assert.Contains(t, out, "$85.50", "a declared hourlyRate must still render")
	assert.Contains(t, out, "0.9550", "a declared qualityRate must still render")
}
