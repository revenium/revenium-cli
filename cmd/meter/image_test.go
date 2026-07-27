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
		"--agentic-job-id", "loan-app-12345",
		"--agentic-job-type", "loan-processing",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "img-123")
	assert.Equal(t, "dall-e-3", receivedBody["model"])
	assert.Equal(t, "openai", receivedBody["provider"])
	assert.Equal(t, float64(1), receivedBody["actualImageCount"])
	assert.Equal(t, "PER_IMAGE", receivedBody["billingUnit"])
	assert.Equal(t, "loan-app-12345", receivedBody["agenticJobId"])
	assert.Equal(t, "loan-processing", receivedBody["agenticJobType"])
}

func TestMeterImageSquadAbsentWhenNotPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "img-789", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
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
	assert.NotContains(t, receivedBody, "squadId")
	assert.NotContains(t, receivedBody, "squadName")
	assert.NotContains(t, receivedBody, "squadRole")
}

func TestMeterImageSquadPresentWhenPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "img-790", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
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
		"--squad-id", "sq-1",
		"--squad-name", "Alpha",
		"--squad-role", "planner",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "sq-1", receivedBody["squadId"])
	assert.Equal(t, "Alpha", receivedBody["squadName"])
	assert.Equal(t, "planner", receivedBody["squadRole"])
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

var requiredImageArgs = []string{
	"--model", "dall-e-3",
	"--provider", "openai",
	"--request-time", "2024-01-15T10:00:00Z",
	"--response-time", "2024-01-15T10:00:05Z",
	"--request-duration", "5000",
	"--actual-image-count", "1",
	"--billing-unit", "PER_IMAGE",
}

func runImageCmdCapturingBody(t *testing.T, args []string) map[string]interface{} {
	t.Helper()
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "img-999", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newImageCmd()
	c.SetOut(&buf)
	fullArgs := append(append([]string{}, requiredImageArgs...), args...)
	c.SetArgs(fullArgs)
	err := c.Execute()
	require.NoError(t, err)

	return receivedBody
}

func TestMeterImageNewOptionalFields(t *testing.T) {
	cases := []struct {
		flag     string
		args     []string
		bodyKey  string
		expected interface{}
	}{
		{"cost-type", []string{"--cost-type", "AI"}, "costType", "AI"},
		{"parent-transaction-id", []string{"--parent-transaction-id", "ptx-1"}, "parentTransactionId", "ptx-1"},
		{"transaction-name", []string{"--transaction-name", "my-txn"}, "transactionName", "my-txn"},
		{"trace-type", []string{"--trace-type", "agentic"}, "traceType", "agentic"},
		{"trace-name", []string{"--trace-name", "my-trace"}, "traceName", "my-trace"},
		{"error-reason", []string{"--error-reason", "timeout"}, "errorReason", "timeout"},
		{"error-code", []string{"--error-code", "429"}, "errorCode", float64(429)},
		{"retry-number", []string{"--retry-number", "2"}, "retryNumber", float64(2)},
		{"middleware-source", []string{"--middleware-source", "proxy-1"}, "middlewareSource", "proxy-1"},
		{"source-transaction-id", []string{"--source-transaction-id", "src-tx-1"}, "sourceTransactionId", "src-tx-1"},
		{"skip-reason", []string{"--skip-reason", "FREE_TIER"}, "skipReason", "FREE_TIER"},
		{"pricing-tier", []string{"--pricing-tier", "STANDARD"}, "pricingTier", "STANDARD"},
		{"requested-service-tier", []string{"--requested-service-tier", "priority"}, "requestedServiceTier", "priority"},
		{"actual-service-tier", []string{"--actual-service-tier", "default"}, "actualServiceTier", "default"},
		{"priority-tier", []string{"--priority-tier", "on_demand"}, "priorityTier", "on_demand"},
		{"revised-prompt-provided", []string{"--revised-prompt-provided"}, "revisedPromptProvided", true},
		{"prompts-truncated", []string{"--prompts-truncated"}, "promptsTruncated", true},
		{"billing-skipped", []string{"--billing-skipped"}, "billingSkipped", true},
		{"output-response", []string{"--output-response", "the response text"}, "outputResponse", "the response text"},
		{"input-messages", []string{"--input-messages", `[{"role":"user","content":"hi"}]`}, "inputMessages", `[{"role":"user","content":"hi"}]`},
		{"subscriber-id", []string{"--subscriber-id", "sub-1"}, "subscriber", map[string]interface{}{"id": "sub-1"}},
		{"subscriber-email", []string{"--subscriber-email", "a@b.com"}, "subscriber", map[string]interface{}{"email": "a@b.com"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.flag+"_absent", func(t *testing.T) {
			receivedBody := runImageCmdCapturingBody(t, nil)
			assert.NotContains(t, receivedBody, tc.bodyKey)
		})
		t.Run(tc.flag+"_present", func(t *testing.T) {
			receivedBody := runImageCmdCapturingBody(t, tc.args)
			assert.Equal(t, tc.expected, receivedBody[tc.bodyKey])
		})
	}
}

func TestMeterImageInputMessagesInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newImageCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	fullArgs := append(append([]string{}, requiredImageArgs...), "--input-messages", "{not json")
	c.SetArgs(fullArgs)
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be valid JSON array")
}
