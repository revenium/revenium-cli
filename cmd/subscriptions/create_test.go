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

func TestCreateSubscription(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/subscriptions", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-new", "label": "New Sub", "description": "New sub"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--description", "New sub", "--subscriber-id", "sub-1", "--product-id", "prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "sub-new")
	assert.Equal(t, "New sub", receivedBody["description"])
	assert.Equal(t, "sub-1", receivedBody["subscriberId"])
	assert.Equal(t, "prod-1", receivedBody["productId"])
}

func TestCreateSubscriptionMinimal(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-new", "label": "", "description": "Minimal"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--description", "Minimal"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Minimal", receivedBody["description"])
	_, hasSubscriberID := receivedBody["subscriberId"]
	assert.False(t, hasSubscriberID, "subscriberId should not be sent when not specified")
	_, hasProductID := receivedBody["productId"]
	assert.False(t, hasProductID, "productId should not be sent when not specified")
}
