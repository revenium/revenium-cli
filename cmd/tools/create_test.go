package tools

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

func TestCreateTool(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/tools", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "tool-1", "toolId": "my-tool", "name": "My Tool", "toolType": "MCP_SERVER", "toolProvider": "", "enabled": true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "My Tool", "--tool-id", "my-tool", "--tool-type", "MCP_SERVER"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "tool-1")
	assert.Equal(t, "My Tool", receivedBody["name"])
	assert.Equal(t, "my-tool", receivedBody["toolId"])
	assert.Equal(t, "MCP_SERVER", receivedBody["toolType"])
	// Optional fields should not be in body
	_, hasDescription := receivedBody["description"]
	assert.False(t, hasDescription, "description should not be sent when not specified")
	_, hasProvider := receivedBody["toolProvider"]
	assert.False(t, hasProvider, "toolProvider should not be sent when not specified")
	_, hasEnabled := receivedBody["enabled"]
	assert.False(t, hasEnabled, "enabled should not be sent when not specified")
}

func TestCreateToolAllFields(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "tool-1", "toolId": "my-tool", "name": "My Tool", "description": "A test tool", "toolType": "MCP_SERVER", "toolProvider": "acme", "enabled": false}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--name", "My Tool",
		"--tool-id", "my-tool",
		"--tool-type", "MCP_SERVER",
		"--description", "A test tool",
		"--tool-provider", "acme",
		"--enabled=false",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "My Tool", receivedBody["name"])
	assert.Equal(t, "A test tool", receivedBody["description"])
	assert.Equal(t, "acme", receivedBody["toolProvider"])
	assert.Equal(t, false, receivedBody["enabled"])
}
