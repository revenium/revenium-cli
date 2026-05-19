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
