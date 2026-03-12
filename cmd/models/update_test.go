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

func TestUpdateModel(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		assert.Equal(t, "/v2/api/sources/ai/models/mdl-1", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "mdl-1", "name": "GPT-4", "provider": "OpenAI", "mode": "chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "--team-id", "team-123", "--input-cost-per-token", "0.003"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "PATCH", receivedMethod, "update must use PATCH, not PUT")
	assert.Contains(t, buf.String(), "mdl-1")
	// Only input-cost-per-token should be in the body
	assert.Equal(t, 0.003, receivedBody["inputCostPerToken"])
	_, hasOutput := receivedBody["outputCostPerToken"]
	assert.False(t, hasOutput, "outputCostPerToken should not be sent when not changed")
}

func TestUpdateModelTeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "mdl-1", "name": "GPT-4", "provider": "OpenAI", "mode": "chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "--team-id", "team-456", "--output-cost-per-token", "0.006"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}

func TestUpdateModelNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"mdl-1", "--team-id", "team-123"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}
