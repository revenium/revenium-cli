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

// populatedTransactionsBody is the canonical OAS JobTimelineResource_Read fixture
// used by TestTransactions and TestTransactionsJSON. Cost 0.010468 rounds via
// FormatCurrency ("%.2f") to "$0.01", which the populated-table assertion pins.
const populatedTransactionsBody = `{
    "transactions": [
        {
            "transactionId": "txn-1",
            "timestamp": "2026-04-26T04:16:31.874Z",
            "model": "claude-3-5-haiku-latest",
            "provider": "anthropic",
            "duration": 5207,
            "cost": 0.010468,
            "inputTokens": 7693,
            "outputTokens": 555,
            "totalTokens": 8248,
            "status": "success"
        }
    ],
    "totalCount": 1
}`

// errorTransactionsBody pins the second StatusColumn arm (status=error renders
// red via Plan 13-01's statusStyle "error" extension).
const errorTransactionsBody = `{
    "transactions": [
        {
            "transactionId": "txn-2",
            "timestamp": "2026-04-26T04:17:00.000Z",
            "model": "claude-3-5-haiku-latest",
            "provider": "anthropic",
            "duration": 1000,
            "cost": 0.005,
            "status": "error"
        }
    ],
    "totalCount": 1
}`

// emptyTransactionsBody is the wrapper-object empty fixture per RESEARCH §A3 —
// totalCount=0 must be preserved in --json mode (NOT collapsed to bare []).
const emptyTransactionsBody = `{"transactions": [], "totalCount": 0}`

// TestTransactions is the populated 4-column table case: header row + the four
// header tokens, the timestamp/model values, FormatCurrency-rendered "$0.01",
// and the "success" status token (may be ANSI-wrapped — assert.Contains finds it).
func TestTransactions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/jobs/loan-app-1/transactions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, populatedTransactionsBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTransactionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	// 4 column headers
	assert.Contains(t, out, "Timestamp")
	assert.Contains(t, out, "Model")
	assert.Contains(t, out, "Cost")
	assert.Contains(t, out, "Status")
	// row values
	assert.Contains(t, out, "2026-04-26T04:16:31.874Z")
	assert.Contains(t, out, "claude-3-5-haiku-latest")
	// FormatCurrency rounds 0.010468 (%.2f) -> $0.01 — pins the Cost cell wiring
	assert.Contains(t, out, "$0.01")
	// status token is present in the rendered cell (statusStyle wraps it in ANSI; substring still matches)
	assert.Contains(t, out, "success")
}

// TestTransactionsError pins the StatusColumn=3 "error" arm (Plan 13-01
// extended statusStyle to include lowercase "error" in the red case).
func TestTransactionsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, errorTransactionsBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTransactionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-2"})
	err := c.Execute()

	require.NoError(t, err)
	// The literal status token appears in the rendered Status cell — even if
	// statusStyle colors it red, the substring is still present.
	assert.Contains(t, buf.String(), "error")
}

// TestTransactionsEmpty pins the empty-array non-JSON stdout text per D-05.
func TestTransactionsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, emptyTransactionsBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTransactionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No transactions found.")
}

// TestTransactionsEmptyJSON is the critical wrapper-preservation case
// (RESEARCH §A3): the empty JSON payload MUST be the wrapper object
// {"transactions":[],"totalCount":0}, NOT the bare [] that list.go emits.
// We unmarshal into map[string]interface{} (NOT []interface{}) and assert
// both wrapper keys are present.
func TestTransactionsEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, emptyTransactionsBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newTransactionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	// LOAD-BEARING per RESEARCH §A3: the JSON payload is the WRAPPER, not bare [].
	// Decoding into map[string]interface{} succeeds; decoding into []interface{} would not.
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "empty-state JSON must decode as a wrapper object (map), NOT a bare array")
	// totalCount preserved as 0
	require.Contains(t, result, "totalCount")
	assert.Equal(t, float64(0), result["totalCount"])
	// transactions key preserved as an empty slice
	require.Contains(t, result, "transactions")
	txnsAny, ok := result["transactions"].([]interface{})
	require.True(t, ok, "transactions key must be a JSON array")
	assert.Empty(t, txnsAny)
}

// TestTransactionsJSON is the populated --json case: full wrapper preserved.
func TestTransactionsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, populatedTransactionsBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newTransactionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	// totalCount preserved
	require.Contains(t, result, "totalCount")
	assert.Equal(t, float64(1), result["totalCount"])
	// transactions array has length 1
	txnsAny, ok := result["transactions"].([]interface{})
	require.True(t, ok)
	assert.Len(t, txnsAny, 1)
}

// TestTransactionsTeamId asserts the auto-injected teamId query parameter is
// present on the GET request when the client is constructed with a TeamID.
// Mirrors cmd/jobs/update_test.go TestUpdateJobTeamId.
func TestTransactionsTeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, populatedTransactionsBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTransactionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}
