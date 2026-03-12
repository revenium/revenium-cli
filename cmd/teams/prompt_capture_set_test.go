package teams

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

func TestPromptCaptureSet(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v2/api/teams/team-1/settings/prompts", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"enabled": true, "maxPromptLength": 4096, "systemMaxPromptLength": 8192}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPromptCaptureSetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"team-1", "--enabled=true"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, true, receivedBody["enabled"])
	out := buf.String()
	assert.Contains(t, out, "enabled")
	assert.Contains(t, out, "true")
}

func TestPromptCaptureSetNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPromptCaptureSetCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"team-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}
