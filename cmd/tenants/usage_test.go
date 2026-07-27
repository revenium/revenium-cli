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

// TestGetTenantUsage asserts GET /v2/api/tenants/<id>/usage decodes a bare
// JSON string body (Pitfall 4 — the response schema is {"type":"string"},
// NOT an object) and prints it without an unmarshal error.
func TestGetTenantUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/tenants/ten-1/usage", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `"42 units"`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUsageCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"ten-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "42 units")
}

// TestGetTenantUsageJSON asserts --json mode round-trips the bare string
// through RenderJSON without attempting to decode it as a map.
func TestGetTenantUsageJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/tenants/ten-1/usage", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `"42 units"`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newUsageCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"ten-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "42 units", result)
}

// TestGetTenantUsageInvalidID asserts malformed ids are rejected before any
// HTTP call is made.
func TestGetTenantUsageInvalidID(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUsageCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"../etc/passwd"})
	err := c.Execute()

	require.Error(t, err)
	assert.False(t, called, "must not make an HTTP call for an invalid id")
}
