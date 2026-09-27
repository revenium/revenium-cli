package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every test in this file constructs cmd.APIClient FRESH with api.NewClient.
// D-29-05: the command's host swap is process-wide with no restore, so a client
// shared across two command executions would carry the analytics host and
// bearer auth into the second one (RESEARCH Pitfall 6).

// TestROISummaryHitsAnalyticsHostWithBearerAndNoTeamParams is JOBS-18 SC2's
// whole evidence, and it is deliberately NOT shaped like
// cmd/metrics/dimensions_test.go:18-46 — that test hand-sets AnalyticsBaseURL
// and UseBearerAuth before executing, so deleting the production swap leaves it
// green. Four properties here are load-bearing:
//
//  1. A DECOY platform server whose handler fails the test. cmd.APIClient.BaseURL
//     points at the decoy and AnalyticsBaseURL at the real stub, so a missing
//     BaseURL swap sends the request to the decoy — the decoy IS the assertion.
//  2. TeamID and TenantID are NON-EMPTY. An empty-teamId assertion against a
//     client that never had a team passes vacuously.
//  3. UseBearerAuth is deliberately NOT set by the test. The command's own
//     PersistentPreRunE must set it, or this test proves nothing.
//  4. The Authorization / x-api-key pair is asserted both ways round.
//
// Executing the command unattached is safe and still exercises the swap: cobra
// runs the resolved command's own PersistentPreRunE, and with c.Root() == c the
// self-delegation guard skips the root call while the swap block still runs.
func TestROISummaryHitsAnalyticsHostWithBearerAndNoTeamParams(t *testing.T) {
	decoy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("platform decoy was hit: %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusTeapot)
	}))
	defer decoy.Close()

	var gotAuth, gotAPIKey, gotTeam, gotTenant, gotPath, gotMethod string
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotAPIKey = r.Header.Get("Authorization"), r.Header.Get("x-api-key")
		gotTeam, gotTenant = r.URL.Query().Get("teamId"), r.URL.Query().Get("tenantId")
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"summary":{"totalJobs":3}}`)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(decoy.URL, "test-key", "test-team", "test-tenant", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	require.NoError(t, c.Execute())

	assert.Equal(t, "GET", gotMethod)
	assert.Equal(t, "/api/v2/analytics/jobs/roi-summary", gotPath)
	assert.Equal(t, "Bearer test-key", gotAuth)
	assert.Empty(t, gotAPIKey, "the analytics host must not receive the platform x-api-key header")
	assert.Empty(t, gotTeam, "bearer auth suppresses teamId injection (internal/api/client.go:74)")
	assert.Empty(t, gotTenant, "bearer auth suppresses tenantId injection (internal/api/client.go:74)")
}

// TestROISummaryTableRendersBothSections is JOBS-18 SC1's table-path evidence:
// two labelled sections, the Summary key-value block first, then the By Job
// Type table.
//
// It asserts CONTENT, not row order. The byJobType ordering guarantee arrives
// in 29-02 and gets its own test there.
func TestROISummaryTableRendersBothSections(t *testing.T) {
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/analytics/jobs/roi-summary", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id":"roi-summary-1",
			"resourceType":"jobTypeROISummary",
			"label":"Job type ROI summary",
			"period":{"start":"2025-09-01","end":"2025-09-30"},
			"byJobType":[
				{"jobType":"loan-processing","totalJobs":120,"totalCost":450.5,"tokenCost":300,
				 "externalToolCost":100,"humanCost":50.5,"conversions":42,"deflections":9,
				 "totalValue":9000,"averageValue":75,"costPerConversion":10.73,
				 "costPerOutcome":8.83,"roi":1897.78,"successRate":91.25},
				{"jobType":"claims-triage","totalJobs":80,"totalCost":220,"tokenCost":180,
				 "externalToolCost":30,"humanCost":10,"conversions":25,"deflections":4,
				 "totalValue":4000,"averageValue":50,"costPerConversion":8.8,
				 "costPerOutcome":7.59,"roi":1718.18,"successRate":88.5}
			],
			"summary":{"totalJobTypes":2,"totalJobs":200,"totalCost":670.5,
				"totalValue":13000,"overallROI":1839.19},
			"_links":{"self":{"href":"/api/v2/analytics/jobs/roi-summary"}}
		}`)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "test-team", "test-tenant", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	require.NoError(t, c.Execute())

	out := buf.String()
	// Both section headings.
	assert.Contains(t, out, "Summary")
	assert.Contains(t, out, "By Job Type")
	// Summary section: labels and a formatted money cell from it.
	assert.Contains(t, out, "Total Job Types")
	assert.Contains(t, out, "Overall ROI")
	assert.Contains(t, out, "2025-09-01")
	assert.Contains(t, out, "2025-09-30")
	assert.Contains(t, out, "$670.50")
	assert.Contains(t, out, "$13,000.00")
	// By Job Type section: both job-type names and a formatted money cell from it.
	assert.Contains(t, out, "loan-processing")
	assert.Contains(t, out, "claims-triage")
	assert.Contains(t, out, "$450.50")
	assert.Contains(t, out, "$220.00")
	// The rate cells are the SERVER's values, rendered bare — no suffix, no
	// multiplication (D-29-10). A client-side recomputation would not produce
	// these numbers.
	assert.Contains(t, out, "1897.78")
	assert.Contains(t, out, "91.25")
	assert.Contains(t, out, "1839.19")
}

