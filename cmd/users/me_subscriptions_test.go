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

// TestUsersMeSubscriptions asserts the two-call pattern: /v2/api/users/me
// first, then /v2/api/users/{id}/subscriptions, in that order.
func TestUsersMeSubscriptions(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1"}`)
		case "/v2/api/users/user-1/subscriptions":
			fmt.Fprint(w, `[{"id":"sub-1","name":"My Sub","product":{"label":"Widget Plan"}}]`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeSubscriptionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	require.Equal(t, []string{"/v2/api/users/me", "/v2/api/users/user-1/subscriptions"}, paths)
	out := buf.String()
	assert.Contains(t, out, "sub-1")
	assert.Contains(t, out, "Widget Plan")
}

// TestUsersMeSubscriptionsProductID asserts --product-id, when passed, is
// sent as a productId query param.
func TestUsersMeSubscriptionsProductID(t *testing.T) {
	var sawProductID bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1"}`)
		case "/v2/api/users/user-1/subscriptions":
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

	c := newMeSubscriptionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--product-id", "prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, sawProductID, "expected productId query param when --product-id passed")
}

func TestUsersMeSubscriptionsEmpty(t *testing.T) {
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

	c := newMeSubscriptionsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No subscriptions found.")
}

func TestUsersMeSubscriptionsCmdRegistered(t *testing.T) {
	found := false
	for _, c := range meCmd.Commands() {
		if c.Use == "subscriptions" {
			found = true
			break
		}
	}
	require.True(t, found, "subscriptions subcommand must be registered on the me parent Cmd")
}
