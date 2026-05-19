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

// TestCreateJob verifies that `revenium jobs create --agentic-job-id X --name Y`
// issues a POST to /v2/api/jobs with the expected body, and that optional fields
// not supplied are omitted from the body (proves Flags().Changed gating per
// RESEARCH A3).
func TestCreateJob(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/jobs", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"JMwX9g4","agenticJobId":"loan-app-1","label":"Process Loan","executionStatus":""}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--agentic-job-id", "loan-app-1", "--name", "Process Loan"})
	err := c.Execute()

	require.NoError(t, err)

	// Body must contain the two flags the user supplied.
	assert.Equal(t, "loan-app-1", receivedBody["agenticJobId"])
	assert.Equal(t, "Process Loan", receivedBody["name"])

	// Optional fields NOT supplied on the command line must not appear in the
	// body. This proves the c.Flags().Changed("...") gating in create.go.
	_, hasType := receivedBody["type"]
	assert.False(t, hasType, "type should not be sent when --type not provided")
	_, hasVersion := receivedBody["version"]
	assert.False(t, hasVersion, "version should not be sent when --version not provided")
	_, hasEnvironment := receivedBody["environment"]
	assert.False(t, hasEnvironment, "environment should not be sent when --environment not provided")

	// Response is rendered via renderJob — output should contain the system id.
	assert.Contains(t, buf.String(), "JMwX9g4")
	assert.Contains(t, buf.String(), "Process Loan")
}

// TestCreateJobMissingRequired confirms that omitting --agentic-job-id surfaces
// a Cobra "required flag(s)" error before any HTTP call is made. Proves
// MarkFlagRequired("agentic-job-id") is wired correctly (D-04).
func TestCreateJobMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	// Use a URL that would fail loudly if accidentally hit — the test should
	// error during flag validation, well before the HTTP layer.
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{}) // no flags — agentic-job-id missing

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "agentic-job-id",
		"missing-flag error should mention agentic-job-id by name")
}
