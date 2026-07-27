package meteringelements

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

// TestMeteringElementsDeleteWithYes — standard happy path with --yes.
// Asserts DELETE method, path, success line, AND that the
// DeleteResponse_Read body is decoded (A2 deviation from organizations).
func TestMeteringElementsDeleteWithYes(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/metering-element-definitions/me-1", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-1","message":"Deleted","created":"2026-01-01T00:00:00Z","updated":"2026-01-01T00:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"me-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Deleted metering element me-1.")
}

// TestMeteringElementsDeleteJSONModeBody asserts JSON mode auto-confirms
// without --yes and decodes/renders the DeleteResponse_Read body.
func TestMeteringElementsDeleteJSONModeBody(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/metering-element-definitions/me-1", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-1","message":"Deleted"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"me-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
	assert.Contains(t, buf.String(), `"message"`)
	assert.Contains(t, buf.String(), `"Deleted"`)
}

// TestMeteringElementsDeleteQuiet — confirms quiet mode suppresses the
// success line.
func TestMeteringElementsDeleteQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-1","message":"Deleted"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"me-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "Deleted metering element", "success line must be suppressed in quiet mode")
}

// TestMeteringElementsDeleteDryRun pins the dry-run output contract.
func TestMeteringElementsDeleteDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	path := "/v2/api/metering-element-definitions/me-1"

	err := dryrun.Render(out, "delete", "metering element", path, nil)

	require.NoError(t, err)
	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: delete metering element")
	assert.Contains(t, rendered, path)
	assert.Contains(t, rendered, "No changes were made.")
}
