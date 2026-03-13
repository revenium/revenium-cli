package models

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

func TestDeleteModel(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/sources/ai/models/mdl-1", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "mdl-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDeleteCmd()
	// Register --yes flag for standalone test (inherited from rootCmd at runtime)
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Deleted model mdl-1.")
}

func TestDeleteModelJSON(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "mdl-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDeleteCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
}
