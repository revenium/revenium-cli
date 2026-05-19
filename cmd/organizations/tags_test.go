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

// TestOrganizationTags asserts the populated path: GET /v2/api/organizations/org-1/tags
// returns ["enterprise","production","api"] and the table render contains every tag.
func TestOrganizationTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/org-1/tags", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `["enterprise","production","api"]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTagsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "enterprise")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "api")
}

// TestOrganizationTagsEmpty asserts the empty-text branch — server returns [] and
// the verb prints exactly "No tags." (RESEARCH D-02 LOCKED empty-state phrase).
func TestOrganizationTagsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/org-1/tags", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTagsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No tags.")
}

// TestOrganizationTagsJSON asserts that --json mode renders the raw []string
// pass-through (single-element fixture) so consumers can json.Unmarshal it.
func TestOrganizationTagsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/org-1/tags", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `["enterprise"]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newTagsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1"})
	err := c.Execute()

	require.NoError(t, err)
	var parsed []string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Len(t, parsed, 1)
	assert.Equal(t, "enterprise", parsed[0])
}

// TestOrganizationTagsEmptyJSON asserts that --json mode on empty tags renders
// a JSON empty array (not "No tags.") for script-friendly consumption.
func TestOrganizationTagsEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/organizations/org-1/tags", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newTagsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Empty(t, result)
}
