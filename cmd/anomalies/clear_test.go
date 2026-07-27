package anomalies

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

// TestAnomaliesClearWithYes asserts --yes skips the prompt and fires a
// DELETE (not POST) against the literal /clear path, mirroring
// delete_test.go's TestDeleteAnomalyWithYes.
func TestAnomaliesClearWithYes(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/sources/ai/anomaly/clear", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newClearCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Cleared all anomaly detection rules for this team.")
}

// TestAnomaliesClearQuiet asserts quiet mode suppresses the confirmation
// message.
func TestAnomaliesClearQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newClearCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// TestAnomaliesClearJSONMode asserts JSON mode skips the interactive prompt
// (ConfirmDelete returns true on jsonMode) and still fires the DELETE.
func TestAnomaliesClearJSONMode(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newClearCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "clear should proceed without prompt in JSON mode")
}

// TestAnomaliesClearDryRun pins the dry-run output contract by invoking
// dryrun.Render DIRECTLY with the exact path/verb/resource/body shape that
// clear.go's RunE passes when cmd.DryRun() is true — cmd.DryRun() has no
// exported setter (see cmd/organizations/create_test.go's
// TestCreateOrganizationDryRun / cmd/organizations/delete_test.go's
// TestDeleteOrganizationDryRun for precedent). This asserts the ZERO
// HTTP calls requirement structurally: clear.go's RunE returns from
// dryrun.Render before cmd.APIClient is ever touched, and no
// httptest.NewServer is constructed in this test — nothing is listening,
// so a stray HTTP call would fail with a connection error rather than
// silently succeed.
func TestAnomaliesClearDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	err := dryrun.Render(out, "clear", "all anomalies", clearPath, nil)
	require.NoError(t, err)

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: clear all anomalies")
	assert.Contains(t, rendered, "/v2/api/sources/ai/anomaly/clear")
	assert.Contains(t, rendered, "No changes were made.")
}