// TestROISummaryWorksWithNoTeamConfigured is D-29-03's direct proof: the
// analytics spec declares no team parameter, so this command must run to
// success with no team resolved at all. requireTeam() refuses when the client's
// TeamID is empty, so this test goes red the moment someone "fixes" this
// command to call the package's team guard.
func TestROISummaryWorksWithNoTeamConfigured(t *testing.T) {
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"period":{"start":"2025-09-01","end":"2025-09-30"},
			"byJobType":[],"summary":{"totalJobTypes":0,"totalJobs":0,
			"totalCost":0,"totalValue":0,"overallROI":0}}`)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})

	require.NoError(t, c.Execute(), "no team is resolved and none is required (D-29-03)")
	assert.Contains(t, buf.String(), "By Job Type")
}

// TestROISummaryQueryParamsTranslateToOASNames pins the CLI-kebab-to-OAS-camel
// translation for all four declared query parameters, and the encoding that
// keeps an operator-supplied value from becoming a second query key.
//
// The injection half is the load-bearing one: a --job-type carrying & and =
// reaches a DIFFERENT host with a bearer token on it (T-29-04), so it must
// arrive as one jobType value with that exact content. TeamID and TenantID are
// non-empty here so an "injected teamId" assertion is not vacuous.
func TestROISummaryQueryParamsTranslateToOASNames(t *testing.T) {
	var got url.Values
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"summary":{"totalJobs":0},"byJobType":[]}`)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "test-team", "test-tenant", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--from", "2025-09-01T00:00:00Z",
		"--to", "2025-09-30T23:59:59Z",
		"--job-type", "loan-processing",
		"--metric-type", "avg",
	})
	require.NoError(t, c.Execute())

	assert.Equal(t, "2025-09-01T00:00:00Z", got.Get("startDate"), "--from maps to startDate")
	assert.Equal(t, "2025-09-30T23:59:59Z", got.Get("endDate"), "--to maps to endDate")
	assert.Equal(t, "loan-processing", got.Get("jobType"), "--job-type maps to jobType")
	assert.Equal(t, "avg", got.Get("metricType"), "--metric-type maps to metricType")
	assert.Len(t, got, 4, "exactly the four declared parameters, and no others")

	// Second execution: the same stub, a value carrying query metacharacters.
	got = nil
	var buf2 bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "test-team", "test-tenant", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf2, &buf2, false, false)

	c2 := newROISummaryCmd()
	c2.SetOut(&buf2)
	c2.SetArgs([]string{"--job-type", "a&teamId=other"})
	require.NoError(t, c2.Execute())

	assert.Equal(t, "a&teamId=other", got.Get("jobType"), "the value arrives whole, percent-encoded on the wire")
	assert.Len(t, got, 1, "an injected & = must not create a second query key")
	assert.Empty(t, got.Get("teamId"), "no teamId key may be conjured by an operator-supplied value")
}

