package jobs

import (
	"bytes"
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

// TestDeleteJobWithYes — standard happy path with --yes.
//
// Asserts:
//   - HTTP method is exactly "DELETE" (literal, not method-helper-erased)
//   - URL path is /v2/api/jobs/loan-app-1 (the single-item form, D-22+D-25)
//   - URL path is NOT /v2/api/jobs (RESEARCH §Risk 4: bulk-delete collision guard)
//   - Output contains "Deleted job loan-app-1." exactly (pattern S8 / D-12)
func TestDeleteJobWithYes(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1", r.URL.Path)
		// Defensive: never hit the bulk-delete endpoint (RESEARCH §Risk 4).
		// ExactArgs(1) prevents zero-arg invocation at the cobra layer, but this
		// runtime assertion catches future regressions where someone might
		// accidentally drop the path-suffix interpolation.
		assert.NotEqual(t, "/v2/api/jobs", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "loan-app-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDeleteCmd()
	// Pattern S10: re-register --yes as a local flag for the standalone
	// subcommand test. At runtime the flag is inherited from rootCmd as a
	// persistent flag — but tests instantiate the subcommand in isolation.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Deleted job loan-app-1.")
}

// TestDeleteJobQuiet — confirms --quiet suppresses the "Deleted job ..." line
// (pattern S8). The DELETE call still fires; only the stdout summary is muted.
func TestDeleteJobQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1", r.URL.Path)
		assert.NotEqual(t, "/v2/api/jobs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "loan-app-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 4th arg true = quiet=true.
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// TestDeleteJobJSONMode — confirms JSON mode auto-confirms without --yes
// (D-12). The handler still sees DELETE on the single-item path.
func TestDeleteJobJSONMode(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1", r.URL.Path)
		assert.NotEqual(t, "/v2/api/jobs", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "loan-app-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 3rd arg true = jsonMode=true; ConfirmDelete should auto-confirm.
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDeleteCmd()
	// Register --yes so the flag is parseable; the test does NOT pass it.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
}
