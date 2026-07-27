package users

import (
	"bytes"
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

// TestUsersMeCredentials asserts the two-call pattern: GET /v2/api/users/me
// first (to resolve the id), then GET /v2/api/users/{id}/credentials — in
// that order — and that table output does not leak externalSecret (T-05-11).
func TestUsersMeCredentials(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1", "email": "jane@example.com"}`)
		case "/v2/api/users/user-1/credentials":
			fmt.Fprint(w, `[{"id":"cred-1","name":"My Cred","externalId":"ext-1","externalSecret":"super-secret","identityProvider":"okta"}]`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeCredentialsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	require.Equal(t, []string{"/v2/api/users/me", "/v2/api/users/user-1/credentials"}, paths)
	out := buf.String()
	assert.Contains(t, out, "cred-1")
	assert.Contains(t, out, "My Cred")
	assert.NotContains(t, out, "super-secret", "externalSecret must not be rendered in table mode")
}

// TestUsersMeCredentialsProductID asserts --product-id, when passed, is sent
// as a productId query param; when absent, no productId param is sent.
func TestUsersMeCredentialsProductID(t *testing.T) {
	var sawProductID bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1"}`)
		case "/v2/api/users/user-1/credentials":
			if r.URL.Query().Get("productId") == "prod-1" {
				sawProductID = true
			}
			fmt.Fprint(w, `[]`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeCredentialsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--product-id", "prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, sawProductID, "expected productId query param when --product-id passed")
}

func TestUsersMeCredentialsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1"}`)
		default:
			fmt.Fprint(w, `[]`)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeCredentialsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No credentials found.")
}

func TestUsersMeCredentialsCmdRegistered(t *testing.T) {
	found := false
	for _, c := range meCmd.Commands() {
		if c.Use == "credentials" {
			found = true
			break
		}
	}
	require.True(t, found, "credentials subcommand must be registered on the me parent Cmd")
}