// TestROISummaryOmitsUnsetFlags is the changed-bit gate's whole evidence.
//
// It asserts KEY ABSENCE, not empty-string equality: url.Values.Get returns ""
// for an absent key AND for a key present with an empty value, so an
// equality-to-"" assertion would pass whether the gate exists or not. The
// difference between the two is the entire reason the gate is there — an empty
// value sent for an unset flag is what opens an actionable field row on an
// endpoint the audit reports as covered (D-29-11).
func TestROISummaryOmitsUnsetFlags(t *testing.T) {
	var got url.Values
	var gotRawQuery, gotRequestURI string
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, gotRawQuery, gotRequestURI = r.URL.Query(), r.URL.RawQuery, r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"summary":{"totalJobs":0},"byJobType":[]}`)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--from", "2025-09-01"})
	require.NoError(t, c.Execute())

	// Passed through unchanged: no Z appended, no reformatting (RESEARCH D3).
	assert.Equal(t, "2025-09-01", got.Get("startDate"))
	assert.Equal(t, "startDate=2025-09-01", gotRawQuery)
	_, hasEnd := got["endDate"]
	_, hasJobType := got["jobType"]
	_, hasMetricType := got["metricType"]
	assert.False(t, hasEnd, "endDate must be absent from the query, not sent empty")
	assert.False(t, hasJobType, "jobType must be absent from the query, not sent empty")
	assert.False(t, hasMetricType, "metricType must be absent from the query, not sent empty")

	// Second execution: no flags at all produces no query string whatsoever.
	gotRequestURI = ""
	var buf2 bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf2, &buf2, false, false)

	c2 := newROISummaryCmd()
	c2.SetOut(&buf2)
	c2.SetArgs([]string{})
	require.NoError(t, c2.Execute())

	assert.NotContains(t, gotRequestURI, "?", "with no flags the request URI carries no query string at all")
}

// TestROISummaryRejectsMetricTypeOutsideEnum copies the shape of
// cmd/metrics/dimensions_test.go:58-81 — TestDimensionsUnknownNameRejectedBeforeHTTP.
//
// The zero-calls assertion is the load-bearing half. A bare require.Error would
// still pass if the refusal happened AFTER the request was issued, which is
// exactly the property the pre-run placement exists to guarantee: cobra runs
// PersistentPreRunE, then PreRunE, then RunE, so a refusal in the middle slot
// happens before any path or request object is built.
func TestROISummaryRejectsMetricTypeOutsideEnum(t *testing.T) {
	called := false
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--metric-type", "bogus"})
	err := c.Execute()

	require.Error(t, err)
	for _, v := range validMetricTypes {
		assert.Contains(t, err.Error(), v, "the refusal must name every legal value, not a subset")
	}
	assert.False(t, called, "no HTTP call should be made for an invalid metric type")

	// An explicitly-empty value is CHANGED but is not a member of the enum, so
	// the same check must refuse it rather than silently sending metricType=.
	called = false
	var buf2 bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf2, &buf2, false, false)

	c2 := newROISummaryCmd()
	c2.SetOut(&buf2)
	c2.SetErr(&buf2)
	c2.SetArgs([]string{"--metric-type", ""})

	require.Error(t, c2.Execute(), "an empty string is not a member of the enum")
	assert.False(t, called, "no HTTP call should be made for an empty metric type either")
}

// TestROISummaryAcceptsEveryLegalMetricType turns "the list is right" from a
// claim into an assertion: every one of the ten declared values is accepted and
// reaches the wire unchanged under the OAS name.
func TestROISummaryAcceptsEveryLegalMetricType(t *testing.T) {
	require.Len(t, validMetricTypes, 10, "the operation declares exactly ten metric types")

	var got url.Values
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"summary":{"totalJobs":0},"byJobType":[]}`)
	}))
	defer analytics.Close()

	for _, v := range validMetricTypes {
		got = nil
		var buf bytes.Buffer
		cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
		cmd.APIClient.AnalyticsBaseURL = analytics.URL
		cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

		c := newROISummaryCmd()
		c.SetOut(&buf)
		c.SetErr(&buf)
		c.SetArgs([]string{"--metric-type", v})

		require.NoErrorf(t, c.Execute(), "%q is a declared metric type and must be accepted", v)
		assert.Equalf(t, v, got.Get("metricType"), "%q must reach the wire unchanged", v)
	}
}

