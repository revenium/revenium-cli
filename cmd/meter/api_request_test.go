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

func TestMeterAPIRequest(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/apis/requests", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "req-123", "resourceType": "metered.api.request", "label": "metered.api.request", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAPIRequestCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--transaction-id", "txn-123", "--method", "POST", "--resource", "/api/users"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "req-123")
	assert.Equal(t, "txn-123", receivedBody["transactionId"])
	assert.Equal(t, "POST", receivedBody["method"])
	assert.Equal(t, "/api/users", receivedBody["resource"])
}

func TestMeterAPIRequestMissingTransactionID(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAPIRequestCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--method", "GET"})
	err := c.Execute()

	assert.Error(t, err)
}
