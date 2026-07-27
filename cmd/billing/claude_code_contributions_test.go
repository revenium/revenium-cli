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
	c.SetOut(&buf)
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
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	contributions, ok := parsed["contributions"].([]interface{})
	require.True(t, ok)
	assert.Len(t, contributions, 1)
}
