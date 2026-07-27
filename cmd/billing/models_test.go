package billing

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

// modelsFixture has BOTH an _embedded array AND a top-level totalCost
// aggregate — the D-04 load-bearing shape.
const modelsFixture = `{
	"_embedded": {
		"objectList": [
			{"model": "gpt-4", "provider": "openai", "cost": 15.0}
		]
	},
	"totalCost": 200.0,
	"costByProvider": {"openai": 200.0},
	"creditsAppliedByProvider": {"openai": 0},
	"lastRefreshDate": "2026-07-19T00:00:00Z"
}`

// TestBillingModels asserts the correct path is hit and the _embedded rows
// render in table mode.
func TestBillingModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/billing/models", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, modelsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newModelsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "gpt-4")
	assert.Contains(t, out, "openai")
}

// TestBillingModelsJSONPreservesAggregates is the load-bearing D-04
// assertion: in --json mode, the output must STILL contain totalCost.
func TestBillingModelsJSONPreservesAggregates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, modelsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newModelsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, 200.0, parsed["totalCost"])
}

// TestBillingModelsEmptyTableMode asserts the canonical empty-state phrase
// fires in table mode.
func TestBillingModelsEmptyTableMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded": {"objectList": []}, "totalCost": 0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newModelsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No billing model data found.")
}
