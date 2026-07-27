package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestCodingAssistantFilterSet asserts set PUTs apiRateProviders only, per
// Assumption A4 — the body must NOT contain enabled/defaultProviders/allowUserOverride.
func TestCodingAssistantFilterSet(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v2/api/teams/team-1/settings/coding-assistant-filter", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apiRateProviders": ["ClaudeCode", "CursorIde"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCodingAssistantFilterSetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1", "--api-rate-providers", "ClaudeCode,CursorIde"})
	err := c.Execute()

	require.NoError(t, err)

	providers, ok := receivedBody["apiRateProviders"].([]interface{})
	require.True(t, ok, "apiRateProviders must be present in the request body")
	assert.ElementsMatch(t, []interface{}{"ClaudeCode", "CursorIde"}, providers)

	_, hasEnabled := receivedBody["enabled"]
	_, hasDefaultProviders := receivedBody["defaultProviders"]
	_, hasAllowUserOverride := receivedBody["allowUserOverride"]
	assert.False(t, hasEnabled, "A4: enabled must not be sent — deprecated in prose")
	assert.False(t, hasDefaultProviders, "A4: defaultProviders must not be sent — deprecated in prose")
	assert.False(t, hasAllowUserOverride, "A4: allowUserOverride must not be sent — deprecated in prose")
}

// TestCodingAssistantFilterSetNoFields asserts the no-op error when no flags are passed.
func TestCodingAssistantFilterSetNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCodingAssistantFilterSetCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"team-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}

// TestCodingAssistantFilterSetDryRun pins the dry-run output contract by invoking
// dryrun.Render directly with the exact path/verb/resource/body shape that
// coding_assistant_filter_set.go's RunE passes when cmd.DryRun() is true.
// cmd.DryRun() has no exported setter (see cmd/organizations/create_test.go
// precedent), so the dry-run branch itself is a trivial single `if` covered
// by manual smoke; this test pins the render contract.
func TestCodingAssistantFilterSetDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	body := map[string]interface{}{
		"apiRateProviders": []string{"ClaudeCode"},
	}

	err := dryrun.Render(out, "update", "coding-assistant filter settings", "/v2/api/teams/team-1/settings/coding-assistant-filter", body)
	require.NoError(t, err)

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: update coding-assistant filter settings")
	assert.Contains(t, rendered, "/v2/api/teams/team-1/settings/coding-assistant-filter")
	assert.Contains(t, rendered, "No changes were made.")
}

// TestCodingAssistantFilterParentHasGetAndSet asserts the parent command exposes
// both get and set subcommands once Task 3 wires set.
func TestCodingAssistantFilterParentHasGetAndSet(t *testing.T) {
	var names []string
	for _, sub := range codingAssistantFilterCmd.Commands() {
		names = append(names, sub.Name())
	}
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "set")
}
