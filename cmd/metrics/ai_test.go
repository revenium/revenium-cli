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
		fmt.Fprint(w, `[{"id": "txn-m-1", "transactionId": "txn-m-1", "model": "gpt-4", "inputTokenCount": 1000, "outputTokenCount": 500, "cacheReadTokenCount": 200, "reasoningTokenCount": 100, "timeToFirstToken": 180, "tokensPerMinute": 2500, "requestDuration": 3200, "stopReason": "END", "totalCost": 0.05, "organization": {"id": "org-1", "label": "Acme"}, "agent": "assistant", "subscriberCredential": {"id": "cred-1", "label": "Bob"}}]`)
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
