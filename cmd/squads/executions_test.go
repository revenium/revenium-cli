package squads

import (
	"bytes"
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

// fixtureSquadExecutionsFlat is a real-schema EntityModelSquadExecutionResource_Read
// array fixture (RESEARCH: id, squadName, agentCount, traceCount, duration,
// totalCost, status) backing the flat GET /v2/api/squads endpoint.
const fixtureSquadExecutionsFlat = `[
	{
		"id": "squad-loan-proc-12345",
		"resourceType": "squadExecution",
		"label": "Loan Processing",
		"squadName": "Loan Processing",
		"startTime": "2025-09-01T00:00:00Z",
		"endTime": "2025-09-01T00:01:30Z",
		"duration": 90000,
		"agentCount": 5,
		"traceCount": 3,
		"totalCost": 0.0524,
		"status": "SUCCESS"
	}
]`

// fixtureSquadExecutionsGroup is the same schema, used for the within-group
// endpoint (GET /v2/api/squads/entities/{squadId}/executions), which also
// returns a plain array per RESEARCH.
const fixtureSquadExecutionsGroup = `[
	{
		"id": "squad-loan-proc-67890",
		"resourceType": "squadExecution",
		"label": "Loan Processing 2",
		"squadName": "Loan Processing 2",
		"startTime": "2025-09-02T00:00:00Z",
		"endTime": "2025-09-02T00:01:30Z",
		"duration": 45000,
		"agentCount": 2,
		"traceCount": 1,
		"totalCost": 0.0111,
		"status": "SUCCESS"
	}
]`

func TestSquadsExecutionsFlatNoArg(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecutionsFlat)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newExecutionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads", gotPath)
	assert.Contains(t, buf.String(), "Loan Processing")
}

func TestSquadsExecutionsWithinGroupArgHeldPerD12(t *testing.T) {
	// D-12: live validation (02-04 Task 2) found no squad-group data to
	// confirm the {groupId} identity space, so the with-arg branch is held
	// behind a clear error instead of making an HTTP call.
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecutionsGroup)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newExecutionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"JMwX9g4"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "held per D-12")
	assert.Equal(t, 0, requestCount, "D-12-held branch must not make an HTTP call")
}

func TestSquadsExecutionsRejectsPathTraversal(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecutionsGroup)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newExecutionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"../evil"})
	err := c.Execute()

	require.Error(t, err)
	assert.Equal(t, 0, requestCount, "path traversal must be rejected before any HTTP request")
}

func TestSquadsExecutionsRejectsTwoArgs(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecutionsGroup)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newExecutionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"group-a", "group-b"})
	err := c.Execute()

	require.Error(t, err)
	assert.Equal(t, 0, requestCount, "MaximumNArgs(1) must reject two args before any HTTP request")
}

func TestSquadsExecutionsFlatDelegatesToSharedHelper(t *testing.T) {
	// Proves the flat (no-arg) branch shares the exact fetch path with
	// `metrics squads` (D-04) by asserting it produces the same request
	// shape cmd.FetchSquadExecutions issues (path + no group segment).
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecutionsFlat)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newExecutionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads", gotPath, "flat branch must hit the same path as cmd.FetchSquadExecutions")
}

func TestSquadsExecutionsEmptyFlat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "[]")
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newExecutionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No executions found.")
}

func TestSquadsExecutionsHelpTextDocumentsBothBranchesAndD12Status(t *testing.T) {
	c := newExecutionsCmd()
	longText := c.Long
	assert.Contains(t, longText, "flat", "help text must explain the no-arg flat branch")
	assert.Contains(t, longText, "group", "help text must explain the with-arg group branch")
	assert.Contains(t, longText, "D-12", "help text must document the D-12 held status")
	assert.Contains(t, longText, "HELD", "help text must make the held-per-D-12 status obvious")
}
