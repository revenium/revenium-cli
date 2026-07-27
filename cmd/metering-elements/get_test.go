package meteringelements

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

// TestMeteringElementsGet asserts GET
// /v2/api/metering-element-definitions/<id> renders the single-row 4-col
// table (ID / Name / Type / Description).
func TestMeteringElementsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/metering-element-definitions/me-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-1","name":"Tokens","type":"NUMBER","description":"Token count","_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"me-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "me-1")
	assert.Contains(t, out, "Tokens")
	assert.Contains(t, out, "NUMBER")
}

// TestMeteringElementsGetJSON asserts --json mode emits parseable JSON
// containing the metering element id and preserves the `_links` HATEOAS
// field (pass-through, no stripping).
func TestMeteringElementsGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/metering-element-definitions/me-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-1","name":"Tokens","type":"NUMBER","description":"Token count","_links":{"self":{"href":"/v2/api/metering-element-definitions/me-1"}}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"me-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "me-1", result["id"])
	assert.Equal(t, "Tokens", result["name"])
	assert.Contains(t, result, "_links")
}
