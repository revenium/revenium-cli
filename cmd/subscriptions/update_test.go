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
		assert.Equal(t, "/v2/api/subscriptions/sub-1", r.URL.Path)
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "sub-1",
				"description": "Old description",
			})
			return
		}
		assert.Equal(t, "PUT", r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1", "label": "Updated", "description": "Updated"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
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

// TestEmailPrecedenceUpdate asserts that when both flags are passed, --subscriber-email
// wins (D-03): subscriberEmail is sent, clientEmailAddress is NOT.
func TestEmailPrecedenceUpdate(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "sub-1"})
			return
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs([]string{
		"sub-1",
		"--subscriber-email", "primary@example.com",
		"--client-email", "deprecated@example.com",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "primary@example.com", receivedBody["subscriberEmail"])
	_, hasClientEmail := receivedBody["clientEmailAddress"]
	assert.False(t, hasClientEmail, "clientEmailAddress must NOT be sent when --subscriber-email is also present")
	assert.Contains(t, errBuf.String(), "deprecated", "warning still fires because --client-email was used")
}

// TestUpdateSubscriptionBackfillsEmailWhenNeitherFlagPassed is the D-05
// relocation regression test: a subscriptions update with NEITHER
// --subscriber-email nor --client-email, against a GET response that has a
// nested "client" object + "label" and no flat email keys, must still send
// an email field (subscriberEmail) in the outgoing PUT body — not blank it.
func TestUpdateSubscriptionBackfillsEmailWhenNeitherFlagPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/subscriptions/sub-1", r.URL.Path)
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "sub-1",
				"label":  "x@y.com",
				"client": map[string]interface{}{"id": "client-1"},
			})
			return
		}
		assert.Equal(t, "PUT", r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1", "description": "Updated"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1", "--description", "Updated"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "x@y.com", receivedBody["subscriberEmail"], "email must be backfilled from the derived label, not blanked")
}

// TestUpdateSubscriptionExplicitEmailSkipsBackfill asserts an explicit
// --subscriber-email is sent as-is with no backfill needed and no
// clientEmailAddress sent.
func TestUpdateSubscriptionExplicitEmailSkipsBackfill(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "sub-1",
				"label":  "old@example.com",
				"client": map[string]interface{}{"id": "client-1"},
			})
			return
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"sub-1", "--subscriber-email", "new@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "new@example.com", receivedBody["subscriberEmail"])
	_, hasClientEmail := receivedBody["clientEmailAddress"]
	assert.False(t, hasClientEmail)
}

// TestClientEmailDeprecationWarningUpdate asserts the deprecation warning lands
// only in stderr, never stdout, so JSON-mode stdout stays parseable (D-04).
func TestClientEmailDeprecationWarningUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "sub-1"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newUpdateCmd()
	var outBuf, errBuf bytes.Buffer
	c.SetOut(&outBuf)
	c.SetErr(&errBuf)
	c.SetArgs([]string{"sub-1", "--client-email", "deprecated@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, errBuf.String(), "--client-email is deprecated")
	assert.NotContains(t, buf.String(), "deprecated", "deprecation warning must never appear on stdout/JSON output")
}
