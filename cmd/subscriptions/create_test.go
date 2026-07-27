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
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "API Access", "--client-email", "user@example.com", "--description", "New sub", "--subscriber-id", "sub-1", "--product-id", "prod-1"})
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
		fmt.Fprint(w, `{"id": "sub-new", "label": "", "description": ""}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "Minimal Sub", "--client-email", "user@example.com", "--product-id", "prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "prod-1", receivedBody["productId"])
	_, hasSubscriberID := receivedBody["subscriberId"]
	assert.False(t, hasSubscriberID, "subscriberId should not be sent when not specified")
	_, hasDescription := receivedBody["description"]
	assert.False(t, hasDescription, "description should not be sent when not specified")
}

// TestCreateSubscriptionSubscriberEmailPrimary asserts --subscriber-email alone
// sends subscriberEmail and no clientEmailAddress (D-02).
func TestCreateSubscriptionSubscriberEmailPrimary(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-new"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs([]string{"--name", "API Access", "--subscriber-email", "primary@example.com", "--product-id", "prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "primary@example.com", receivedBody["subscriberEmail"])
	_, hasClientEmail := receivedBody["clientEmailAddress"]
	assert.False(t, hasClientEmail, "clientEmailAddress should not be sent when --subscriber-email is used")
	assert.Empty(t, errBuf.String(), "no deprecation warning expected when only --subscriber-email is used")
}

// TestCreateSubscriptionMissingEmail asserts create requires one of the two email flags.
func TestCreateSubscriptionMissingEmail(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--name", "API Access", "--product-id", "prod-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "one of --subscriber-email or --client-email is required")
}

// TestEmailPrecedenceCreate asserts that when both flags are passed, --subscriber-email
// wins (D-03): subscriberEmail is sent, clientEmailAddress is NOT.
func TestEmailPrecedenceCreate(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-new"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs([]string{
		"--name", "API Access",
		"--subscriber-email", "primary@example.com",
		"--client-email", "deprecated@example.com",
		"--product-id", "prod-1",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "primary@example.com", receivedBody["subscriberEmail"])
	_, hasClientEmail := receivedBody["clientEmailAddress"]
	assert.False(t, hasClientEmail, "clientEmailAddress must NOT be sent when --subscriber-email is also present")
	assert.Contains(t, errBuf.String(), "deprecated", "warning still fires because --client-email was used")
}

// TestClientEmailDeprecationWarningCreate asserts the deprecation warning lands
// only in stderr, never stdout, so JSON-mode stdout stays parseable (D-04).
func TestClientEmailDeprecationWarningCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-new"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newCreateCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs([]string{"--name", "API Access", "--client-email", "deprecated@example.com", "--product-id", "prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, errBuf.String(), "--client-email is deprecated")
	assert.NotContains(t, buf.String(), "deprecated", "deprecation warning must never appear on stdout/JSON output")
}
