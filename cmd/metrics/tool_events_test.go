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

func TestToolEventMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/metrics/tool/events", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("startDate"))
		assert.NotEmpty(t, r.URL.Query().Get("endDate"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "te-1", "tool": "code-search", "invocations": 250, "totalCost": 0.125}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newToolEventsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "te-1")
	assert.Contains(t, out, "code-search")
	assert.Contains(t, out, "250")
	assert.Contains(t, out, "$0.1250")
}

func TestToolEventMetricsEmpty(t *testing.T) {
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

	c := newToolEventsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No tool event metrics found.")
}

func TestToolEventMetricsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "te-1", "tool": "code-search", "invocations": 250, "totalCost": 0.125}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newToolEventsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "te-1", result[0]["id"])
}
