package models

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

func TestPricingCreate(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "dim-new", "billingUnit": "PER_TOKEN", "modality": "TEXT", "costType": "TEXT_TOKEN_INPUT", "unitPrice": 0.003}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "--billing-unit", "PER_TOKEN", "--modality", "TEXT", "--cost-type", "TEXT_TOKEN_INPUT", "--price", "0.003"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "PER_TOKEN", receivedBody["billingUnit"])
	assert.Equal(t, "TEXT", receivedBody["modality"])
	assert.Equal(t, "TEXT_TOKEN_INPUT", receivedBody["costType"])
	assert.Equal(t, 0.003, receivedBody["unitPrice"])
	assert.Contains(t, buf.String(), "dim-new")
}

func TestPricingCreateVerifyPath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "dim-new", "billingUnit": "PER_TOKEN", "modality": "TEXT", "costType": "TEXT_TOKEN_INPUT", "unitPrice": 0.003}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"my-model-id", "--billing-unit", "PER_TOKEN", "--modality", "TEXT", "--cost-type", "TEXT_TOKEN_INPUT", "--price", "0.001"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/sources/ai/models/my-model-id/pricing/dimensions", receivedPath)
}
