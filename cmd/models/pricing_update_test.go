package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingUpdate(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "dim-1", "name": "Input Tokens", "dimensionType": "input", "unitPrice": "0.005"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "dim-1", "--price", "0.005"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "PUT", receivedMethod)
	assert.Equal(t, 0.005, receivedBody["unitPrice"])
	// Only price should be in the body, not name or type
	_, hasName := receivedBody["name"]
	assert.False(t, hasName, "name should not be sent when not changed")
	assert.Contains(t, buf.String(), "dim-1")
}

func TestPricingUpdatePath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "dim-1", "name": "Input Tokens", "dimensionType": "input", "unitPrice": "0.005"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"my-model", "my-dim", "--name", "Updated Name"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/sources/ai/models/my-model/pricing/dimensions/my-dim", receivedPath)
}

func TestPricingUpdateNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"mdl-1", "dim-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}
