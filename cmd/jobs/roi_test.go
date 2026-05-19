package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roiFixture is the OAS-derived 13-field example response (Plan 13-03
// <interfaces> block) used across the happy-path tests.
const roiFixture = `{
    "agenticJobId":"loan-app-1",
    "agenticJobName":"Process Loan",
    "agenticJobType":"loan-processing",
    "executionStatus":"SUCCESS",
    "hasOutcome":true,
    "totalCost":2.5,
    "outcomeValue":150,
    "outcomeCurrency":"USD",
    "roi":19900,
    "transactionCount":3,
    "inputTokens":8000,
    "outputTokens":555,
    "totalTokens":8555
}`

// TestROI is the load-bearing JOBS-07 test: GET /v2/api/jobs/<id>/roi renders
// a 2-column key-value table with currency-formatted Total Cost / Outcome
// Value / ROI % cells. Mirrors cmd/jobs/get_test.go scaffold.
func TestROI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1/roi", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, roiFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROICmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	// Headers and structure
	assert.Contains(t, out, "Metric")
	assert.Contains(t, out, "Value")
	// Row labels
	assert.Contains(t, out, "Total Cost")
	// Currency formatting (Plan 13-01 FormatCurrency: USD prefix "$", 2 decimals)
	assert.Contains(t, out, "$2.50")
	assert.Contains(t, out, "$150.00")
	// ROI % formatting — fixture roi=19900 → "19900.00%"
	assert.Contains(t, out, "19900.00%")
	// Identity row values pulled via str()
	assert.Contains(t, out, "Process Loan")
	assert.Contains(t, out, "loan-processing")
	// Token row value (proves explicit ordering reaches row 13)
	assert.Contains(t, out, "8555")
}

// TestROIJSON asserts that --json mode returns the raw JobROIResource_Read
// response unmodified (all 13 fields plus outcomeCurrency preserved).
func TestROIJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/jobs/loan-app-1/roi", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, roiFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newROICmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "loan-app-1", result["agenticJobId"])
	// JSON numeric assertions: json.Unmarshal yields float64 by default.
	totalCost, ok := result["totalCost"].(float64)
	require.True(t, ok, "totalCost must be a JSON number")
	assert.InDelta(t, 2.5, totalCost, 0.0001)
	roi, ok := result["roi"].(float64)
	require.True(t, ok, "roi must be a JSON number")
	assert.InDelta(t, 19900, roi, 0.0001)
	// Proves the dynamic render preserved every field, including the one
	// that's used as a formatting driver (not its own table row).
	assert.Contains(t, result, "outcomeCurrency")
}

// TestROIPathEscape pins the CF-13/D-25 path-construction contract.
//
// Character locked at planning time. Per .planning/phases/.../13-03-PLAN.md
// <action> block, the LOCKED character is "/" — verified by reading
// internal/validate/validate.go (only ?, &, #, %, ../, ..\ and ASCII control
// chars are rejected; bare "/" is allowed). url.PathEscape("/") → "%2F"
// (Go stdlib path-segment encoder). An id containing "/" cleanly passes the
// validator AND requires URL escaping, making it the canonical fixture for
// proving the code uses url.PathEscape rather than naive interpolation.
//
// If at execution time ValidResourceID was found to reject "/", the executor
// would have to STOP and surface that as a validator regression. The
// validator was re-read at executor start and bare "/" is still accepted.
func TestROIPathEscape(t *testing.T) {
	var receivedEscapedPath string
	var receivedDecodedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Go's net/http exposes the wire-format escaped path via EscapedPath()
		// (which returns RawPath when set, else computes from Path). r.URL.Path
		// is the decoded form (%2F → /), so we must check the escaped form to
		// prove url.PathEscape was applied to the id before the request was
		// sent — otherwise the assertion would pass even with naive string
		// interpolation that lets the bare "/" through.
		receivedEscapedPath = r.URL.EscapedPath()
		receivedDecodedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"agenticJobId":"loan/app/1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROICmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan/app/1"})
	err := c.Execute()

	require.NoError(t, err)
	// LOAD-BEARING: slashes inside the id segment must be encoded to %2F by
	// url.PathEscape; the path-separator slashes around the segment remain
	// literal. The wire-format (EscapedPath) preserves the %2F; the decoded
	// form (Path) collapses them back to "/". Assert both to make intent
	// unambiguous.
	assert.Equal(t, "/v2/api/jobs/loan%2Fapp%2F1/roi", receivedEscapedPath,
		"wire-format path must contain %%2F (proves url.PathEscape was applied)")
	assert.Equal(t, "/v2/api/jobs/loan/app/1/roi", receivedDecodedPath,
		"decoded form: server-side r.URL.Path collapses %%2F back to /")
}

// TestROITeamId asserts that the teamId query parameter is auto-injected by
// Client.Do when the client is constructed with a TeamID. Mirrors
// cmd/jobs/update_test.go TestUpdateJobTeamId.
func TestROITeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, roiFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newROICmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}
