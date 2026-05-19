package guardrails

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

// TestBudgetRulesDeleteWithYes — standard happy path with --yes.
//
// Asserts:
//   - HTTP method is exactly "DELETE" (literal, not method-helper-erased).
//   - URL path is /v2/api/ai/cost-controls/jR2kmLs (single-item path).
//   - Output contains "Deleted budget rule jR2kmLs." (CF-12-25 success line).
func TestBudgetRulesDeleteWithYes(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "jR2kmLs"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesDeleteCmd()
	// Pattern S10: re-register --yes locally because the global persistent
	// flag from rootCmd is not present when the subcommand is invoked in
	// isolation.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Deleted budget rule jR2kmLs.")
}

// TestBudgetRulesDeleteQuiet — confirms quiet mode suppresses the success
// line. The DELETE call still fires; only the stdout summary is muted
// (CF-12-25).
func TestBudgetRulesDeleteQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "jR2kmLs"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 4th arg true = quiet=true.
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newBudgetRulesDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "Deleted budget rule", "success line must be suppressed in quiet mode")
}

// TestBudgetRulesDeleteJSONMode — confirms JSON mode auto-confirms without
// --yes (CF-12-25). The handler still sees DELETE on the single-item path.
func TestBudgetRulesDeleteJSONMode(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "jR2kmLs"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 3rd arg true = jsonMode=true; ConfirmDelete should auto-confirm.
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newBudgetRulesDeleteCmd()
	// Register --yes so flag parsing works; the test does NOT pass it.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
}

// TestBudgetRulesDeleteDryRun pins the dry-run output contract by invoking
// dryrun.Render DIRECTLY with the exact path/action/resource/body shape
// that delete.go's RunE passes in the dry-run branch. This decouples the
// contract assertion from cmd.DryRun() (which has no exported setter as of
// planning time — same situation as cmd/jobs/outcome_test.go
// TestOutcomeDryRun). The cmd.DryRun() gate inside delete.go is a
// structurally trivial single `if` branch — integration is deferred to
// manual smoke; the dry-run output contract (this test) is the load-bearing
// assertion.
func TestBudgetRulesDeleteDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact path delete.go's RunE constructs via
	// fmt.Sprintf("/v2/api/ai/cost-controls/%s", url.PathEscape("jR2kmLs")).
	path := "/v2/api/ai/cost-controls/jR2kmLs"

	err := dryrun.Render(out, "delete", "budget rule", path, nil)

	require.NoError(t, err)
	rendered := buf.String()
	// Header line — matches internal/dryrun/dryrun.go format.
	assert.Contains(t, rendered, "Dry run: delete budget rule")
	// Path round-trip.
	assert.Contains(t, rendered, path)
	// Footer — matches internal/dryrun/dryrun.go.
	assert.Contains(t, rendered, "No changes were made.")
}
