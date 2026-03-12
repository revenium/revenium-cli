package users

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

func TestCreateUser(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/users", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "jane@example.com", "firstName": "Jane", "lastName": "Doe", "roles": ["ROLE_API_CONSUMER"], "teamIds": ["team-1"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--email", "jane@example.com", "--first-name", "Jane", "--last-name", "Doe", "--roles", "ROLE_API_CONSUMER", "--team-ids", "team-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "user-1")
	assert.Equal(t, "jane@example.com", receivedBody["email"])
	assert.Equal(t, "Jane", receivedBody["firstName"])
	assert.Equal(t, "Doe", receivedBody["lastName"])
	// Verify roles sent as array
	roles, ok := receivedBody["roles"].([]interface{})
	require.True(t, ok, "roles should be an array")
	assert.Contains(t, roles, "ROLE_API_CONSUMER")
	// Verify teamIds sent as array
	teamIDs, ok := receivedBody["teamIds"].([]interface{})
	require.True(t, ok, "teamIds should be an array")
	assert.Contains(t, teamIDs, "team-1")
}

func TestCreateUserWithOptional(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "jane@example.com", "firstName": "Jane", "lastName": "Doe", "roles": ["ROLE_API_CONSUMER"], "teamIds": ["team-1"], "phoneNumber": "555-1234", "canViewPromptData": true}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--email", "jane@example.com", "--first-name", "Jane", "--last-name", "Doe", "--roles", "ROLE_API_CONSUMER", "--team-ids", "team-1", "--phone-number", "555-1234", "--can-view-prompt-data"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "555-1234", receivedBody["phoneNumber"])
	assert.Equal(t, true, receivedBody["canViewPromptData"])
}
