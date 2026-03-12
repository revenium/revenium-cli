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

func TestVideoMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/metrics/ai/video", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("startDate"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "v-1", "model": "sora", "totalDuration": 120, "totalCost": 2.50}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newVideoCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "sora")
	assert.Contains(t, out, "120")
	assert.Contains(t, out, "$2.5000")
}

func TestVideoMetricsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newVideoCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No metrics found.")
}

func TestVideoMetricsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": "v-1", "model": "sora", "totalDuration": 120, "totalCost": 2.50}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newVideoCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "v-1", result[0]["id"])
}
