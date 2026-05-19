package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateJob is the LOAD-BEARING test for JOBS-04. It asserts that
// `revenium jobs update <agenticJobId> --name X` issues a single HTTP PATCH
// request (not PUT, not the GET+merge+PUT helper) to /v2/api/jobs/<id>
// with only the changed fields in the body, and that the rendered output
// contains the returned label.
//
// This mirrors cmd/models/update_test.go:19-48 — NOT the anomalies update
// test (which handles GET+PUT for the merge helper, the wrong pattern for
// jobs per D-02 / D-23).
func TestUpdateJob(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		assert.Equal(t, "/v2/api/jobs/loan-app-1", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan v2","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--name", "Process Loan v2"})
	err := c.Execute()

	require.NoError(t, err)
	// THE load-bearing JOBS-04 assertion per D-02 / RESEARCH §"Test Patterns" §A.
	// Mirrors cmd/models/update_test.go:42 verbatim with the jobs path.
	assert.Equal(t, "PATCH", receivedMethod, "update must use PATCH, not PUT")
	// Only the changed field must be in the body — proves Flags().Changed gating works.
	assert.Equal(t, "Process Loan v2", receivedBody["name"])
	// Phase 12 only exposes --name; assert the body contains exactly that one field.
	assert.Equal(t, 1, len(receivedBody), "body must contain only the changed field (name)")
	// Output assertion: renderJob() output reaches stdout with the returned label.
	assert.Contains(t, buf.String(), "Process Loan v2")
}

// TestUpdateJobTeamId asserts that the teamId query parameter is auto-injected
// by Client.Do (internal/api/client.go:68-74) when the client is constructed
// with a TeamID. Mirrors the models update teamId test (lines 50-70 of the
// models package's update test file).
func TestUpdateJobTeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan v2","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--name", "Process Loan v2"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}

// TestUpdateJobNoFields asserts that invoking `revenium jobs update <id>` with
// no flags returns the exact error string "no fields specified to update"
// BEFORE any HTTP call (D-03). No httptest server is created — if the command
// makes any HTTP call, the test would surface a different error (network
// connection refused or similar).
func TestUpdateJobNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}
