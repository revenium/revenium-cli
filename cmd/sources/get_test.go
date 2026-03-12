package sources

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

func TestGetSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/src-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "src-1", "name": "My API", "type": "API", "status": "active"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"src-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "src-1")
	assert.Contains(t, out, "My API")
	assert.Contains(t, out, "API")
	assert.Contains(t, out, "active")
}

func TestGetSourceJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "src-1", "name": "My API", "type": "API", "status": "active"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"src-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "src-1", result["id"])
	assert.Equal(t, "My API", result["name"])
}

func TestGetSourceMissingArg(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	err := c.Execute()

	assert.Error(t, err)
}
