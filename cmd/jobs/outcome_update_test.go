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

// TestOutcomeUpdate is the happy-path test for
// `revenium jobs outcome-update <id> --reason <value>`. It pins the
// LOAD-BEARING contract that the request method is PATCH (not POST) to a
// path ending `/outcome` — the same path as the existing POST `outcome`
// command but a distinct verb (RES-07).
func TestOutcomeUpdate(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "corrected value"})
	err := c.Execute()

	require.NoError(t, err)

	assert.Equal(t, "PATCH", receivedMethod, "outcome-update must use PATCH, not POST")
	assert.Equal(t, "/v2/api/jobs/loan-app-1/outcome", receivedPath)
	assert.Equal(t, "corrected value", receivedBody["reason"])

	assert.Contains(t, buf.String(), "JMwX9g4")
	assert.Contains(t, buf.String(), "Process Loan")
}

// TestOutcomeUpdateOptionalFieldsGated asserts that omitted optional flags do
// NOT appear in the PATCH body, mirroring outcome_test.go's gating pattern.
func TestOutcomeUpdateOptionalFieldsGated(t *testing.T) {
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

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "corrected value"})
	err := c.Execute()

	require.NoError(t, err)

	assert.Equal(t, "corrected value", receivedBody["reason"])

	_, hasExecutionStatus := receivedBody["executionStatus"]
	assert.False(t, hasExecutionStatus, "executionStatus must NOT be sent when --execution-status omitted")
	_, hasType := receivedBody["outcomeType"]
	assert.False(t, hasType, "outcomeType must NOT be sent when --outcome-type omitted")
	_, hasValue := receivedBody["outcomeValue"]
	assert.False(t, hasValue, "outcomeValue must NOT be sent when --outcome-value omitted")
	_, hasCurrency := receivedBody["outcomeCurrency"]
	assert.False(t, hasCurrency, "outcomeCurrency must NOT be sent when --outcome-currency omitted")
	_, hasMetadata := receivedBody["metadata"]
	assert.False(t, hasMetadata, "metadata must NOT be sent when --metadata omitted")
}

// TestOutcomeUpdateMissingReason confirms that omitting --reason surfaces a
// Cobra "required flag(s)" error BEFORE any HTTP call.
func TestOutcomeUpdateMissingReason(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1"}) // no --reason

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reason",
		"missing-flag error should mention 'reason' by name")
}

// TestOutcomeUpdateConflict pins the DISTINCT 409 copy (Pitfall 2 regression
// guard): outcome-update's 409 means "concurrent update conflict, retry" —
// NOT "already reported... immutable" like the existing outcome command's
// 409. The rendered error must contain "concurrent" and must NOT contain
// "immutable".
func TestOutcomeUpdateConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"timestamp":"2026-05-12T00:00:00Z","status":409,"error":"Conflict","message":"Concurrent update","path":"/v2/api/jobs/loan-app-1/outcome"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "retry"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "concurrent")
	assert.NotContains(t, err.Error(), "immutable",
		"outcome-update's 409 must NOT reuse outcome.go's 'immutable' copy (Pitfall 2)")
	assert.Contains(t, err.Error(), "loan-app-1")
}

// TestOutcomeUpdateUnprocessable pins the 422 copy: "no outcome has been
// reported yet" — distinct from the 409 copy above and from outcome.go's
// 409 entirely.
func TestOutcomeUpdateUnprocessable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"timestamp":"2026-05-12T00:00:00Z","status":422,"error":"Unprocessable Entity","message":"No outcome reported","path":"/v2/api/jobs/loan-app-1/outcome"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"loan-app-1", "--reason", "first update"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no outcome has been reported yet")
	assert.Contains(t, err.Error(), "revenium jobs outcome loan-app-1")
}

// TestOutcomeUpdateDryRun pins the dry-run output contract by invoking
// dryrun.Render DIRECTLY with the exact path/action/resource/body shape that
// outcome_update.go's RunE will pass in the dry-run branch — mirroring
// outcome_test.go's TestOutcomeDryRun pattern. This decouples the contract
// assertion from the cmd.DryRun() global-flag toggle.
func TestOutcomeUpdateDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact path outcome_update.go's RunE constructs via
	// fmt.Sprintf("/v2/api/jobs/%s/outcome", url.PathEscape("loan-app-1")).
	path := "/v2/api/jobs/loan-app-1/outcome"

	// The exact body outcome_update.go's RunE builds for a reason-only call
	// with no optional flags set.
	body := map[string]interface{}{"reason": "preview"}

	err := dryrun.Render(out, "outcome-update", "job", path, body)

	require.NoError(t, err)
	rendered := buf.String()

	assert.Contains(t, rendered, "Dry run: outcome-update job")
	assert.Contains(t, rendered, "/v2/api/jobs/loan-app-1/outcome")
	assert.Contains(t, rendered, "reason")
	assert.Contains(t, rendered, "No changes were made.")
}
