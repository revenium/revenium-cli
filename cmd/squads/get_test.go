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

// fixtureSquadDetail is a real-schema SquadDetailResource_Read fixture
// (RESEARCH: id, squadName, agentCount, traceCount, transactionCount,
// errorCount, duration, totalCost, status).
const fixtureSquadDetail = `{
	"id": "squad-loan-proc-12345",
	"resourceType": "squad.detail",
	"label": "Loan Processing",
	"squadName": "Loan Processing",
	"startTime": "2025-09-01T00:00:00Z",
	"endTime": "2025-09-01T00:01:30Z",
	"duration": 90000,
	"agentCount": 5,
	"traceCount": 3,
	"transactionCount": 15,
	"errorCount": 2,
	"totalCost": 0.0524,
	"status": "SUCCESS"
}`

func TestSquadsGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads/squad-loan-proc-12345", gotPath)
	out := buf.String()
	assert.Contains(t, out, "Loan Processing")
	assert.Contains(t, out, "15") // transactionCount
	assert.Contains(t, out, "2")  // errorCount
	assert.Contains(t, out, "SUCCESS")
}

func TestSquadsGetRejectsPathTraversal(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"../etc"})
	err := c.Execute()

	require.Error(t, err)
	assert.Equal(t, 0, requestCount, "path traversal must be rejected before any HTTP request")
}

func TestSquadsGetPathEscape(t *testing.T) {
	var gotEscapedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is auto-decoded by net/http; EscapedPath() reflects what
		// was actually sent on the wire (the assertion we care about).
		gotEscapedPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad with space"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads/squad%20with%20space", gotEscapedPath, "id with a space must be escaped, not sent raw")
}

func TestSquadsGetWithPeriod(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = "SEVEN_DAYS"
	defer func() { periodFlag = "" }()

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "SEVEN_DAYS", gotQuery)
}

func TestSquadsGetNoPeriod(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Empty(t, gotRawQuery)
}
