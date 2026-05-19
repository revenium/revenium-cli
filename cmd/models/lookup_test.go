package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLookupModel covers the happy path in table mode. The fake server
// asserts that the supplied --name value is interpolated into the URL path
// segment (NOT the query string) — this is the load-bearing distinction
// vs. subscribers/users lookup which use ?email= query parameters.
func TestLookupModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/name/gpt-4", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "mdl-1", "name": "gpt-4", "provider": "OpenAI", "mode": "chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "gpt-4"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "mdl-1")
	assert.Contains(t, out, "gpt-4")
	assert.Contains(t, out, "OpenAI")
	assert.Contains(t, out, "chat")
}

// TestLookupModelJSON covers the happy path in JSON mode. Asserts the body
// is parseable JSON and contains the id/name fields verbatim.
func TestLookupModelJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "mdl-1", "name": "gpt-4", "provider": "OpenAI", "mode": "chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "gpt-4"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "mdl-1", result["id"])
	assert.Equal(t, "gpt-4", result["name"])
}

// TestLookupModelNotFound asserts 404 responses flow through the default
// mapHTTPError path, producing "Resource not found." (D-03 — no per-verb
// override). The verb path goes through cmd.APIClient.Do, so the standard
// error message is produced verbatim (internal/api/client.go:144).
func TestLookupModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "missing-model"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Resource not found")
}

func TestLookupCmdRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "lookup" {
			found = true
			break
		}
	}
	require.True(t, found, "lookup subcommand must be registered on models parent Cmd")
}

// TestLookupModelMissingRequired is the LOAD-BEARING D-04 assertion: omitting
// --name must error at the Cobra MarkFlagRequired layer BEFORE any HTTP
// round-trip. Configures an unreachable BaseURL so an accidental HTTP attempt
// would surface a connection error instead of the expected Cobra one.
func TestLookupModelMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{})

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name",
		"missing-flag error must name 'name' so users know what to add")
}
