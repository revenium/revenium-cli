package users

import (
	"bytes"
	"encoding/json"
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

func TestLookupUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/users/lookup-by-email", r.URL.Path)
		assert.Equal(t, "jane@example.com", r.URL.Query().Get("email"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "jane@example.com", "firstName": "Jane", "lastName": "Doe", "roles": ["ROLE_API_CONSUMER"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--email", "jane@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "user-1")
	assert.Contains(t, out, "jane@example.com")
}

func TestLookupUserJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "jane@example.com", "firstName": "Jane", "lastName": "Doe", "roles": ["ROLE_API_CONSUMER"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--email", "jane@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "user-1", result["id"])
	assert.Equal(t, "jane@example.com", result["email"])
}

func TestLookupUserNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--email", "missing@example.com"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Resource not found")
}

func TestLookupCmdRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "lookup" {
			found = true
			break
		}
	}
	require.True(t, found, "lookup subcommand must be registered on users parent Cmd")
}

func TestLookupUserMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLookupCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{}) // deliberately omit --email
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "email",
		"missing-flag error must name 'email' so users know what to add")
}
