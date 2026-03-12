package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.example.com", "test-key", true)

	assert.Equal(t, "https://api.example.com", c.BaseURL)
	assert.Equal(t, "test-key", c.APIKey)
	assert.True(t, c.Verbose)
	assert.NotNil(t, c.HTTPClient)
}

func TestClientSetsAuthHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-api-key")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-secret-key", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "my-secret-key", gotHeader)
}

func TestClientSetsContentType(t *testing.T) {
	var gotContentType, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "application/json", gotAccept)
}

func TestClientSetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(gotUA, "revenium-cli/"), "User-Agent should start with revenium-cli/, got: %s", gotUA)
}

func TestClientTimeout(t *testing.T) {
	c := NewClient("https://api.example.com", "key", false)
	assert.Equal(t, 30*time.Second, c.HTTPClient.Timeout)
}

func TestErrorMapping401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "bad-key", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
	assert.Contains(t, err.Error(), "revenium config set key")
}

func TestErrorMapping403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"forbidden"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Access denied")
}

func TestErrorMapping404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Resource not found")
}

func TestErrorMapping500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Revenium API error")
}

func TestSuccessfulRequest(t *testing.T) {
	type result struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result{Name: "test-source", ID: 42})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", false)
	var got result
	err := c.Do(context.Background(), http.MethodGet, "/sources/42", nil, &got)

	require.NoError(t, err)
	assert.Equal(t, "test-source", got.Name)
	assert.Equal(t, 42, got.ID)
}

func TestVerboseLogging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	c := NewClient(srv.URL, "my-api-key-1234", true)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/test")
	assert.Contains(t, output, "200")
	// API key should be masked in verbose output
	assert.NotContains(t, output, "my-api-key-1234")
	assert.Contains(t, output, "1234") // last 4 chars shown
}
