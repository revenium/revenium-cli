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

func TestMeterImage(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/ai/images", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "img-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newImageCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "dall-e-3",
		"--provider", "openai",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--actual-image-count", "1",
		"--billing-unit", "PER_IMAGE",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "img-123")
	assert.Equal(t, "dall-e-3", receivedBody["model"])
	assert.Equal(t, "openai", receivedBody["provider"])
	assert.Equal(t, float64(1), receivedBody["actualImageCount"])
	assert.Equal(t, "PER_IMAGE", receivedBody["billingUnit"])
}

func TestMeterImageMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newImageCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--model", "dall-e-3"})
	err := c.Execute()

	assert.Error(t, err)
}
