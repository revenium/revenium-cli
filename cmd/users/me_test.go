package users

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestUsersMe asserts `users me` GETs the literal /v2/api/users/me path and
// renders the current user via the existing renderUser helper (D-06).
func TestUsersMe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/users/me", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "jane@example.com", "firstName": "Jane", "lastName": "Doe", "roles": ["ROLE_API_CONSUMER"]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "jane@example.com")
}

// TestUsersMeCmdRegistered asserts `me` is registered as a subcommand of the
// users parent Cmd.
func TestUsersMeCmdRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "me" {
			found = true
			break
		}
	}
	require.True(t, found, "me subcommand must be registered on users parent Cmd")
}

// TestCurrentUserID asserts the shared helper resolves the id field from
// GET /v2/api/users/me and errors clearly when it is absent.
func TestCurrentUserID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/users/me", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "user-1", "email": "jane@example.com"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeCmd()
	c.SetContext(context.Background())
	id, err := currentUserID(c)
	require.NoError(t, err)
	assert.Equal(t, "user-1", id)
}

func TestCurrentUserIDMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"email": "jane@example.com"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeCmd()
	c.SetContext(context.Background())
	_, err := currentUserID(c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not resolve current user id")
}
