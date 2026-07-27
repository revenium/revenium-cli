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

func TestAIMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/metrics/ai", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("startDate"))
		assert.NotEmpty(t, r.URL.Query().Get("endDate"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "txn-m-1", "transactionId": "txn-m-1", "model": "gpt-4", "inputTokenCount": 1000, "outputTokenCount": 500, "cacheReadTokenCount": 200, "reasoningTokenCount": 100, "timeToFirstToken": 180, "tokensPerMinute": 2500, "requestDuration": 3200, "stopReason": "END", "totalCost": 0.05, "organization": {"id": "org-1", "label": "Acme"}, "agent": "assistant", "subscriberCredential": {"id": "cred-1", "label": "Bob"}, "squadId": "sq-ai-1"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newAICmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "gpt-4")
	assert.Contains(t, out, "1,000")
	assert.Contains(t, out, "500")
	assert.Contains(t, out, "200")
	assert.Contains(t, out, "100")
	assert.Contains(t, out, "180ms")
	assert.Contains(t, out, "2,500")
	assert.Contains(t, out, "3.20s")
	assert.Contains(t, out, "END")
	assert.Contains(t, out, "$0.05")
	assert.Contains(t, out, "Acme")
	assert.Contains(t, out, "assistant")
	assert.Contains(t, out, "Bob")
	assert.Contains(t, out, "Squad")
	assert.Contains(t, out, "sq-ai-1")
}

func TestAIMetricsHeadersEndWithSquad(t *testing.T) {
	assert.Equal(t, "Squad", aiTableDef.Headers[len(aiTableDef.Headers)-1])
}

func TestAIMetricsEmpty(t *testing.T) {
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

	c := newAICmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No metrics found.")
}

func TestAIMetricsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "txn-m-1", "transactionId": "txn-m-1", "model": "gpt-4", "totalTokenCount": 1500, "totalCost": 0.05}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newAICmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "txn-m-1", result[0]["id"])
}

// TestAIMetricsSquadIDForcesFetchAll proves --squad-id overrides an explicit
// --page-size and fetches every page before filtering (D-10 / Pitfall 5).
// The server ignores the requested page size and instead paginates by its
// own 3-page fixture, keyed off the "page" query param.
func TestAIMetricsSquadIDForcesFetchAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "0", "":
			fmt.Fprint(w, `{"_embedded":{"aiList":[{"id":"a-1","transactionId":"a-1","model":"gpt-4","squadId":"sq-other-1"}]},"page":{"totalPages":3}}`)
		case "1":
			fmt.Fprint(w, `{"_embedded":{"aiList":[{"id":"a-2","transactionId":"a-2","model":"gpt-4","squadId":"sq-other-2"}]},"page":{"totalPages":3}}`)
		case "2":
			fmt.Fprint(w, `{"_embedded":{"aiList":[{"id":"a-3","transactionId":"a-3","model":"gpt-4","squadId":"sq-target"}]},"page":{"totalPages":3}}`)
		default:
			fmt.Fprint(w, `{"_embedded":{"aiList":[]},"page":{"totalPages":3}}`)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newAICmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--squad-id", "sq-target", "--page-size", "1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "a-3")
	assert.NotContains(t, out, "a-1")
	assert.NotContains(t, out, "a-2")
}

// TestAIMetricsSquadIDNoMatch proves zero matches after filtering is a
// normal empty result (exit 0), not an error (D-10).
func TestAIMetricsSquadIDNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "txn-m-1", "transactionId": "txn-m-1", "model": "gpt-4", "squadId": "sq-ai-1"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newAICmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--squad-id", "no_match"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No metrics found.")
}

// TestAIMetricsSquadIDExactMatch proves --squad-id is an exact-match filter
// over the already-fetched result set (D-09).
func TestAIMetricsSquadIDExactMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "a-1", "transactionId": "a-1", "model": "gpt-4", "squadId": "sq-x"}, {"id": "a-2", "transactionId": "a-2", "model": "gpt-4", "squadId": "sq-y"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newAICmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--squad-id", "sq-x"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "a-1")
	assert.NotContains(t, out, "a-2")
}
