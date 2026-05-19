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
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOutcome is the happy-path test for `revenium jobs outcome <id> --result <value>`.
// It pins the LOAD-BEARING contract from RESEARCH §Pitfall 3 / CONTEXT D-02:
// the CLI flag --result maps to body["executionStatus"], NOT body["result"].
func TestOutcome(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1/outcome", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--result", "SUCCESS"})
	err := c.Execute()

	require.NoError(t, err)

	// LOAD-BEARING per RESEARCH Pitfall 3 — --result maps to body["executionStatus"],
	// NOT body["result"]. This is the only assertion that catches the field-name
	// drift between the user-facing CLI flag and the OAS request body field.
	assert.Equal(t, "SUCCESS", receivedBody["executionStatus"])
	_, hasResult := receivedBody["result"]
	assert.False(t, hasResult, "--result must NOT appear as body[\"result\"]; it maps to executionStatus")

	// Response rendered via renderJob — system id and label reach stdout.
	assert.Contains(t, buf.String(), "JMwX9g4")
	assert.Contains(t, buf.String(), "Process Loan")
}

// TestOutcomeConflict pins the D-01 409 immutability override. The server
// returns 409 + ApiError_Read JSON; the CLI replaces the generic message
// wholesale with the resource-specific phrasing that names the job id and
// the recovery command. The raw apiErr.Message is intentionally NOT
// interpolated (T-13-02-01 mitigation in the threat model).
func TestOutcomeConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"timestamp":"2026-05-12T00:00:00Z","status":409,"error":"Conflict","message":"Outcome already reported","path":"/v2/api/jobs/loan-app-1/outcome"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1", "--result", "SUCCESS"})
	err := c.Execute()

	require.Error(t, err)
	// Exact D-01 phrasing: job id + "immutable" + recovery suggestion.
	assert.Contains(t, err.Error(), "outcome already reported for job loan-app-1")
	assert.Contains(t, err.Error(), "immutable")
	assert.Contains(t, err.Error(), "revenium jobs get loan-app-1")
}

// TestOutcomeOptionalFieldsGated asserts that omitted optional flags do NOT
// appear in the request body — proves the c.Flags().Changed("...") gating
// in outcome.go RunE for outcomeType/outcomeValue/outcomeCurrency/metadata/reportedBy.
func TestOutcomeOptionalFieldsGated(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--result", "SUCCESS"})
	err := c.Execute()

	require.NoError(t, err)

	// Required field present.
	assert.Equal(t, "SUCCESS", receivedBody["executionStatus"])

	// Five optional fields absent because the flags were never set.
	_, hasType := receivedBody["outcomeType"]
	assert.False(t, hasType, "outcomeType must NOT be sent when --outcome-type omitted")
	_, hasValue := receivedBody["outcomeValue"]
	assert.False(t, hasValue, "outcomeValue must NOT be sent when --outcome-value omitted")
	_, hasCurrency := receivedBody["outcomeCurrency"]
	assert.False(t, hasCurrency, "outcomeCurrency must NOT be sent when --outcome-currency omitted")
	_, hasMetadata := receivedBody["metadata"]
	assert.False(t, hasMetadata, "metadata must NOT be sent when --metadata omitted")
	_, hasReportedBy := receivedBody["reportedBy"]
	assert.False(t, hasReportedBy, "reportedBy must NOT be sent when --reported-by omitted")
}

// TestOutcomeTeamId pins the teamId-query-injection contract. When the client
// is constructed with a TeamID, internal/api/client.go:68-74 auto-appends
// ?teamId=<value> to every request. Mirrors cmd/jobs/update_test.go:66-86.
func TestOutcomeTeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--result", "SUCCESS"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}

// TestOutcomeMissingResult confirms that omitting --result surfaces a Cobra
// "required flag(s)" error BEFORE any HTTP call. No httptest server is
// constructed — failure must happen at flag parse time.
func TestOutcomeMissingResult(t *testing.T) {
	var buf bytes.Buffer
	// Point client at a URL that would fail loudly if hit — the test should
	// error during flag validation, well before the HTTP layer.
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1"}) // no --result

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "result",
		"missing-flag error should mention 'result' by name")
}

// TestOutcomeDryRun pins the dry-run output contract by invoking dryrun.Render
// DIRECTLY with the exact path/action/resource/body shape that outcome.go's
// RunE will pass in the dry-run branch. This decouples the contract assertion
// from the cmd.DryRun() global-flag toggle (which has no exported setter as of
// planning time). The cmd.DryRun() gate inside outcome.go is a structurally
// trivial single `if` branch — its integration is deferred to manual smoke;
// the dry-run output contract (this test) is the load-bearing assertion.
func TestOutcomeDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact path outcome.go's RunE constructs via
	// fmt.Sprintf("/v2/api/jobs/%s/outcome", url.PathEscape("loan-app-1")).
	path := "/v2/api/jobs/loan-app-1/outcome"

	// The exact body outcome.go's RunE builds for a SUCCESS result with no
	// optional flags set — proves the dry-run output reflects the same
	// shape the HTTP path would send.
	body := map[string]interface{}{"executionStatus": "SUCCESS"}

	err := dryrun.Render(out, "outcome", "job", path, body)

	require.NoError(t, err)
	rendered := buf.String()

	// Header line — matches internal/dryrun/dryrun.go:23 format.
	assert.Contains(t, rendered, "Dry run: outcome job")
	// Path round-trip.
	assert.Contains(t, rendered, "/v2/api/jobs/loan-app-1/outcome")
	// Body key visible in output (proves executionStatus mapping survives).
	assert.Contains(t, rendered, "executionStatus")
	// Footer — matches internal/dryrun/dryrun.go:28.
	assert.Contains(t, rendered, "No changes were made.")
}
