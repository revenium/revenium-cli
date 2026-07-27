package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureSquadExecution is a real-schema EntityModelSquadExecutionResource_Read
// fixture (RESEARCH: id, squadName, agentCount, traceCount, duration,
// totalCost, status) — never the fictional transactionId/name/executions
// fields the previous implementation read.
const fixtureSquadExecution = `[{
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
}]`

func TestSquadMetrics(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		assert.Empty(t, r.URL.Query().Get("startDate"))
		assert.Empty(t, r.URL.Query().Get("endDate"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecution)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	squadsPeriodFlag = ""

	c := newSquadsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads", gotPath)
	out := buf.String()
	assert.Contains(t, out, "squad-loan-proc-12345")
	assert.Contains(t, out, "Loan Processing")
	assert.Contains(t, out, "5") // agentCount
	assert.Contains(t, out, "3") // traceCount
	assert.Contains(t, out, "SUCCESS")
}

func TestSquadMetricsWithPeriod(t *testing.T) {
	var gotQuery string
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("period")
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecution)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	squadsPeriodFlag = "SEVEN_DAYS"
	defer func() { squadsPeriodFlag = "" }()

	c := newSquadsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--period", "SEVEN_DAYS"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "SEVEN_DAYS", gotQuery)
	assert.NotContains(t, gotRawQuery, "startDate")
	assert.NotContains(t, gotRawQuery, "endDate")
}

func TestSquadMetricsInvalidPeriod(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecution)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	squadsPeriodFlag = ""

	c := newSquadsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--period", "BOGUS"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	assert.Equal(t, 0, requestCount, "invalid --period must fail before any HTTP request")
}

func TestSquadMetricsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	squadsPeriodFlag = ""

	c := newSquadsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No squad metrics found.")
}

func TestSquadMetricsEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	squadsPeriodFlag = ""

	c := newSquadsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestSquadMetricsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecution)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	squadsPeriodFlag = ""

	c := newSquadsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "squad-loan-proc-12345", result[0]["id"])
	assert.Equal(t, "Loan Processing", result[0]["squadName"])
}

func TestSquadMetricsDeprecationNote(t *testing.T) {
	var buf bytes.Buffer
	c := newSquadsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--help"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "DEPRECATED")
	assert.Contains(t, out, "squads executions")
}
