package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	var errBuf bytes.Buffer
	c.SetOut(&buf)
	// SetErr keeps the BILL-07 deprecation notice out of `go test` output; the
	// notice itself is asserted in TestUsersGetDeprecationWarning.
	c.SetErr(&errBuf)
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
	var errBuf bytes.Buffer
	c.SetOut(&buf)
	// SetErr keeps the BILL-07 deprecation notice out of `go test` output; the
	// notice itself is asserted in TestUsersGetDeprecationWarning.
	c.SetErr(&errBuf)
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

// TestUsersGetDeprecationWarning asserts all four halves of BILL-07's criterion
// for `billing users get <email>` — notice on stderr, absent from stdout,
// stdout still parseable, exit code unchanged — plus the two-successor naming
// and the fixed order D-30-01 chose.
func TestUsersGetDeprecationWarning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"userEmail": "john@example.com", "entries": [{"cost": 1.0}]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// JSON mode ON so the test proves the strongest form of the claim.
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newUsersCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	// c.SetErr is load-bearing: without it ErrOrStderr() falls through to the
	// process's real stderr, the assertions below read an empty string, and the
	// test passes proving nothing. Task 30-04-03 mutation-proves this.
	c.SetErr(&errBuf)
	c.SetArgs([]string{"get", "john@example.com"})
	err := c.Execute()

	require.NoError(t, err, "the deprecation notice must not change the exit code")
	notice := errBuf.String()
	assert.Contains(t, notice, "deprecated")
	assert.Contains(t, notice, "revenium billing users")
	assert.Contains(t, notice, "revenium billing vcs-pr-health")
	// D-30-01: the cost successor is named FIRST because the withdrawn endpoint
	// returned per-user cost detail; the PR-health command answers an adjacent
	// question and comes second. An operator reading left to right meets the
	// closer redirect first.
	assert.Less(t, strings.Index(notice, "revenium billing users"),
		strings.Index(notice, "revenium billing vcs-pr-health"),
		"the cost successor must be named before the activity successor")
	assert.NotContains(t, buf.String(), "deprecated",
		"the deprecation warning must never appear on stdout or JSON output")
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed),
		"stdout must still be a single parseable JSON document")
}

// TestUsersListEmitsNoDeprecation asserts the SURVIVING `billing users` list
// stays silent. Per D-30-02 it targets /v2/api/billing/users, which is present
// and healthy in the dev platform document at 2.20.0-SNAPSHOT; warning there
// would tell an operator a working command is going away. Emptiness is asserted
// rather than a missing substring — the stronger claim, and it also catches a
// notice that gets reworded later.
func TestUsersListEmitsNoDeprecation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"userEmail": "jane@example.com", "totalCost": 42.0}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUsersCmd()
	var errBuf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&errBuf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Empty(t, errBuf.String(),
		"the surviving billing users list must write nothing to stderr (D-30-02)")
}
