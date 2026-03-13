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

func TestMeterToolEvent(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/tool/events", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "te-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newToolEventCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--tool-id", "search-api",
		"--duration-ms", "150",
		"--success",
		"--timestamp", "2024-01-15T10:00:00Z",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "te-123")
	assert.Equal(t, "search-api", receivedBody["toolId"])
	assert.Equal(t, float64(150), receivedBody["durationMs"])
	assert.Equal(t, true, receivedBody["success"])
}

func TestMeterToolEventWithOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "te-456", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newToolEventCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--tool-id", "db-query",
		"--duration-ms", "5000",
		"--success=false",
		"--timestamp", "2024-01-15T10:00:00Z",
		"--error-message", "connection timeout",
		"--agent", "my-agent",
		"--usage-metadata", `{"queries": 5}`,
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "connection timeout", receivedBody["errorMessage"])
	assert.Equal(t, "my-agent", receivedBody["agent"])
	usageMeta := receivedBody["usageMetadata"].(map[string]interface{})
	assert.Equal(t, float64(5), usageMeta["queries"])
}

func TestMeterToolEventMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newToolEventCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--tool-id", "search-api"})
	err := c.Execute()

	assert.Error(t, err)
}

func TestMeterToolEventInvalidUsageMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "te-789"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newToolEventCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{
		"--tool-id", "search-api",
		"--duration-ms", "150",
		"--success",
		"--timestamp", "2024-01-15T10:00:00Z",
		"--usage-metadata", "not-json",
	})
	err := c.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--usage-metadata must be valid JSON")
}
