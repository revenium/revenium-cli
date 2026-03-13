package alerts

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

func TestCreateAlert(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/sources/ai/anomaly", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "anom-new", "label": "High Cost Alert", "created": "2026-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "High Cost Alert", "--threshold", "100"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "anom-new")
	assert.Equal(t, "High Cost Alert", receivedBody["name"])
	assert.Equal(t, "THRESHOLD", receivedBody["alertType"])
	assert.Equal(t, "DAILY", receivedBody["periodDuration"])
	assert.Equal(t, true, receivedBody["enabled"])
	assert.Equal(t, false, receivedBody["firing"])
}
