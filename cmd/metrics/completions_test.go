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
		fmt.Fprint(w, `[{"id": "txn-c-1", "transactionId": "txn-c-1", "model": "gpt-3.5-turbo", "inputTokenCount": 20000, "outputTokenCount": 5000, "cacheReadTokenCount": 1000, "reasoningTokenCount": 500, "timeToFirstToken": 250, "tokensPerMinute": 3000, "requestDuration": 5000, "stopReason": "END", "totalCost": 0.125, "organization": {"id": "org-1", "label": "Acme Corp"}, "agent": "chat-bot", "subscriberCredential": {"id": "cred-1", "label": "Alice"}, "squadId": "sq-c-1", "squadName": "Loan Processing", "squadRole": "planner"}]`)
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
	assert.Contains(t, out, "Squad")
	// squadName present -> Squad column renders the name, not the id (D-06)
	assert.Contains(t, out, "Loan Processing")
}

func TestCompletionMetricsHeadersEndWithSquad(t *testing.T) {
	assert.Equal(t, "Squad", completionsTableDef.Headers[len(completionsTableDef.Headers)-1])
}

// TestCompletionMetricsSquadIDOnly proves that when a row is tagged with a
// squadId but no squadName, the Squad column falls back to the id (D-06).
func TestCompletionMetricsSquadIDOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "txn-c-2", "transactionId": "txn-c-2", "model": "gpt-3.5-turbo", "squadId": "sq-c-2"}]`)
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
	assert.Contains(t, buf.String(), "sq-c-2")
}

// TestCompletionMetricsUntaggedSquadBlank proves an untagged row renders a
// blank Squad cell (no em-dash placeholder, D-07).
func TestCompletionMetricsUntaggedSquadBlank(t *testing.T) {
	row := map[string]interface{}{"id": "txn-c-3", "transactionId": "txn-c-3", "model": "gpt-3.5-turbo"}
	assert.Equal(t, "", squadCell(row))
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

// TestCompletionsGet proves `metrics completions get <id>` PathEscapes the
// id, hits the platform host with x-api-key auth (not Authorization), and
// renders the single AICompletionMetricResource response.
func TestCompletionsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is the decoded form (%2F -> /); EscapedPath() preserves
		// the wire-format %2F, proving url.PathEscape was applied (mirrors
		// cmd/jobs/roi_test.go's TestROIPathEscape).
		assert.Equal(t, "/v2/api/sources/metrics/ai/completions/txn%2Fabc", r.URL.EscapedPath())
		assert.Equal(t, "/v2/api/sources/metrics/ai/completions/txn/abc", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "l3Ka8j", "transactionId": "txn-test-002", "model": "gpt-4-turbo", "inputTokenCount": 75, "outputTokenCount": 125, "totalCost": 0.23, "stopReason": "END", "organization": {"id": "org-1", "label": "Acme Corp"}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"get", "txn/abc"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "gpt-4-turbo")
	assert.Contains(t, out, "txn-test-002")
	assert.Contains(t, out, "Acme Corp")
	assert.NotContains(t, out, "UseBearerAuth")
}

// TestCompletionsGetRequiresID proves a bare `get` (no id) is rejected by
// ExactArgs(1) before any HTTP call is made.
func TestCompletionsGetRequiresID(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionsCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"get"})
	err := c.Execute()

	require.Error(t, err)
}

// TestCompletionsPrompts proves `metrics completions prompts <id>` GETs the
// single-object PromptDataResource (not a list) and renders its fields.
func TestCompletionsPrompts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/metrics/ai/completions/txn-abc123/prompts", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"systemPrompt": "You are a helpful assistant.", "inputMessages": "[{\"role\": \"user\", \"content\": \"Hello\"}]", "outputResponse": "I'm doing well!", "promptsTruncated": false}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prompts", "txn-abc123"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "You are a helpful assistant.")
	assert.Contains(t, out, "I'm doing well!")
	assert.Contains(t, out, "false")
}

// TestCompletionsReferenceData proves `metrics completions reference-data`
// takes no args, hits the no-id platform path, and joins the two string-array
// fields for display.
func TestCompletionsReferenceData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/metrics/ai/completions/reference-data", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"traceTypes": ["chat", "document_analysis"], "stopReasons": ["END", "TIMEOUT"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"reference-data"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "chat, document_analysis")
	assert.Contains(t, out, "END, TIMEOUT")
}

// TestCompletionsReferenceDataNoArgs proves reference-data rejects a
// stray positional arg.
func TestCompletionsReferenceDataNoArgs(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionsCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"reference-data", "unexpected"})
	err := c.Execute()

	require.Error(t, err)
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
