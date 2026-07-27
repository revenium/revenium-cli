package tenants

import (
	"bytes"
	"encoding/json"
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

// TestGetTenant asserts GET /v2/api/tenants/<id> renders the object via
// renderTenant.
func TestGetTenant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/tenants/ten-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"ten-1","name":"Acme Tenant","status":"ACTIVE"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"ten-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "ten-1")
	assert.Contains(t, out, "Acme Tenant")
}

// TestGetTenantJSON asserts --json mode emits parseable JSON containing the
// tenant id.
func TestGetTenantJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/tenants/ten-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"ten-1","name":"Acme Tenant","status":"ACTIVE"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"ten-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "ten-1", result["id"])
	assert.Equal(t, "Acme Tenant", result["name"])
}

// TestGetTenantInvalidID asserts malformed ids are rejected by
// cmd.ValidResourceID before any HTTP call is made.
func TestGetTenantInvalidID(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"../etc/passwd"})
	err := c.Execute()

	require.Error(t, err)
	assert.False(t, called, "must not make an HTTP call for an invalid id")
}
