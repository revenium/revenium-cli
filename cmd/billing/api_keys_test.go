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

// apiKeysFixture has BOTH an _embedded array AND a top-level totalCost
// aggregate — the D-04 load-bearing shape.
const apiKeysFixture = `{
	"_embedded": {
		"objectList": [
			{"id": "key-1", "provider": "openai", "cost": 12.5}
		]
	},
	"totalCost": 42.75,
	"costByProvider": {"openai": 42.75},
	"creditsAppliedByProvider": {"openai": 0},
	"lastRefreshDate": "2026-07-19T00:00:00Z"
}`

// TestBillingApiKeys asserts the correct path is hit and the _embedded rows
// render in table mode.
func TestBillingApiKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/billing/api-keys", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, apiKeysFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAPIKeysCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "key-1")
	assert.Contains(t, out, "openai")
}

// TestBillingApiKeysJSONPreservesAggregates is the load-bearing D-04
// assertion: in --json mode, the output must STILL contain totalCost even
// though the DoList path (which would discard it) is not used.
func TestBillingApiKeysJSONPreservesAggregates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, apiKeysFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newAPIKeysCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, 42.75, parsed["totalCost"])
	embedded, ok := parsed["_embedded"].(map[string]interface{})
	require.True(t, ok)
	items, ok := embedded["objectList"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 1)
}

// TestBillingApiKeysEmptyPreservesAggregates asserts an empty _embedded page
// still preserves aggregates in JSON mode.
func TestBillingApiKeysEmptyPreservesAggregates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded": {"objectList": []}, "totalCost": 0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newAPIKeysCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	_, hasTotalCost := parsed["totalCost"]
	assert.True(t, hasTotalCost)
}

// TestBillingApiKeysEmptyTableMode asserts the canonical empty-state
// phrase fires in table mode.
func TestBillingApiKeysEmptyTableMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded": {"objectList": []}, "totalCost": 0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAPIKeysCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No billing API-key data found.")
}
