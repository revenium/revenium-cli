package tools

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

// TestToolsLookup asserts the by-tool-id lookup hits the LOCKED
// /v2/api/tools/by-tool-id/{toolId} path (a path param, not a `?query=` string —
// RES-09) and renders the tool via the existing renderTool helper.
func TestToolsLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/tools/by-tool-id/my-tool", r.URL.Path)
		assert.Empty(t, r.URL.Query().Get("toolId"), "toolId must be a path param, not a query param")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"tool-1","name":"My Tool","toolType":"MCP_SERVER","toolProvider":"custom","enabled":true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--tool-id", "my-tool"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "tool-1")
	assert.Contains(t, out, "My Tool")
}

// TestToolsLookupJSON asserts --json mode emits parseable JSON.
func TestToolsLookupJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/tools/by-tool-id/my-tool", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"tool-1","name":"My Tool","toolType":"MCP_SERVER","toolProvider":"custom","enabled":true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--tool-id", "my-tool"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "tool-1", result["id"])
}

// TestToolsLookupMissingRequired asserts missing --tool-id yields a pre-HTTP
// required-flag error naming "tool-id".
func TestToolsLookupMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{}) // deliberately omit --tool-id
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool-id")
}

// TestToolsLookupNoNameFlag asserts the command has no --name flag (RES-09
// scope correction: the spec only supports toolId-based lookup).
func TestToolsLookupNoNameFlag(t *testing.T) {
	c := newLookupCmd()
	assert.Nil(t, c.Flags().Lookup("name"), "tools lookup must not have a --name flag")

	f := c.Flags().Lookup("tool-id")
	require.NotNil(t, f, "tools lookup must have a --tool-id flag")
}

// TestToolsLookupCmdRegistered asserts lookup is wired into the tools parent
// Cmd's init(), so the Phase 1 SAFE-01 registration-completeness test stays green.
func TestToolsLookupCmdRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "lookup" {
			found = true
			break
		}
	}
	require.True(t, found, "lookup subcommand must be registered on tools parent Cmd")
}
