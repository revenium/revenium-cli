package anomalies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

const dimensionsFixture = `{
	"providers": ["openai", "anthropic"],
	"workspaces": [{"id": "ws-1", "name": "Default"}],
	"apiKeys": [{"id": "key-1", "hint": "sk-***abcd"}],
	"models": ["gpt-4", "claude-3"],
	"supportedMetricsByProvider": {"openai": ["cost"]},
	"supportedDimensions": ["WORKSPACE", "API_KEY", "MODEL", "PROVIDER"]
}`

// TestAnomaliesDimensions asserts the request path is
// /v2/api/sources/ai/anomaly/provider-dimensions and the table-mode summary
// renders counts (not raw nested Go literals).
func TestAnomaliesDimensions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/anomaly/provider-dimensions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, dimensionsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "2") // 2 providers, 2 models
	assert.Contains(t, out, "4") // 4 supported dimensions
	assert.Contains(t, out, "use --json for full detail")
	// Must not spill raw Go-literal nested structs (e.g. "map[" or "[map[") into
	// table mode output.
	assert.NotContains(t, out, "map[")
}

// TestAnomaliesDimensionsProviderFlagPresent asserts --provider adds a
// `provider` query param.
func TestAnomaliesDimensionsProviderFlagPresent(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, dimensionsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--provider", "anthropic"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, gotQuery, "provider=anthropic")
}

// TestAnomaliesDimensionsProviderFlagAbsent asserts omitting --provider
// sends no `provider` query param.
func TestAnomaliesDimensionsProviderFlagAbsent(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, dimensionsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, gotQuery, "provider")
}

// TestAnomaliesDimensionsJSON asserts JSON mode passes the raw nested
// response through unchanged.
func TestAnomaliesDimensionsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, dimensionsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDimensionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	providers, ok := parsed["providers"].([]interface{})
	require.True(t, ok)
	assert.Len(t, providers, 2)
	apiKeys, ok := parsed["apiKeys"].([]interface{})
	require.True(t, ok)
	assert.Len(t, apiKeys, 1)
}
