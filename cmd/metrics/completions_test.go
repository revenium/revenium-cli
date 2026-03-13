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

func TestCompletionMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/metrics/ai/completions", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("startDate"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "txn-c-1", "transactionId": "txn-c-1", "model": "gpt-3.5-turbo", "inputTokenCount": 20000, "outputTokenCount": 5000, "cacheReadTokenCount": 1000, "reasoningTokenCount": 500, "timeToFirstToken": 250, "tokensPerMinute": 3000, "requestDuration": 5000, "stopReason": "END", "totalCost": 0.125, "organization": {"id": "org-1", "label": "Acme Corp"}, "agent": "chat-bot", "subscriberCredential": {"id": "cred-1", "label": "Alice"}}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newCompletionsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "gpt-3.5-turbo")
	assert.Contains(t, out, "20,000")
	assert.Contains(t, out, "5,000")
	assert.Contains(t, out, "1,000")
	assert.Contains(t, out, "500")
	assert.Contains(t, out, "250ms")
	assert.Contains(t, out, "3,000")
	assert.Contains(t, out, "5.00s")
	assert.Contains(t, out, "END")
	assert.Contains(t, out, "$0.12")
	assert.Contains(t, out, "Acme Corp")
	assert.Contains(t, out, "chat-bot")
	assert.Contains(t, out, "Alice")
}

func TestCompletionMetricsEmpty(t *testing.T) {
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

	c := newCompletionsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No metrics found.")
}

func TestCompletionMetricsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "txn-c-1", "transactionId": "txn-c-1", "model": "gpt-3.5-turbo", "totalTokenCount": 25000, "totalCost": 0.125}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newCompletionsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "txn-c-1", result[0]["id"])
}
