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

// TestFunnel exercises the basic populated GET path for
// `revenium jobs conversion-funnel`. Asserts the endpoint path, HTTP method,
// the three synthesized stage labels (Total / Successful / Converted), the
// three count values, and the three Conversion % cells (the literal 100.00%
// for the Total stage by definition, then server-supplied successRate * 100
// and conversionRate * 100 formatted to 2 decimal places).
//
// CRITICAL (RESEARCH Pitfall 5): the rate cells come from the server-supplied
// successRate / conversionRate decimals — the CLI must NOT recompute them
// from successfulJobs / totalJobs client-side. The fixture uses successRate
// 0.85 and conversionRate 0.71 to make rate sourcing observable separately
// from the underlying counts.
func TestFunnel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v2/api/jobs/conversion-funnel", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"totalJobs":100,"successfulJobs":85,"convertedJobs":60,"successRate":0.85,"conversionRate":0.71}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newConversionFunnelCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	// Stage labels
	assert.Contains(t, out, "Total")
	assert.Contains(t, out, "Successful")
	assert.Contains(t, out, "Converted")
	// Counts
	assert.Contains(t, out, "100")
	assert.Contains(t, out, "85")
	assert.Contains(t, out, "60")
	// Conversion % cells
	assert.Contains(t, out, "100.00%") // Total stage — literal
	assert.Contains(t, out, "85.00%")  // successRate * 100
	assert.Contains(t, out, "71.00%")  // conversionRate * 100
}

// TestFunnelFilters asserts that each of the four CLI flags translates to its
// corresponding OAS field name in the query string (RESEARCH §A5):
//
//	--from         -> startDate
//	--to           -> endDate
//	--job-type     -> jobType
//	--environment  -> environment
func TestFunnelFilters(t *testing.T) {
	var receivedQuery map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		receivedQuery = map[string]string{
			"startDate":   q.Get("startDate"),
			"endDate":     q.Get("endDate"),
			"jobType":     q.Get("jobType"),
			"environment": q.Get("environment"),
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"totalJobs":1,"successfulJobs":1,"convertedJobs":1,"successRate":1.0,"conversionRate":1.0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newConversionFunnelCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--from", "2025-09-01T00:00:00Z",
		"--to", "2025-09-30T23:59:59Z",
		"--job-type", "loan_processing",
		"--environment", "production",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "2025-09-01T00:00:00Z", receivedQuery["startDate"], "--from must map to startDate")
	assert.Equal(t, "2025-09-30T23:59:59Z", receivedQuery["endDate"], "--to must map to endDate")
	assert.Equal(t, "loan_processing", receivedQuery["jobType"], "--job-type must map to jobType")
	assert.Equal(t, "production", receivedQuery["environment"], "--environment must map to environment")
}

// TestFunnelNoFilters asserts that absent flags are NOT serialized as empty
// query parameters — proves the c.Flags().Changed gating works. teamId is
// expected (auto-injected by the API client) but the four filter keys must
// be absent.
func TestFunnelNoFilters(t *testing.T) {
	var rawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"totalJobs":0,"successfulJobs":0,"convertedJobs":0,"successRate":0.0,"conversionRate":0.0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newConversionFunnelCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	// None of the four filter keys may appear in the query string — they
	// were not Changed() so url.Values must remain empty.
	assert.NotContains(t, rawQuery, "startDate=")
	assert.NotContains(t, rawQuery, "endDate=")
	assert.NotContains(t, rawQuery, "jobType=")
	assert.NotContains(t, rawQuery, "environment=")
}

// TestFunnelJSON exercises --json: the server's raw flat object must be
// returned UNMODIFIED, not the synthesized 3-row table data. Proves
// renderFunnel branches on IsJSON() at the top per RESEARCH §A5 + D-04.
func TestFunnelJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"totalJobs":100,"successfulJobs":85,"convertedJobs":60,"successRate":0.85,"conversionRate":0.71}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newConversionFunnelCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	// All five raw fields must be present — proves --json emits the raw
	// flat object, not the synthesized Total/Successful/Converted shape.
	assert.Equal(t, float64(100), result["totalJobs"])
	assert.Equal(t, float64(85), result["successfulJobs"])
	assert.Equal(t, float64(60), result["convertedJobs"])
	assert.InDelta(t, 0.85, result["successRate"], 1e-9)
	assert.InDelta(t, 0.71, result["conversionRate"], 1e-9)
}

// TestFunnelTeamId asserts teamId auto-injection by Client.Do
// (internal/api/client.go:68-74) when the client is constructed with a
// TeamID. Mirrors cmd/jobs/update_test.go:66-86.
func TestFunnelTeamId(t *testing.T) {
	var receivedTeamID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTeamID = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"totalJobs":0,"successfulJobs":0,"convertedJobs":0,"successRate":0.0,"conversionRate":0.0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newConversionFunnelCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedTeamID, "teamId must be present as query parameter")
}
