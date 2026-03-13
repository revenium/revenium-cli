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

func TestUpdateUser(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/users/user-1", r.URL.Path)
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "user-1",
				"email":     "old@example.com",
				"firstName": "Jane",
				"lastName":  "Doe",
			})
			return
		}
		assert.Equal(t, "PUT", r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "new@example.com", "firstName": "Jane", "lastName": "Doe", "roles": ["ROLE_API_CONSUMER"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"user-1", "--email", "new@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "new@example.com")
	assert.Equal(t, "new@example.com", receivedBody["email"])
}

func TestUpdateUserWithDefaults(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "user-1",
				"email":     "test@example.com",
				"firstName": "Jane",
				"lastName":  "Doe",
			})
			return
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "test@example.com", "firstName": "Updated", "lastName": "Doe"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"user-1", "--first-name", "Updated"})
	err := c.Execute()

	require.NoError(t, err)
	// Verify default roles and teamIds are included
	roles, ok := receivedBody["roles"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, roles, "ROLE_API_CONSUMER")
	teamIds, ok := receivedBody["teamIds"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, teamIds, "team-1")
}
