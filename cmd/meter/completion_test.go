package meter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeterCompletion(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/ai/completions", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "gpt-4",
		"--provider", "openai",
		"--input-tokens", "500",
		"--output-tokens", "200",
		"--total-tokens", "700",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--is-streamed",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "cmp-123")
	assert.Equal(t, "gpt-4", receivedBody["model"])
	assert.Equal(t, "openai", receivedBody["provider"])
	assert.Equal(t, float64(500), receivedBody["inputTokenCount"])
	assert.Equal(t, float64(200), receivedBody["outputTokenCount"])
	assert.Equal(t, float64(700), receivedBody["totalTokenCount"])
	assert.Equal(t, "END", receivedBody["stopReason"])
	assert.Equal(t, true, receivedBody["isStreamed"])
}

func TestMeterCompletionWithOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-456", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "claude-3-opus",
		"--provider", "anthropic",
		"--input-tokens", "1000",
		"--output-tokens", "500",
		"--total-tokens", "1500",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:10Z",
		"--request-duration", "10000",
		"--is-streamed",
		"--total-cost", "0.045",
		"--agent", "my-agent",
		"--environment", "production",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, 0.045, receivedBody["totalCost"])
	assert.Equal(t, "my-agent", receivedBody["agent"])
	assert.Equal(t, "production", receivedBody["environment"])
}

func TestMeterCompletionMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--model", "gpt-4"})
	err := c.Execute()

	assert.Error(t, err)
}

func TestMeterCompletionJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "gpt-4",
		"--provider", "openai",
		"--input-tokens", "500",
		"--output-tokens", "200",
		"--total-tokens", "700",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--is-streamed",
	})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "cmp-123", result["id"])
}
