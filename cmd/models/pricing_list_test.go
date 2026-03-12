package models

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

func TestPricingList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/mdl-1/pricing/dimensions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"id": "dim-1", "name": "Input Tokens", "dimensionType": "input", "unitPrice": "0.003"},
			{"id": "dim-2", "name": "Output Tokens", "dimensionType": "output", "unitPrice": "0.006"}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "dim-1")
	assert.Contains(t, out, "Input Tokens")
	assert.Contains(t, out, "input")
	assert.Contains(t, out, "0.003")
	assert.Contains(t, out, "dim-2")
	assert.Contains(t, out, "Output Tokens")
}

func TestPricingListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No pricing dimensions found.")
}

func TestPricingListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"id": "dim-1", "name": "Input Tokens", "dimensionType": "input", "unitPrice": "0.003"}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newPricingListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "dim-1", result[0]["id"])
}

func TestPricingListEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newPricingListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestPricingListVerifyPath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"my-model-id"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, receivedPath, "my-model-id")
	assert.Equal(t, "/v2/api/sources/ai/models/my-model-id/pricing/dimensions", receivedPath)
}