// roiSummaryUnsortedFixture is deliberately served in a NON-alphabetical order,
// so a test that merely asserts each job type is present would pass on any
// order at all. All fourteen byJobType fields are declared on every item so the
// JSON-verbatim assertion has something to lose.
const roiSummaryUnsortedFixture = `{
	"id":"roi-summary-1",
	"period":{"start":"2025-09-01","end":"2025-09-30"},
	"byJobType":[
		{"jobType":"zeta","totalJobs":3,"totalCost":30,"tokenCost":20,
		 "externalToolCost":6,"humanCost":4,"conversions":3,"deflections":1,
		 "totalValue":300,"averageValue":100,"costPerConversion":10,
		 "costPerOutcome":7.5,"roi":19900,"successRate":0.42},
		{"jobType":"alpha","totalJobs":1,"totalCost":10,"tokenCost":7,
		 "externalToolCost":2,"humanCost":1,"conversions":1,"deflections":0,
		 "totalValue":100,"averageValue":100,"costPerConversion":10,
		 "costPerOutcome":10,"roi":900,"successRate":1},
		{"jobType":"mid","totalJobs":2,"totalCost":20,"tokenCost":14,
		 "externalToolCost":4,"humanCost":2,"conversions":2,"deflections":1,
		 "totalValue":200,"averageValue":100,"costPerConversion":10,
		 "costPerOutcome":6.67,"roi":900,"successRate":0.5}
	],
	"summary":{"totalJobTypes":3,"totalJobs":6,"totalCost":60,
		"totalValue":600,"overallROI":900}
}`

// TestROISummarySortsByJobTypeAscending asserts POSITIONS, not presence.
// Row order is a property of our output rather than an assumption about a
// server contract the spec never states (D-29-10, D-27-08), and a Contains-only
// assertion would be green on every possible ordering.
//
// It also pins the two rate cells: a roi of 19900 renders 19900.00 and a
// successRate of 0.42 renders 0.42 — no multiplication and no percent suffix
// while the unit convention is unsettled (D-29-10).
func TestROISummarySortsByJobTypeAscending(t *testing.T) {
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, roiSummaryUnsortedFixture)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	require.NoError(t, c.Execute())

	out := buf.String()
	alpha, mid, zeta := strings.Index(out, "alpha"), strings.Index(out, "mid"), strings.Index(out, "zeta")
	require.NotEqual(t, -1, alpha)
	require.NotEqual(t, -1, mid)
	require.NotEqual(t, -1, zeta)
	assert.Less(t, alpha, mid, "alpha must be rendered above mid")
	assert.Less(t, mid, zeta, "mid must be rendered above zeta")

	assert.Contains(t, out, "19900.00", "roi renders as a bare two-decimal number")
	assert.Contains(t, out, "0.42", "successRate is not multiplied by 100")
	assert.NotContains(t, out, "19900.00%", "no percent suffix asserts a unit the schema does not state")
}

