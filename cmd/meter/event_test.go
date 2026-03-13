package meter

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

func TestMeterEvent(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/events", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "evt-123", "resourceType": "meter.event", "label": "meter.event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--transaction-id", "txn-123", "--payload", `{"apiCalls": 100}`})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "evt-123")
	assert.Equal(t, "txn-123", receivedBody["transactionId"])
	payload := receivedBody["payload"].(map[string]interface{})
	assert.Equal(t, float64(100), payload["apiCalls"])
}

func TestMeterEventWithOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "evt-456", "resourceType": "meter.event", "label": "meter.event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--transaction-id", "txn-456", "--payload", `{"storageGB": 15.5}`, "--source-id", "src-789", "--subscriber-credential", "cred-abc"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "src-789", receivedBody["sourceId"])
	assert.Equal(t, "cred-abc", receivedBody["subscriberCredential"])
}

func TestMeterEventInvalidPayload(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--transaction-id", "txn-123", "--payload", "not-json"})
	err := c.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--payload must be valid JSON")
}

func TestMeterEventMissingTransactionID(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--payload", `{"apiCalls": 100}`})
	err := c.Execute()

	assert.Error(t, err)
}

func TestMeterEventJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "evt-123", "resourceType": "meter.event", "label": "meter.event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newEventCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--transaction-id", "txn-123", "--payload", `{"apiCalls": 100}`})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "evt-123", result["id"])
}
