package models

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
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestCreateModel verifies that `models create` with only required flags
// POSTs to /v2/api/sources/ai/models with exactly the 5 required keys — no
// optional keys present (proves c.Flags().Changed gating omits them).
func TestCreateModel(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/sources/ai/models", r.URL.Path)
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))

		assert.Equal(t, "GPT-4", receivedBody["name"])
		assert.Equal(t, "chat", receivedBody["mode"])
		assert.Equal(t, "OpenAI", receivedBody["provider"])
		assert.Equal(t, 0.00003, receivedBody["inputCostPerToken"])
		assert.Equal(t, 0.00006, receivedBody["outputCostPerToken"])

		_, hasVision := receivedBody["supportsVision"]
		assert.False(t, hasVision, "supportsVision must not be sent when --supports-vision not provided")
		assert.Len(t, receivedBody, 5, "body must contain exactly the 5 required fields")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"mdl-new","name":"GPT-4","provider":"OpenAI","mode":"chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--name", "GPT-4",
		"--mode", "chat",
		"--provider", "OpenAI",
		"--input-cost-per-token", "0.00003",
		"--output-cost-per-token", "0.00006",
	})

	require.NoError(t, c.Execute())
	assert.Contains(t, buf.String(), "mdl-new")
}

// TestCreateModelOptionalFlags is the inverse of TestCreateModel: passing an
// optional flag (--supports-vision) must add supportsVision to the body.
func TestCreateModelOptionalFlags(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))

		assert.Equal(t, true, receivedBody["supportsVision"])
		assert.Equal(t, 0.000001, receivedBody["cacheReadCostPerInputToken"])

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"mdl-new2","name":"GPT-4V","provider":"OpenAI","mode":"chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--name", "GPT-4V",
		"--mode", "chat",
		"--provider", "OpenAI",
		"--input-cost-per-token", "0.00003",
		"--output-cost-per-token", "0.00006",
		"--supports-vision",
		"--cache-read-cost-per-input-token", "0.000001",
	})

	require.NoError(t, c.Execute())
	assert.Contains(t, buf.String(), "mdl-new2")
}

// TestCreateModelDryRun pins the dry-run render contract directly, mirroring
// cmd/organizations/create_test.go's TestCreateOrganizationDryRun approach
// (cmd.DryRun() has no exported setter).
func TestCreateModelDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	body := map[string]interface{}{
		"name":               "GPT-4",
		"mode":               "chat",
		"provider":           "OpenAI",
		"inputCostPerToken":  0.00003,
		"outputCostPerToken": 0.00006,
	}

	err := dryrun.Render(out, "create", "model", "/v2/api/sources/ai/models", body)
	require.NoError(t, err)

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: create model")
	assert.Contains(t, rendered, "/v2/api/sources/ai/models")
	assert.Contains(t, rendered, "No changes were made.")
}

// TestCreateModelMissingRequired asserts that omitting a required flag
// errors at the Cobra MarkFlagRequired layer BEFORE any HTTP round-trip.
func TestCreateModelMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	// Deliberately omit --name.
	c.SetArgs([]string{
		"--mode", "chat",
		"--provider", "OpenAI",
		"--input-cost-per-token", "0.00003",
		"--output-cost-per-token", "0.00006",
	})

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

// TestCreateModelMutatingAnnotation guards the mutating annotation contract.
func TestCreateModelMutatingAnnotation(t *testing.T) {
	c := newCreateCmd()
	assert.Equal(t, "true", c.Annotations["mutating"])
}
