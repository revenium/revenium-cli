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

// TestUsersMeInvoices asserts the two-call pattern: /v2/api/users/me first,
// then /v2/api/users/{id}/invoices, in that order.
func TestUsersMeInvoices(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1"}`)
		case "/v2/api/users/user-1/invoices":
			fmt.Fprint(w, `[{"id":"inv-1","invoiceNumber":"INV-001","state":"PAID","totalAmount":42.5,"currency":"USD"}]`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMeInvoicesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	require.Equal(t, []string{"/v2/api/users/me", "/v2/api/users/user-1/invoices"}, paths)
	out := buf.String()
	assert.Contains(t, out, "inv-1")
	assert.Contains(t, out, "INV-001")
}

func TestUsersMeInvoicesEmpty(t *testing.T) {
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

	c := newMeInvoicesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No invoices found.")
}

func TestUsersMeInvoicesCmdRegistered(t *testing.T) {
	found := false
	for _, c := range meCmd.Commands() {
		if c.Use == "invoices" {
			found = true
			break
		}
	}
	require.True(t, found, "invoices subcommand must be registered on the me parent Cmd")
}
