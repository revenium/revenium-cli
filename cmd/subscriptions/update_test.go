package subscriptions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSubscriptionPUT(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v2/api/subscriptions/sub-1", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1", "label": "Updated", "description": "Updated"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1", "--description", "Updated"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Updated")
	assert.Equal(t, "Updated", receivedBody["description"])
}

func TestUpdateSubscriptionPATCH(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/v2/api/subscriptions/sub-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1", "label": "Partial", "description": "Partial"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1", "--patch", "--description", "Partial"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Partial")
}

func TestUpdateSubscriptionNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"sub-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}

func TestUpdateSubscriptionPATCHPartialBody(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1", "label": "Only this", "description": "Only this"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1", "--patch", "--description", "Only this"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Only this", receivedBody["description"])
	_, hasSubscriberID := receivedBody["subscriberId"]
	assert.False(t, hasSubscriberID, "subscriberId should not be in partial body")
	_, hasProductID := receivedBody["productId"]
	assert.False(t, hasProductID, "productId should not be in partial body")
}
