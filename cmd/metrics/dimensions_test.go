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

func TestDimensionsAnalyticsBaseAndBearerAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/analytics/filter-options/agents", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("x-api-key"))
		assert.Empty(t, r.URL.Query().Get("teamId"))
		assert.Empty(t, r.URL.Query().Get("tenantId"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded":{"items":[{"value":"agent-1","metricResult":42,"metricType":"count"}]},"page":{"totalPages":1}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "test-team", "test-tenant", "", false)
	cmd.APIClient.AnalyticsBaseURL = srv.URL
	cmd.APIClient.UseBearerAuth = true
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"agents"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "agent-1")
	assert.Contains(t, out, "42")
	assert.Contains(t, out, "count")
}

// TestDimensionsHelpDoesNotAdvertiseTimeRange guards against re-introducing a
// misleading --from/--to example: filter-options/{name} lookups are enumerations,
// not time-series, so the command must not advertise date-range filtering it never
// applies to the request (regression for the Phase 4 CR-01 finding).
func TestDimensionsHelpDoesNotAdvertiseTimeRange(t *testing.T) {
	c := newDimensionsCmd()
	assert.NotContains(t, c.Example, "--from")
	assert.NotContains(t, c.Example, "--to")
}

func TestDimensionsUnknownNameRejectedBeforeHTTP(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = srv.URL
	cmd.APIClient.UseBearerAuth = true
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"bogus"})
	err := c.Execute()

	require.Error(t, err)
	for _, name := range validDimensionNames {
		assert.Contains(t, err.Error(), name)
	}
	assert.False(t, called, "no HTTP call should be made for an invalid dimension name")
}

func TestDimensionsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded":{"items":[]},"page":{"totalPages":0}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = srv.URL
	cmd.APIClient.UseBearerAuth = true
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"models"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No metrics found.")
}

func TestDimensionsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded":{"items":[{"value":"vendor-1","metricResult":7,"metricType":"sum"}]},"page":{"totalPages":1}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.APIClient.AnalyticsBaseURL = srv.URL
	cmd.APIClient.UseBearerAuth = true
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"vendors"})
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "vendor-1", result[0]["value"])
}
