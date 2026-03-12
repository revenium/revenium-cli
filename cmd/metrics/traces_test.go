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
		assert.Equal(t, "/v2/api/traces", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("startDate"))
		assert.NotEmpty(t, r.URL.Query().Get("endDate"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"traceId": "t-1", "model": "gpt-4", "totalTokens": 500, "totalCost": 0.02}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
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
	assert.Contains(t, out, "$0.0200")
}

func TestTraceMetricsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
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
		fmt.Fprint(w, `[{"traceId": "t-1", "model": "gpt-4", "totalTokens": 500, "totalCost": 0.02}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
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
			{"traceId": "t-1", "model": "gpt-4", "totalTokens": 500, "totalCost": 0.02},
			{"traceId": "t-1", "model": "gpt-4", "totalTokens": 300, "totalCost": 0.01},
			{"traceId": "t-2", "model": "claude-3", "totalTokens": 1000, "totalCost": 0.05},
			{"traceId": "t-2", "model": "claude-3", "totalTokens": 200, "totalCost": 0.01}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
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
	assert.Contains(t, out, "$0.0300")
	// t-2 group: 1000+200=1200 tokens, 0.05+0.01=0.06 cost, 2 entries
	assert.Contains(t, out, "1,200")
	assert.Contains(t, out, "$0.0600")
}

func TestTracesJSONRaw(t *testing.T) {
	// Verify JSON mode passes ungrouped raw data
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"traceId": "t-1", "model": "gpt-4", "totalTokens": 500, "totalCost": 0.02},
			{"traceId": "t-1", "model": "gpt-4", "totalTokens": 300, "totalCost": 0.01}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
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
