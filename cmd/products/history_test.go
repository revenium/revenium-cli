package products

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

// TestHistory asserts the products history verb hits the LOCKED
// /v2/api/products/{id}/changelogs path (RES-10) and renders
// RevisionDataResource_Read field values from a HATEOAS fixture.
func TestHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/v2/api/products/")
		assert.Contains(t, r.URL.Path, "/changelogs")
		assert.Equal(t, "/v2/api/products/prod-1/changelogs", r.URL.Path)
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
	c.SetArgs([]string{"prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "UPDATE")
	assert.Contains(t, out, "Old")
	assert.Contains(t, out, "New")
	assert.Contains(t, out, "jane@example.com")
}

// TestHistoryEmpty asserts the empty-state phrase fires for a plain-array `[]`
// fixture.
func TestHistoryEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/products/prod-1/changelogs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No history found.")
}

// TestHistoryJSON asserts --json mode emits parseable JSON.
func TestHistoryJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/products/prod-1/changelogs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"attribute":"name","field":"name","changeType":"UPDATE","previousValue":"Old","currentValue":"New","changeDate":"2026-07-01T00:00:00Z","user":"jane@example.com"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "name", parsed[0]["attribute"])
}

// TestHistoryRegistered asserts history is wired into the products parent
// Cmd's init(), so the Phase 1 SAFE-01 registration-completeness test stays green.
func TestHistoryRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "history <id>" {
			found = true
			break
		}
	}
	require.True(t, found, "history subcommand must be registered on products parent Cmd")
}
