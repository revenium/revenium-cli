package billing

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

// TestBillingUsersList asserts `billing users` (bare, no subcommand) lists
// GET /v2/api/billing/users and renders rows.
func TestBillingUsersList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/billing/users", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"userEmail": "jane@example.com", "totalCost": 42.0}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUsersCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "jane@example.com")
}

// TestBillingUsersListEmpty asserts the canonical empty-state phrase fires.
func TestBillingUsersListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUsersCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No billing user cost data found.")
}

// TestBillingUsersGet asserts `billing users get <email>` GETs
// /v2/api/billing/users/{email} — proving `@` survives ValidResourceID +
// url.PathEscape unmodified (T-06-09 mitigation) — and renders the detail.
func TestBillingUsersGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"userEmail": "john@example.com", "entries": [{"cost": 1.0}, {"cost": 2.0}]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUsersCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"get", "john@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/billing/users/john@example.com", gotPath)
	out := buf.String()
	assert.Contains(t, out, "john@example.com")
}

// TestBillingUsersGetJSON asserts JSON mode passes the detail object
// through unchanged.
func TestBillingUsersGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"userEmail": "john@example.com", "entries": [{"cost": 1.0}]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newUsersCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"get", "john@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "john@example.com", parsed["userEmail"])
}

// TestBillingUsersGetRejectsMalformedID asserts a malformed positional arg
// (containing "?") is rejected by ValidResourceID before any HTTP call.
func TestBillingUsersGetRejectsMalformedID(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUsersCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"get", "bad?id"})
	err := c.Execute()

	require.Error(t, err)
	assert.False(t, called)
}
