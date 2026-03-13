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

func TestMeterVideo(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/ai/video", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "vid-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVideoCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "veo",
		"--provider", "google",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:01:00Z",
		"--request-duration", "60000",
		"--duration-seconds", "10",
		"--billing-unit", "PER_SECOND",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "vid-123")
	assert.Equal(t, "veo", receivedBody["model"])
	assert.Equal(t, "google", receivedBody["provider"])
	assert.Equal(t, float64(10), receivedBody["durationSeconds"])
	assert.Equal(t, "PER_SECOND", receivedBody["billingUnit"])
}

func TestMeterVideoMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVideoCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--model", "veo"})
	err := c.Execute()

	assert.Error(t, err)
}
