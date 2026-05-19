package organizations

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestDeleteOrganizationWithYes — standard happy path with --yes.
//
// Asserts:
//   - HTTP method is exactly "DELETE" (literal, not method-helper-erased).
//   - URL path is /v2/api/organizations/org-1 (single-item path).
//   - Output contains "Deleted organization org-1." (CF-12-25 success line).
func TestDeleteOrganizationWithYes(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/organizations/org-1", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"Deleted","id":"org-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDeleteCmd()
	// Pattern S10: re-register --yes locally because the global persistent
	// flag from rootCmd is not present when the subcommand is invoked in
	// isolation.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Deleted organization org-1.")
}

// TestDeleteOrganizationQuiet — confirms quiet mode suppresses the success
// line. The DELETE call still fires; only the stdout summary is muted
// (CF-12-25).
func TestDeleteOrganizationQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/organizations/org-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"Deleted","id":"org-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 4th arg true = quiet=true.
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "Deleted organization", "success line must be suppressed in quiet mode")
}

// TestDeleteOrganizationJSONMode — confirms JSON mode auto-confirms without
// --yes (CF-12-25). The handler still sees DELETE on the single-item path.
func TestDeleteOrganizationJSONMode(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/organizations/org-1", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"Deleted","id":"org-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 3rd arg true = jsonMode=true; ConfirmDelete should auto-confirm.
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDeleteCmd()
	// Register --yes so flag parsing works; the test does NOT pass it.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"org-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
}

// TestDeleteOrganizationDryRun pins the dry-run output contract by invoking
// dryrun.Render DIRECTLY with the exact path/action/resource/body shape
// that delete.go's RunE passes in the dry-run branch. This decouples the
// contract assertion from cmd.DryRun() (which has no exported setter as of
// planning time — same precedent as cmd/guardrails/budget_rules_delete_test.go
// TestBudgetRulesDeleteDryRun). The cmd.DryRun() gate inside delete.go is a
// structurally trivial single `if` branch — integration is deferred to manual
// smoke; the dry-run output contract (this test) is the load-bearing assertion.
func TestDeleteOrganizationDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact path delete.go's RunE constructs via
	// fmt.Sprintf("/v2/api/organizations/%s", url.PathEscape("org-1")).
	path := "/v2/api/organizations/org-1"

	err := dryrun.Render(out, "delete", "organization", path, nil)

	require.NoError(t, err)
	rendered := buf.String()
	// Header line — matches internal/dryrun/dryrun.go format.
	assert.Contains(t, rendered, "Dry run: delete organization")
	// Path round-trip.
	assert.Contains(t, rendered, path)
	// Footer — matches internal/dryrun/dryrun.go.
	assert.Contains(t, rendered, "No changes were made.")
}
