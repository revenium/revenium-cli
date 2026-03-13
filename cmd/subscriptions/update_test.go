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
