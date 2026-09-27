package billing

import (
	"bytes"
	"encoding/json"
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

const claudeCodeContributionsFixture = `{
	"contributions": [
		{"userEmail": "jane@example.com", "count": 5}
	],
	"totalContributions": 5
}`

// TestBillingClaudeCode asserts the exact GET-only request path (read-only
// getClaudeCodeContributions, NOT the mutating sync POST — D-02) and that
// the summary renders in table mode.
func TestBillingClaudeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v2/api/billing/users/claude-code-contributions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, claudeCodeContributionsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newClaudeCodeContributionsCmd()
	var errBuf bytes.Buffer
	c.SetOut(&buf)
	// SetErr keeps the BILL-07 deprecation notice out of `go test` output; the
	// notice itself is asserted in TestClaudeCodeContributionsDeprecationWarning.
	c.SetErr(&errBuf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1")
	assert.Contains(t, out, "5")
}

// TestBillingClaudeCodeJSON asserts JSON mode passes the object through
// unchanged.
func TestBillingClaudeCodeJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, claudeCodeContributionsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newClaudeCodeContributionsCmd()
	var errBuf bytes.Buffer
	c.SetOut(&buf)
	// SetErr keeps the BILL-07 deprecation notice out of `go test` output; the
	// notice itself is asserted in TestClaudeCodeContributionsDeprecationWarning.
	c.SetErr(&errBuf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	contributions, ok := parsed["contributions"].([]interface{})
	require.True(t, ok)
	assert.Len(t, contributions, 1)
}

// TestClaudeCodeContributionsDeprecationWarning asserts all four halves of
// BILL-07's criterion for `billing claude-code-contributions`: the notice is
// present on the command's error writer, absent from stdout, stdout is still a
// single parseable JSON document, and the exit code is unchanged.
func TestClaudeCodeContributionsDeprecationWarning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, claudeCodeContributionsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// JSON mode ON so the test proves the strongest form of the claim.
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newClaudeCodeContributionsCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	// c.SetErr is load-bearing: no other cmd/billing test sets it, so without
	// this line ErrOrStderr() falls through to the process's real stderr, the
	// assertion below reads an empty string, and the test passes proving
	// nothing. Task 30-04-03 mutation-proves that this assertion is not vacuous.
	c.SetErr(&errBuf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err, "the deprecation notice must not change the exit code")
	assert.Contains(t, errBuf.String(), "deprecated")
	assert.Contains(t, errBuf.String(), "revenium billing vcs-prs")
	assert.NotContains(t, buf.String(), "deprecated",
		"the deprecation warning must never appear on stdout or JSON output")
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed),
		"stdout must still be a single parseable JSON document")
}

// TestDeprecationWarningPrecedesRequest asserts the notice is emitted BEFORE
// the request is built, so it still reaches the operator on the exact day the
// withdrawn endpoint starts failing. A 500-responding stub stands in for the
// 404 the withdrawal will produce; the relayed error is unchanged.
func TestDeprecationWarningPrecedesRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newClaudeCodeContributionsCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	// Load-bearing for the same reason as above.
	c.SetErr(&errBuf)
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.SetArgs([]string{})
	err := c.Execute()

	require.Error(t, err, "the relayed client error is unchanged by the deprecation notice")
	assert.Contains(t, errBuf.String(), "deprecated",
		"the notice must fire even when the withdrawn endpoint returns an error")
	assert.NotContains(t, buf.String(), "deprecated")
}
