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

// TestListOrganizations asserts the list verb hits the RESEARCH-LOCKED
// /v2/api/organizations path and renders ID / Name rows from the HATEOAS
// `_embedded.organizationResourceList` envelope (resource-specific key per
// Pitfall 4 — NOT the generic `objectList`).
func TestListOrganizations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// Verified HATEOAS fixture: resource-specific `organizationResourceList` key
		// per RESEARCH Pitfall 4 (matches the OAS OrganizationPagedModel envelope).
		fmt.Fprint(w, `{
			"_embedded": {
				"organizationResourceList": [
					{"id":"org-1","name":"Acme Corporation","label":"Acme Corporation"}
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

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "org-1")
	assert.Contains(t, out, "Acme Corporation")
}

// TestListOrganizationsEmpty asserts the empty-state phrase fires for the
// plain-array `[]` fixture (DoList tries plain decode first per client.go:372-374)
// and that NO table header is rendered.
func TestListOrganizationsEmpty(t *testing.T) {
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
	assert.Contains(t, buf.String(), "No organizations found.")
	assert.NotContains(t, buf.String(), "ID")
}

// TestListOrganizationsJSON asserts --json mode emits parseable JSON containing
// the organization id. Uses the plain-array path for parity with jobs/budget-rules
// JSON parity tests.
func TestListOrganizationsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"org-1","name":"Acme Corporation"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "org-1", parsed[0]["id"])
	assert.Equal(t, "Acme Corporation", parsed[0]["name"])
}

// TestListOrganizationsEmptyJSON asserts --json mode on an empty fixture emits
// the canonical empty array `[]`.
func TestListOrganizationsEmptyJSON(t *testing.T) {
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
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}
