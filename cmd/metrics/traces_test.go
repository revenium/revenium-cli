package metrics

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

func TestTraceMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/metrics/ai/traces", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("startDate"))
		assert.NotEmpty(t, r.URL.Query().Get("endDate"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500, "totalCost": 0.02}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "t-1")
	assert.Contains(t, out, "gpt-4")
	assert.Contains(t, out, "500")
	assert.Contains(t, out, "$0.02")
}

func TestTraceMetricsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No traces found.")
}

func TestTraceMetricsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500, "totalCost": 0.02}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "t-1", result[0]["traceId"])
}

func TestTracesGrouping(t *testing.T) {
	// Mock returns 4 entries with 2 distinct traceIds
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500, "totalCost": 0.02},
			{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 300, "totalCost": 0.01},
			{"traceId": "t-2", "model": "claude-3", "totalTokenCount": 1000, "totalCost": 0.05},
			{"traceId": "t-2", "model": "claude-3", "totalTokenCount": 200, "totalCost": 0.01}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	// Should show 2 grouped rows, not 4 raw rows
	assert.Contains(t, out, "t-1")
	assert.Contains(t, out, "t-2")
	// t-1 group: 500+300=800 tokens, 0.02+0.01=0.03 cost, 2 entries
	assert.Contains(t, out, "800")
	assert.Contains(t, out, "$0.03")
	// t-2 group: 1000+200=1200 tokens, 0.05+0.01=0.06 cost, 2 entries
	assert.Contains(t, out, "1,200")
	assert.Contains(t, out, "$0.06")
}

func TestTracesHeadersEndWithSquad(t *testing.T) {
	assert.Equal(t, "Squad", tracesTableDef.Headers[len(tracesTableDef.Headers)-1])
}

// TestGroupByTraceIdThreadsSquadId proves squadId is threaded through
// aggregation (first-non-empty-wins, same as model/source) — Pitfall 3.
func TestGroupByTraceIdThreadsSquadId(t *testing.T) {
	metrics := []map[string]interface{}{
		{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500.0, "totalCost": 0.02, "squadId": ""},
		{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 300.0, "totalCost": 0.01, "squadId": "sq-trace-1"},
	}
	grouped := groupByTraceId(metrics)
	require.Len(t, grouped, 1)
	assert.Equal(t, "sq-trace-1", str(grouped[0], "squadId"))
}

// TestTracesSquadColumnNonBlank proves the Squad column renders non-blank
// on `metrics traces` for a squad-tagged trace (fails if groupByTraceId is
// not touched — proves the threading end-to-end, not just at unit level).
func TestTracesSquadColumnNonBlank(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500, "totalCost": 0.02, "squadId": "sq-trace-1"},
			{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 300, "totalCost": 0.01, "squadId": "sq-trace-1"}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Squad")
	assert.Contains(t, out, "sq-trace-1")
}

// TestTracesSquadIDFilterJSON proves --squad-id filters the raw pre-grouping
// slice, so JSON mode (which renders raw ungrouped metrics) reflects the
// filter too (filter-then-group, not group-then-filter — Pitfall 3).
func TestTracesSquadIDFilterJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500, "totalCost": 0.02, "squadId": "sq-x"},
			{"traceId": "t-2", "model": "claude-3", "totalTokenCount": 300, "totalCost": 0.01, "squadId": "sq-y"}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--squad-id", "sq-x"})
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "t-1", result[0]["traceId"])
}

// TestTracesSquadIDForcesFetchAll proves --squad-id overrides an explicit
// --page-size and fetches every page before filtering (D-10 / Pitfall 5).
func TestTracesSquadIDForcesFetchAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "0", "":
			fmt.Fprint(w, `{"_embedded":{"tracesList":[{"traceId":"t-1","model":"gpt-4","totalTokenCount":100,"totalCost":0.01,"squadId":"sq-other-1"}]},"page":{"totalPages":3}}`)
		case "1":
			fmt.Fprint(w, `{"_embedded":{"tracesList":[{"traceId":"t-2","model":"gpt-4","totalTokenCount":100,"totalCost":0.01,"squadId":"sq-other-2"}]},"page":{"totalPages":3}}`)
		case "2":
			fmt.Fprint(w, `{"_embedded":{"tracesList":[{"traceId":"t-3","model":"gpt-4","totalTokenCount":100,"totalCost":0.01,"squadId":"sq-target"}]},"page":{"totalPages":3}}`)
		default:
			fmt.Fprint(w, `{"_embedded":{"tracesList":[]},"page":{"totalPages":3}}`)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--squad-id", "sq-target", "--page-size", "1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "t-3")
	assert.NotContains(t, out, "t-1")
	assert.NotContains(t, out, "t-2")
}

// TestTracesSquadIDNoMatch proves zero matches after filtering is a normal
// empty result (exit 0), not an error (D-10).
func TestTracesSquadIDNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500, "totalCost": 0.02, "squadId": "sq-trace-1"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--squad-id", "no_match"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No traces found.")
}

func TestTracesJSONRaw(t *testing.T) {
	// Verify JSON mode passes ungrouped raw data
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 500, "totalCost": 0.02},
			{"traceId": "t-1", "model": "gpt-4", "totalTokenCount": 300, "totalCost": 0.01}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newTracesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	// Raw data should have 2 entries (ungrouped), not 1 grouped entry
	assert.Len(t, result, 2)
	assert.Equal(t, "t-1", result[0]["traceId"])
	assert.Equal(t, "t-1", result[1]["traceId"])
}
