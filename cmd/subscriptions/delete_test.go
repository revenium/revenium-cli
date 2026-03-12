package subscriptions

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

func TestDeleteSubscriptionWithYes(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/subscriptions/sub-1", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "sub-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDeleteCmd()
	// Register --yes flag for standalone test (inherited from rootCmd at runtime)
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Deleted subscription sub-1.")
}

func TestDeleteSubscriptionQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "sub-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newDeleteCmd()
	// Register --yes flag for standalone test (inherited from rootCmd at runtime)
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestDeleteSubscriptionJSONMode(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "sub-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDeleteCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
}
