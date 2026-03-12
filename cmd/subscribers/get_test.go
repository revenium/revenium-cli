package subscribers

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

func TestGetSubscriber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/subscribers/sub-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1", "email": "john@example.com", "firstName": "John", "lastName": "Doe"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "sub-1")
	assert.Contains(t, out, "John Doe")
	assert.Contains(t, out, "john@example.com")
}

func TestGetSubscriberJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1", "email": "john@example.com", "firstName": "John", "lastName": "Doe"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "sub-1", result["id"])
	assert.Equal(t, "john@example.com", result["email"])
}
