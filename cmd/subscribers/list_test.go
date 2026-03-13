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

func TestListSubscribers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/subscribers", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"id": "sub-1", "email": "john@example.com", "firstName": "John", "lastName": "Doe"},
			{"id": "sub-2", "email": "jane@example.com", "firstName": "Jane", "lastName": "Smith"}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "sub-1")
	assert.Contains(t, out, "John Doe")
	assert.Contains(t, out, "john@example.com")
	assert.Contains(t, out, "sub-2")
	assert.Contains(t, out, "Jane Smith")
	assert.Contains(t, out, "jane@example.com")
}

func TestListSubscribersEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No subscribers found.")
}

func TestListSubscribersJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"id": "sub-1", "email": "john@example.com", "firstName": "John", "lastName": "Doe"}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	// Verify output is valid JSON array
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "sub-1", result[0]["id"])
}

func TestListSubscribersEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	// Verify output is valid empty JSON array
	var result []interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}
