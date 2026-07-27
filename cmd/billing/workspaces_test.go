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

// workspacesFixture has BOTH an _embedded array AND a top-level totalCost
// aggregate — the D-04 load-bearing shape.
const workspacesFixture = `{
	"_embedded": {
		"objectList": [
			{"workspace": "Default", "cost": 30.0}
		]
	},
	"totalCost": 60.0,
	"costByProvider": {"openai": 60.0},
	"creditsAppliedByProvider": {"openai": 0},
	"lastRefreshDate": "2026-07-19T00:00:00Z"
}`

// TestBillingWorkspaces asserts the correct path is hit and the _embedded
// rows render in table mode. Note: this is the billing sub-verb (billing
// workspaces), distinct from the top-level cmd/workspaces package.
func TestBillingWorkspaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/billing/workspaces", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, workspacesFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newWorkspacesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Default")
}

// TestBillingWorkspacesJSONPreservesAggregates is the load-bearing D-04
// assertion: in --json mode, the output must STILL contain totalCost.
func TestBillingWorkspacesJSONPreservesAggregates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, workspacesFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newWorkspacesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, 60.0, parsed["totalCost"])
}

// TestBillingWorkspacesEmptyTableMode asserts the canonical empty-state
// phrase fires in table mode.
func TestBillingWorkspacesEmptyTableMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded": {"objectList": []}, "totalCost": 0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newWorkspacesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No billing workspace data found.")
}
