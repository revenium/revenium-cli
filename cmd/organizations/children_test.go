package organizations

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

// TestOrganizationChildren asserts the children verb hits the LOCKED
// /v2/api/organizations/parent/{parentId} path (RESEARCH Pitfall 1 — literal
// `parent/` segment) and renders ID / Name / Status rows from the HATEOAS
// `_embedded.organizationResourceList` envelope (Pitfall 4 — resource-specific
// embedded key honored). The path-Equal assertion is load-bearing — Contains
// would be insufficient because it could match `/{id}/children` substrings.
func TestOrganizationChildren(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/parent/parent-org-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// Verified HATEOAS fixture per RESEARCH Pitfall 4 — resource-specific key
		// `organizationResourceList` (NOT generic `objectList`).
		fmt.Fprint(w, `{
			"_embedded": {
				"organizationResourceList": [
					{"id":"child-org-1","name":"Acme NA","label":"Acme NA"}
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

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-org-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "child-org-1")
	assert.Contains(t, out, "Acme NA")
}

// TestOrganizationChildrenEmpty asserts the empty-state phrase fires for the
// plain-array `[]` fixture (DoList tries plain decode first per client.go:372-374)
// and that the path assertion still holds for the empty case.
func TestOrganizationChildrenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/parent/parent-org-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-org-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No child organizations found.")
}

// TestOrganizationChildrenJSON asserts --json mode emits parseable JSON for the
// plain-array fixture (DoList plain-decode fallback) and that the path assertion
// fires under JSON mode too.
func TestOrganizationChildrenJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/parent/parent-org-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"child-org-1","name":"Acme NA"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-org-1"})
	err := c.Execute()

	require.NoError(t, err)
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "child-org-1", parsed[0]["id"])
}

// TestOrganizationChildrenEmptyJSON asserts --json mode on an empty fixture emits
// the canonical empty array `[]` (RenderJSON([]interface{}{}) path).
func TestOrganizationChildrenEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/parent/parent-org-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-org-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Empty(t, result)
}
