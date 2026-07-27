package models

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

// TestModelHistory asserts the models history verb hits the LOCKED
// /v2/api/sources/ai/models/{id}/revisions path (RES-01; roadmap verb is
// "history", the spec's path segment is "revisions" per D-08) and renders
// RevisionDataResource_Read field values from a HATEOAS fixture.
func TestModelHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/revisions")
		assert.Equal(t, "/v2/api/sources/ai/models/model-1/revisions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"_embedded": {
				"revisionDataResourceList": [
					{"attribute":"name","field":"name","changeType":"UPDATE","previousValue":"Old","currentValue":"New","changeDate":"2026-07-01T00:00:00Z","user":"jane@example.com"}
				]
			},
			"_links": {},
			"page": {"size": 20, "totalElements": 1, "totalPages": 1, "number": 0}
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"model-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "UPDATE")
	assert.Contains(t, out, "Old")
	assert.Contains(t, out, "New")
	assert.Contains(t, out, "jane@example.com")
}

// TestModelHistoryEmpty asserts the empty-state phrase fires for a plain-array
// `[]` fixture.
func TestModelHistoryEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/model-1/revisions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"model-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No history found.")
}

// TestModelHistoryJSON asserts --json mode emits parseable JSON.
func TestModelHistoryJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/model-1/revisions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"attribute":"name","field":"name","changeType":"UPDATE","previousValue":"Old","currentValue":"New","changeDate":"2026-07-01T00:00:00Z","user":"jane@example.com"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"model-1"})
	err := c.Execute()

	require.NoError(t, err)
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "name", parsed[0]["attribute"])
}

// TestModelHistoryRegistered asserts history is wired into the models parent
// Cmd's init(), so the Phase 1 SAFE-01 registration-completeness test stays green.
func TestModelHistoryRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "history <id>" {
			found = true
			break
		}
	}
	require.True(t, found, "history subcommand must be registered on models parent Cmd")
}

// TestModelRates asserts models rates GETs the flat-wrapper
// /v2/api/sources/ai/models/rates path (not DoList — no _embedded key) and
// renders one row per entry in the "models" array.
func TestModelRates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/rates", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"models":[{"model":"gpt-4","provider":"openai","inputPricePerMillion":30,"outputPricePerMillion":60,"cacheReadPricePerMillion":15,"cacheWritePricePerMillion":37.5}]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newRatesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "gpt-4")
	assert.Contains(t, out, "openai")
}

// TestModelRatesEmpty asserts an empty/absent "models" key renders the
// empty-state phrase without panicking.
func TestModelRatesEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/rates", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newRatesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No rates found.")
}

// TestModelRatesJSON asserts --json mode emits parseable JSON.
func TestModelRatesJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/rates", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"models":[{"model":"gpt-4","provider":"openai","inputPricePerMillion":30,"outputPricePerMillion":60,"cacheReadPricePerMillion":15,"cacheWritePricePerMillion":37.5}]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newRatesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "gpt-4", parsed[0]["model"])
}

// TestModelRatesRegistered asserts rates is wired into the models parent
// Cmd's init(), so the Phase 1 SAFE-01 registration-completeness test stays green.
func TestModelRatesRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "rates" {
			found = true
			break
		}
	}
	require.True(t, found, "rates subcommand must be registered on models parent Cmd")
}
