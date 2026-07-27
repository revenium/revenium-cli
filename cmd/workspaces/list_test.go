package workspaces

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

// TestWorkspacesList asserts the list verb hits GET /v2/api/workspaces and
// renders ID/Name/Provider rows from the generic PagedModel_Read
// `_embedded.objectList` envelope (DoList takes the first array found
// regardless of key name).
func TestWorkspacesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/workspaces", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"_embedded": {
				"objectList": [
					{"id":"ws-1","name":"Production","provider":"aws"}
				]
			},
			"_links": {}
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "ws-1")
	assert.Contains(t, out, "Production")
	assert.Contains(t, out, "aws")
}

// TestWorkspacesListFilters asserts --provider and --query are flag-gated
// and appended to the request query string, chained with `&` when both are
// present (idiom copied from cmd/anomalies/dimensions.go / cmd/squads/squads.go).
func TestWorkspacesListFilters(t *testing.T) {
	var seenQuery, seenProvider string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.Query().Get("query")
		seenProvider = r.URL.Query().Get("provider")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--provider", "aws", "--query", "foo"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "aws", seenProvider)
	assert.Equal(t, "foo", seenQuery)
}

// TestWorkspacesListNoFlags asserts no query segment is added when neither
// --provider nor --query is passed.
func TestWorkspacesListNoFlags(t *testing.T) {
	var sawProvider, sawQuery bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawProvider = r.URL.Query().Has("provider")
		sawQuery = r.URL.Query().Has("query")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.False(t, sawProvider)
	assert.False(t, sawQuery)
}

// TestWorkspacesListEmpty asserts the canonical empty-state phrase fires and
// no table header is rendered.
func TestWorkspacesListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No workspaces found.")
	assert.NotContains(t, buf.String(), "ID")
}

// TestWorkspacesListEmptyJSON asserts --json mode on an empty fixture emits
// the canonical empty array `[]`.
func TestWorkspacesListEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Empty(t, result)
}
