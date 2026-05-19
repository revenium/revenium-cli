package jobs

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

// TestTypes covers the populated path for `revenium jobs types`.
// Asserts the endpoint path, HTTP method, and that the rendered table
// contains the header "Type" and each of the three type values returned by
// the server (RESEARCH §A4 — bare []string, no HATEOAS, no DoList).
func TestTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v2/api/jobs/types", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `["document-extract","loan-processing","underwriting"]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTypesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Type")
	assert.Contains(t, out, "document-extract")
	assert.Contains(t, out, "loan-processing")
	assert.Contains(t, out, "underwriting")
}

// TestTypesEmpty covers the empty-array branch in non-JSON mode.
// Asserts the canonical "No <resource> found." stdout message (Phase 12 D-09).
func TestTypesEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTypesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No job types found.")
}

// TestTypesEmptyJSON covers the empty-array branch in JSON mode.
// Asserts the rendered body is a parseable empty JSON array.
func TestTypesEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newTypesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestTypesJSON covers the populated path in JSON mode.
// Asserts the raw []string array is preserved verbatim (NOT wrapped in
// table-data shape).
func TestTypesJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `["document-extract","loan-processing","underwriting"]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newTypesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "document-extract", result[0])
}

// TestTypesTeamId asserts teamId auto-injection by Client.Do
// (internal/api/client.go:68-74) when the client is constructed with a TeamID.
// Mirrors cmd/jobs/update_test.go:66-86.
func TestTypesTeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `["document-extract"]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newTypesCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}
