package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureSquadExecution returns a real-schema EntityModelSquadExecutionResource_Read
// fixture matching the live platform API (RESEARCH: id, resourceType, label,
// squadName, startTime, endTime, duration, agentCount, traceCount, totalCost,
// status) — never the fictional transactionId/name/executions fields.
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

func TestValidatePeriod(t *testing.T) {
	assert.NoError(t, ValidatePeriod(""))

	validValues := []string{
		"HOUR", "EIGHT_HOURS", "TWENTY_FOUR_HOURS", "SEVEN_DAYS",
		"THIRTY_DAYS", "NINETY_DAYS", "SIX_MONTHS", "TWELVE_MONTHS",
	}
	for _, v := range validValues {
		assert.NoError(t, ValidatePeriod(v), "expected %q to be a valid period", v)
	}

	err := ValidatePeriod("BOGUS")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	for _, v := range validValues {
		assert.Contains(t, err.Error(), v, "error message should name valid value %q", v)
	}
}

func TestFetchSquadExecutionsNoPeriod(t *testing.T) {
	var gotPath string
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotRawQuery = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecution)
	}))
	defer srv.Close()

	APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	Output = output.NewWithWriter(nil, nil, false, false)

	metrics, err := FetchSquadExecutions(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads", gotPath)
	assert.Empty(t, gotRawQuery, "no period query param expected when period is empty")
	require.Len(t, metrics, 1)
}

func TestFetchSquadExecutionsWithPeriod(t *testing.T) {
	var gotQuery string
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("period")
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadExecution)
	}))
	defer srv.Close()

	APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	Output = output.NewWithWriter(nil, nil, false, false)

	metrics, err := FetchSquadExecutions(context.Background(), "SEVEN_DAYS")
	require.NoError(t, err)
	assert.Equal(t, "SEVEN_DAYS", gotQuery)
	assert.NotContains(t, gotRawQuery, "startDate")
	assert.NotContains(t, gotRawQuery, "endDate")
	require.Len(t, metrics, 1)
}

func TestToSquadExecutionRows(t *testing.T) {
	metrics := []map[string]interface{}{
		{
			"id":         "squad-loan-proc-12345",
			"squadName":  "Loan Processing",
			"agentCount": float64(5),
			"traceCount": float64(3),
			"duration":   float64(90000),
			"totalCost":  0.0524,
			"status":     "SUCCESS",
		},
	}
	rows := ToSquadExecutionRows(metrics)
	require.Len(t, rows, 1)
	row := rows[0]
	assert.Contains(t, row, "squad-loan-proc-12345")
	assert.Contains(t, row, "Loan Processing")
	assert.Contains(t, row, "SUCCESS")
	// Cost column rendered via formatCost
	found := false
	for _, cell := range row {
		if cell == formatCost(0.0524) {
			found = true
		}
	}
	assert.True(t, found, "expected a cell rendered via formatCost")
}
