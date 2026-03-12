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

func TestPricingDelete(t *testing.T) {
	var deleteCalled bool
	var receivedMethod string
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalled = true
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "dim-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/v2/api/sources/ai/models/mdl-1/pricing/dimensions/dim-1", receivedPath)
	assert.Contains(t, buf.String(), "Deleted pricing dimension dim-1.")
}

func TestPricingDeleteJSON(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newPricingDeleteCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "dim-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
}
