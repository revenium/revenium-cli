package meteringelements

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestMeteringElementsUpdate asserts the update verb does a GET-then-PUT
// round-trip against /v2/api/metering-element-definitions/<id>
// (DoUpdate's GET-merge-PUT semantics).
func TestMeteringElementsUpdate(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/metering-element-definitions/me-1", r.URL.Path)
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "me-1",
				"name": "Old Name",
				"type": "STRING",
			})
			return
		}
		assert.Equal(t, "PUT", r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-1","name":"New Name","type":"STRING"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"me-1", "--name", "New Name"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "New Name")
	assert.Equal(t, "New Name", receivedBody["name"])
}

// TestMeteringElementsUpdateNoFields asserts omitting all update flags
// errors "no fields specified to update".
func TestMeteringElementsUpdateNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"me-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}

// TestMeteringElementsUpdateInvalidType asserts an invalid --type value is
// rejected client-side with NO HTTP request issued (T-06-13 mitigation on
// the update path).
func TestMeteringElementsUpdateInvalidType(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"me-1", "--type", "BOGUS"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
}
