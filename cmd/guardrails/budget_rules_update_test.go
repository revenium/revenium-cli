package guardrails

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestBudgetRulesUpdate is the LOAD-BEARING test for GRDR-04 / D-08 /
// CF-12-20. It asserts that `revenium guardrails budget-rules update <id>
// --name X` issues a single HTTP PATCH (literal method string — NOT PUT,
// NOT the GET+merge+PUT helper) to /v2/api/ai/cost-controls/<id> with only
// the changed fields in the body.
//
// Mirrors cmd/jobs/update_test.go TestUpdateJob — same load-bearing
// assertion that locks the PATCH vs PUT contract.
func TestBudgetRulesUpdate(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// THE load-bearing assertion — D-08 / GRDR-04 contract lock.
		assert.Equal(t, "PATCH", r.Method, "update must use PATCH, not PUT")
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &received))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"Q3 OpenAI Budget v2","enabled":true,"_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs", "--name", "Q3 OpenAI Budget v2"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Q3 OpenAI Budget v2", received["name"])
	// Only the explicitly-changed field reaches the body — proves D-07 / D-08
	// Flags().Changed gating works and DoUpdate-style full-merge is NOT used.
	assert.Equal(t, 1, len(received), "body must contain only the changed field (name)")
	// Rendered output reaches stdout with the returned id+name.
	assert.Contains(t, buf.String(), "jR2kmLs")
	assert.Contains(t, buf.String(), "Q3 OpenAI Budget v2")
}

// TestBudgetRulesUpdateNoFields asserts that invoking update with no flags
// returns the verbatim D-09 error string "no fields specified to update"
// BEFORE any HTTP call. No httptest server is created — if the command
// were to issue an HTTP request, it would surface a different error
// (connection refused to 127.0.0.1:1).
func TestBudgetRulesUpdateNoFields(t *testing.T) {
	var buf bytes.Buffer
	// Unreachable URL — if the test fails to fail-fast on the empty-body
	// branch, the client would attempt a connection here and surface a
	// connection-refused error rather than the verbatim D-09 string.
	cmd.APIClient = api.NewClient("http://127.0.0.1:1", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"jR2kmLs"})
	err := c.Execute()

	require.Error(t, err)
	// D-09 verbatim string match.
	assert.EqualError(t, err, "no fields specified to update")
}

// TestBudgetRulesUpdateFilters asserts that --filter dim:op:val flags compose
// into a PATCH body containing exactly {"filters":[...],"notificationChannelIds":[...]}
// — no extra keys leak. Per PLAN D-04 (locked), PATCH REPLACES the filters and
// notificationChannelIds arrays wholesale; the proof is that the on-wire body
// contains the exact slice the user supplied (no merge with existing values).
// The "PATCH" method assertion stays load-bearing (D-08).
func TestBudgetRulesUpdateFilters(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// D-08 load-bearing: PATCH method must be preserved.
		assert.Equal(t, "PATCH", r.Method, "update must use PATCH, not PUT")
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &received))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"X","enabled":true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"jR2kmLs",
		"--filter", "MODEL:IS:gpt-4",
		"--filter", "PROVIDER:IS:openai",
		"--notification-channel-id", "chan-9",
	})
	require.NoError(t, c.Execute())

	// PLAN D-04 locked: PATCH replaces the filters array wholesale per PATCH-replace
	// semantics. The on-wire assertion that the body contains the exact slice
	// the user supplied (not a merge) is the proof.
	assert.Equal(t, 2, len(received), "body must contain only the two changed fields (filters + notificationChannelIds)")

	rawFilters, ok := received["filters"].([]interface{})
	require.True(t, ok, "filters must be a JSON array")
	require.Len(t, rawFilters, 2)
	f0 := rawFilters[0].(map[string]interface{})
	assert.Equal(t, "MODEL", f0["dimension"])
	assert.Equal(t, "IS", f0["operator"])
	assert.Equal(t, "gpt-4", f0["value"])
	f1 := rawFilters[1].(map[string]interface{})
	assert.Equal(t, "PROVIDER", f1["dimension"])

	rawChans, ok := received["notificationChannelIds"].([]interface{})
	require.True(t, ok, "notificationChannelIds must be a JSON array")
	require.Len(t, rawChans, 1)
	assert.Equal(t, "chan-9", rawChans[0])
}

// TestBudgetRulesUpdateFiltersJSONOnly proves the --filters-json escape hatch
// works on update too, including the colon-in-value edge case from D-05.
func TestBudgetRulesUpdateFiltersJSONOnly(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &received))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"X","enabled":true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"jR2kmLs",
		"--filters-json", `[{"dimension":"AGENT","operator":"IS","value":"a:b:c"}]`,
	})
	require.NoError(t, c.Execute())

	rawFilters, ok := received["filters"].([]interface{})
	require.True(t, ok)
	require.Len(t, rawFilters, 1)
	f0 := rawFilters[0].(map[string]interface{})
	assert.Equal(t, "AGENT", f0["dimension"])
	assert.Equal(t, "IS", f0["operator"])
	// Colon-laden value passes through verbatim (PLAN D-05 locked).
	assert.Equal(t, "a:b:c", f0["value"])
}

// TestBudgetRulesUpdateFiltersConflict asserts the D-01 mutual-exclusion check
// fires BEFORE any HTTP call on update — same contract as create. Uses the
// unreachable-URL pattern from TestBudgetRulesUpdateNoFields so a connection
// attempt would surface a connection-refused error instead of the conflict
// message.
func TestBudgetRulesUpdateFiltersConflict(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://127.0.0.1:1", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{
		"jR2kmLs",
		"--filter", "X:Y:Z",
		"--filters-json", `[{"dimension":"X","operator":"IS","value":"y"}]`,
	})

	err := c.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--filter")
	assert.Contains(t, err.Error(), "--filters-json")
}

// TestBudgetRulesUpdateFiltersReplaceSemantics is a documentation-style guard
// that pins the D-04 wire contract: when the user passes --filter, the PATCH
// body contains EXACTLY the user-supplied slice (no merge with whatever the
// server already has). The single-element body assertion plus the exact slice
// equality is the proof.
//
// PLAN D-04 locked: PATCH replaces the filters array wholesale per
// PATCH-replace semantics. There is no per-element add/remove API.
func TestBudgetRulesUpdateFiltersReplaceSemantics(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &received))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"X","enabled":true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"jR2kmLs",
		"--filter", "MODEL:IS:only-this-one",
	})
	require.NoError(t, c.Execute())

	// Exactly one key in body (filters), proving merge logic is NOT used.
	assert.Equal(t, 1, len(received), "PATCH body must contain only the explicitly changed field — no GET+merge+PUT")
	rawFilters, ok := received["filters"].([]interface{})
	require.True(t, ok)
	require.Len(t, rawFilters, 1, "filters array must contain exactly what the user supplied — no merge")
	f0 := rawFilters[0].(map[string]interface{})
	assert.Equal(t, "only-this-one", f0["value"])
}

// TestBudgetRulesUpdateTeamId asserts that the teamId query parameter is
// auto-injected by Client.Do when the client is constructed with a TeamID.
// Mirrors cmd/jobs/update_test.go TestUpdateJobTeamId.
func TestBudgetRulesUpdateTeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"X","enabled":true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs", "--name", "X"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}
