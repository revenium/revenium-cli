package teams

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

// TestTeamsChildren asserts the children verb hits the literal
// /v2/api/teams/parent/{parentId} path (Pitfall 6 — literal `parent/` segment,
// NOT `/{id}/children`) and renders ID / Name rows from the HATEOAS
// `_embedded` envelope. The path-Equal assertion is load-bearing — Contains
// would be insufficient because it could match `/{id}/children` substrings.
func TestTeamsChildren(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/teams/parent/parent-team-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"_embedded": {
				"teamResourceList": [
					{"id":"child-team-1","name":"Acme NA","label":"Acme NA"}
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
	c.SetArgs([]string{"parent-team-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "child-team-1")
	assert.Contains(t, out, "Acme NA")
}

// TestTeamsChildrenNotAtIDChildrenPath is a regression guard for Pitfall 6:
// asserts the request path is NEVER /v2/api/teams/{id}/children.
func TestTeamsChildrenNotAtIDChildrenPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEqual(t, "/v2/api/teams/parent-team-1/children", r.URL.Path)
		assert.Equal(t, "/v2/api/teams/parent/parent-team-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-team-1"})
	err := c.Execute()

	require.NoError(t, err)
}

// TestTeamsChildrenEmpty asserts the empty-state phrase fires for the
// plain-array `[]` fixture.
func TestTeamsChildrenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/teams/parent/parent-team-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-team-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No child teams found.")
}

// TestTeamsChildrenJSON asserts --json mode emits parseable JSON for the
// plain-array fixture.
func TestTeamsChildrenJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/teams/parent/parent-team-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"child-team-1","name":"Acme NA"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-team-1"})
	err := c.Execute()

	require.NoError(t, err)
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "child-team-1", parsed[0]["id"])
}

// TestTeamsChildrenIsDemoFlag asserts --is-demo appends the isDemo query
// param only when the flag is explicitly passed.
func TestTeamsChildrenIsDemoFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/teams/parent/parent-team-1", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("isDemo"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-team-1", "--is-demo"})
	err := c.Execute()

	require.NoError(t, err)
}

// TestTeamsChildrenNoIsDemoFlagOmitsParam asserts isDemo is absent from the
// query string when --is-demo is not passed.
func TestTeamsChildrenNoIsDemoFlagOmitsParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/teams/parent/parent-team-1", r.URL.Path)
		assert.Empty(t, r.URL.Query().Get("isDemo"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newChildrenCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"parent-team-1"})
	err := c.Execute()

	require.NoError(t, err)
}
