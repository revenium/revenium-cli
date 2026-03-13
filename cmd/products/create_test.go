package products

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

func TestCreateProduct(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/products", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "prod-1", "name": "My Product", "description": "A test product", "status": "active"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "My Product", "--description", "A test product"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "prod-1")
	assert.Equal(t, "My Product", receivedBody["name"])
	assert.Equal(t, "A test product", receivedBody["description"])
}

func TestCreateProductMinimal(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "prod-1", "name": "My Product", "status": "active"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "My Product"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "My Product", receivedBody["name"])
	_, hasDescription := receivedBody["description"]
	assert.False(t, hasDescription, "description should not be sent when not specified")
}
