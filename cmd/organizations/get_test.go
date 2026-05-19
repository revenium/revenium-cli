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

// TestGetOrganization asserts GET /v2/api/organizations/<id> renders the
// single-row 3-col table (ID / Name / Status). Per RESEARCH D-04a LOCKED the
// Status column is empty for organizations — the load-bearing assertions are
// id + name presence in the rendered table output.
func TestGetOrganization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/org-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"org-1","name":"Acme Corporation","label":"Acme Corporation","resourceType":"organization","_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "org-1")
	assert.Contains(t, out, "Acme Corporation")
}

// TestGetOrganizationJSON asserts --json mode emits parseable JSON containing
// the organization id and preserves the `_links` HATEOAS field (pass-through
// per RESEARCH Open Question 3 — DO NOT strip _links).
func TestGetOrganizationJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/org-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"org-1","name":"Acme Corporation","label":"Acme Corporation","resourceType":"organization","_links":{"self":{"href":"/v2/api/organizations/org-1"}}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "org-1", result["id"])
	assert.Equal(t, "Acme Corporation", result["name"])
	// HATEOAS pass-through (RESEARCH Open Question 3 — do not strip _links).
	assert.Contains(t, result, "_links")
}