// TestROISummaryJSONIsVerbatim is the counterweight to the sort test: --json
// must still be the SERVER's document, in the server's order, with every field
// intact (D-29-09, RESEARCH Pitfall 9). The sort lives below the single
// output-mode branch precisely so it cannot reach this path.
func TestROISummaryJSONIsVerbatim(t *testing.T) {
	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, roiSummaryUnsortedFixture)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	require.NoError(t, c.Execute())

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))

	// Every top-level key the server sent survives, including the two the
	// table path reads and the one it does not.
	for _, key := range []string{"id", "period", "byJobType", "summary"} {
		assert.Containsf(t, doc, key, "--json must carry the server's %q key", key)
	}

	items, ok := doc["byJobType"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 3)
	order := make([]string, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]interface{})
		require.True(t, ok)
		order = append(order, fmt.Sprint(item["jobType"]))
	}
	assert.Equal(t, []string{"zeta", "alpha", "mid"}, order,
		"--json emits the server's order; the table-path sort must not reach it")

	first, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	// All fourteen declared fields, including the six the eight-column table
	// drops. Nothing is lost by the table's narrowing.
	for _, field := range []string{
		"jobType", "totalJobs", "totalCost", "tokenCost", "externalToolCost",
		"humanCost", "conversions", "deflections", "totalValue", "averageValue",
		"costPerConversion", "costPerOutcome", "roi", "successRate",
	} {
		assert.Containsf(t, first, field, "--json must carry the %q field the table drops", field)
	}
	assert.Len(t, first, 14, "exactly the fourteen fields the server sent")
}

// TestROISummaryEmptyByJobType pins the table-path-only empty state
// (economics_render.go:154-160), NOT the shape cmd/jobs/list.go:26-32 uses.
//
// The Summary section still renders: an empty byJobType is a real case (no jobs
// in the range) and the period totals are still worth showing. Under --json the
// document is the whole server object including its typed empty array, which is
// satisfied by the branch at the top of the render returning first — not by a
// second output-mode branch, which would discard the summary object entirely.
func TestROISummaryEmptyByJobType(t *testing.T) {
	const fixture = `{"period":{"start":"2025-09-01","end":"2025-09-30"},
		"byJobType":[],
		"summary":{"totalJobTypes":0,"totalJobs":0,"totalCost":0,
			"totalValue":1234.5,"overallROI":0}}`

	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixture)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "Total Job Types", "the Summary section still renders in full")
	assert.Contains(t, out, "$1,234.50", "the period totals are still worth showing")
	assert.Contains(t, out, "No job types are reported for this period.")
	assert.NotContains(t, out, "Cost/Conversion", "no header row is printed in place of the table")

	// The same fixture under --json: the whole server object, typed empty array
	// and summary object both intact.
	var buf2 bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf2, &buf2, true, false)

	c2 := newROISummaryCmd()
	c2.SetOut(&buf2)
	c2.SetArgs([]string{})
	require.NoError(t, c2.Execute())

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(buf2.Bytes(), &doc))
	assert.Contains(t, doc, "summary", "a second empty-state branch would have discarded this")
	items, ok := doc["byJobType"].([]interface{})
	require.True(t, ok, "byJobType must still be present and an array")
	assert.Empty(t, items)
}

// TestROISummaryUndeclaredNumericRendersDash is the test that goes red the
// moment a numeric cell is switched to the plain string helper or read through
// the lenient float reader: both return a confident zero for a value the server
// never declared (format.go:21-33).
//
// "Not declared" and "declared as zero" are different facts about a cost and
// must stay distinguishable in the same table (D-29-10, D-26-01).
func TestROISummaryUndeclaredNumericRendersDash(t *testing.T) {
	const fixture = `{"period":{"start":"2025-09-01","end":"2025-09-30"},
		"byJobType":[
			{"jobType":"absent","totalJobs":1,"totalCost":10,"totalValue":100,
			 "averageValue":100,"roi":900,"successRate":1},
			{"jobType":"declared-zero","totalJobs":1,"totalCost":10,"totalValue":100,
			 "averageValue":100,"costPerConversion":0,"roi":900,"successRate":1}
		],
		"summary":{"totalJobTypes":2,"totalJobs":2,"totalCost":20,
			"totalValue":200,"overallROI":900}}`

	analytics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixture)
	}))
	defer analytics.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://platform.invalid", "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = analytics.URL
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROISummaryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, undeclared, "an undeclared costPerConversion renders as the undeclared marker")
	assert.Contains(t, out, "$0.00", "a costPerConversion declared as 0 renders as a zero-valued amount")
}
