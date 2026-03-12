package credentials

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

func TestCreateCredential(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/credentials", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "cred-1", "label": "My Key", "provider": "openai", "credentialType": "API_KEY", "apiKey": "sk-abc123xyz7f3a"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--label", "My Key", "--provider", "openai", "--credential-type", "API_KEY", "--api-key", "sk-abc123xyz7f3a"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "cred-1")
	assert.Equal(t, "My Key", receivedBody["label"])
	assert.Equal(t, "openai", receivedBody["provider"])
	assert.Equal(t, "API_KEY", receivedBody["credentialType"])
	assert.Equal(t, "sk-abc123xyz7f3a", receivedBody["apiKey"])
}

func TestCreateCredentialMinimal(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "cred-1", "label": "My Key"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--label", "My Key"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "My Key", receivedBody["label"])
	_, hasProvider := receivedBody["provider"]
	assert.False(t, hasProvider, "provider should not be sent when not specified")
	_, hasType := receivedBody["credentialType"]
	assert.False(t, hasType, "credentialType should not be sent when not specified")
	_, hasKey := receivedBody["apiKey"]
	assert.False(t, hasKey, "apiKey should not be sent when not specified")
}
