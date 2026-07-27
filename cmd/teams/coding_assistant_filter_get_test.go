package teams

import (
	"bytes"
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

// TestCodingAssistantFilterGet asserts GET /v2/api/teams/{id}/settings/coding-assistant-filter
// renders the settings as key-value rows.
func TestCodingAssistantFilterGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/teams/team-1/settings/coding-assistant-filter", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"enabled": true,
			"apiRateProviders": ["ClaudeCode", "CursorIde"],
			"defaultProviders": ["ClaudeCode"],
			"allowUserOverride": false,
			"persistence": "PERSISTED",
			"classificationSource": "AUTO",
			"confidence": 0.9,
			"needsReview": false
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCodingAssistantFilterGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "apiRateProviders")
	assert.Contains(t, out, "ClaudeCode")
}

// TestCodingAssistantFilterGetJSON asserts --json mode renders the raw settings map.
func TestCodingAssistantFilterGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/teams/team-1/settings/coding-assistant-filter", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apiRateProviders": ["ClaudeCode"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newCodingAssistantFilterGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "apiRateProviders")
}

// TestCodingAssistantFilterParentHasGet asserts the parent command exposes
// the get subcommand (set is added in a follow-on task and asserted there).
func TestCodingAssistantFilterParentHasGet(t *testing.T) {
	var names []string
	for _, sub := range codingAssistantFilterCmd.Commands() {
		names = append(names, sub.Name())
	}
	assert.Contains(t, names, "get")
}
