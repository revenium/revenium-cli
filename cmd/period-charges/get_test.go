package periodcharges

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

// TestPeriodChargesGet asserts GET /v2/api/period-charges/<id> issues exactly
// one request and renders the single-row 3-col table (ID / Amount / Currency).
// This is a normal single-object Do — the cursor landmine is list-only.
func TestPeriodChargesGet(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, "/v2/api/period-charges/pc-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"pc-1","amount":10.5,"currency":"USD"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"pc-1"})
	err := c.Execute()

	require.NoError(t, err)
	require.Equal(t, 1, requestCount, "get must issue exactly one request")
	out := buf.String()
	assert.Contains(t, out, "pc-1")
	assert.Contains(t, out, "10.5")
	assert.Contains(t, out, "USD")
}

// TestPeriodChargesGetJSON asserts --json mode emits parseable JSON containing
// the period charge id.
func TestPeriodChargesGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/period-charges/pc-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"pc-1","amount":10.5,"currency":"USD"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"pc-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "pc-1", result["id"])
}

// TestPeriodChargesGetInvalidID asserts a malformed id is rejected by
// ValidResourceID before any HTTP call is made.
func TestPeriodChargesGetInvalidID(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
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
	assert.False(t, requestMade, "malformed id must be rejected before any HTTP call")
}

func TestPeriodChargesGetRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Use == "get <id>" {
			found = true
			break
		}
	}
	require.True(t, found, "get subcommand must be registered on the period-charges parent Cmd")
}
