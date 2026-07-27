package squads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// fixtureSquadEntity is a real-schema EntityModelSquadResource_Read fixture
// (RESEARCH: hash-encoded id, label, description, source, executionCount,
// totalCost, agentCount, traceCount, successCount/partialCount/failedCount).
const fixtureSquadEntity = `[{
	"id": "JMwX9g4",
	"resourceType": "squad",
	"label": "Loan Processing",
	"description": "Handles end-to-end loan application processing",
	"source": "TELEMETRY",
	"executionCount": 42,
	"totalCost": 125.5,
	"agentCount": 8,
	"traceCount": 156,
	"firstExecutionTime": "2025-09-01T00:00:00Z",
	"lastExecutionTime": "2025-09-15T14:30:00Z",
	"successCount": 38,
	"partialCount": 3,
	"failedCount": 1
}]`

// newPeriodValidatingRoot builds a throwaway parent command mirroring
// squads.Cmd's PersistentPreRunE (period validation only, no root config
// init — tests set cmd.APIClient/cmd.Output directly, matching every other
// package's test pattern) so --period validation can be exercised against
// sub without mutating the real package-level Cmd singleton between tests.
func newPeriodValidatingRoot(sub *cobra.Command) *cobra.Command {
	root := &cobra.Command{
		Use:           "squads",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			return cmd.ValidatePeriod(periodFlag)
		},
	}
	root.PersistentFlags().StringVar(&periodFlag, "period", "", "period")
	root.AddCommand(sub)
	return root
}

func TestSquadsList(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadEntity)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads/entities", gotPath)
	out := buf.String()
	assert.Contains(t, out, "Loan Processing")
	assert.Contains(t, out, "42")  // executionCount
	assert.Contains(t, out, "8")   // agentCount
	assert.Contains(t, out, "156") // traceCount
}

func TestSquadsListWithPeriod(t *testing.T) {
	var gotQuery string
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("period")
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadEntity)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = "SEVEN_DAYS"
	defer func() { periodFlag = "" }()

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "SEVEN_DAYS", gotQuery)
	assert.NotContains(t, gotRawQuery, "startDate")
	assert.NotContains(t, gotRawQuery, "endDate")
}

func TestSquadsListInvalidPeriod(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadEntity)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	root := newPeriodValidatingRoot(newListCmd())
	root.SetOut(&buf)
	root.SetArgs([]string{"list", "--period", "BOGUS"})
	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	assert.Equal(t, 0, requestCount, "invalid --period must fail before any HTTP request")
}

func TestSquadsListTeamId(t *testing.T) {
	var gotTeamID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTeamID = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadEntity)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", gotTeamID, "teamId must be auto-injected, no --team-id flag needed")
}

func TestSquadsListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No squads found.")
}

func TestSquadsListEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}
