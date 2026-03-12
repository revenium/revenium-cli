package charts

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

func TestCreateChart(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/reports/chart-definitions", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "chart-1", "label": "Revenue Chart", "type": "bar", "created": "2026-01-01"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--label", "Revenue Chart", "--type", "bar", "--description", "A revenue chart"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "chart-1")
	assert.Equal(t, "Revenue Chart", receivedBody["label"])
	assert.Equal(t, "bar", receivedBody["type"])
	assert.Equal(t, "A revenue chart", receivedBody["description"])
}

func TestCreateChartMinimal(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "chart-1", "label": "Revenue Chart", "created": "2026-01-01"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--label", "Revenue Chart"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Revenue Chart", receivedBody["label"])
	_, hasType := receivedBody["type"]
	assert.False(t, hasType, "type should not be sent when not specified")
	_, hasDescription := receivedBody["description"]
	assert.False(t, hasDescription, "description should not be sent when not specified")
}
