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

// dailyFixture has BOTH an _embedded array AND a top-level totalCost
// aggregate — the D-04 load-bearing shape.
const dailyFixture = `{
	"_embedded": {
		"objectList": [
			{"date": "2026-07-19", "cost": 5.25, "creditsApplied": 1.0}
		]
	},
	"totalCost": 88.5,
	"costByProvider": {"anthropic": 88.5},
	"creditsAppliedByProvider": {"anthropic": 1.0},
	"lastRefreshDate": "2026-07-19T00:00:00Z"
}`

// TestBillingDaily asserts the correct path is hit and the _embedded rows
// render in table mode.
func TestBillingDaily(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/billing/daily", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, dailyFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDailyCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "2026-07-19")
}

// TestBillingDailyJSONPreservesAggregates is the load-bearing D-04
// assertion: in --json mode, the output must STILL contain totalCost.
func TestBillingDailyJSONPreservesAggregates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, dailyFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDailyCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, 88.5, parsed["totalCost"])
}

// TestBillingDailyEmptyTableMode asserts the canonical empty-state phrase
// fires in table mode.
func TestBillingDailyEmptyTableMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded": {"objectList": []}, "totalCost": 0}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDailyCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No billing daily data found.")
}
