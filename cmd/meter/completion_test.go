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

func TestMeterCompletion(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/ai/completions", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "gpt-4",
		"--provider", "openai",
		"--input-tokens", "500",
		"--output-tokens", "200",
		"--total-tokens", "700",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--is-streamed",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "cmp-123")
	assert.Equal(t, "gpt-4", receivedBody["model"])
	assert.Equal(t, "openai", receivedBody["provider"])
	assert.Equal(t, float64(500), receivedBody["inputTokenCount"])
	assert.Equal(t, float64(200), receivedBody["outputTokenCount"])
	assert.Equal(t, float64(700), receivedBody["totalTokenCount"])
	assert.Equal(t, "END", receivedBody["stopReason"])
	assert.Equal(t, true, receivedBody["isStreamed"])
}

func TestMeterCompletionWithOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-456", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "claude-3-opus",
		"--provider", "anthropic",
		"--input-tokens", "1000",
		"--output-tokens", "500",
		"--total-tokens", "1500",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:10Z",
		"--request-duration", "10000",
		"--is-streamed",
		"--total-cost", "0.045",
		"--trace-type", "agentic",
		"--agent", "my-agent",
		"--environment", "production",
		"--agentic-job-id", "loan-app-12345",
		"--agentic-job-name", "Process Loan",
		"--agentic-job-type", "loan-processing",
		"--agentic-job-version", "2.1.0",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, 0.045, receivedBody["totalCost"])
	assert.Equal(t, "agentic", receivedBody["traceType"])
	assert.Equal(t, "my-agent", receivedBody["agent"])
	assert.Equal(t, "production", receivedBody["environment"])
	assert.Equal(t, "loan-app-12345", receivedBody["agenticJobId"])
	assert.Equal(t, "Process Loan", receivedBody["agenticJobName"])
	assert.Equal(t, "loan-processing", receivedBody["agenticJobType"])
	assert.Equal(t, "2.1.0", receivedBody["agenticJobVersion"])
}

func TestMeterCompletionSquadAbsentWhenNotPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-789", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "gpt-4",
		"--provider", "openai",
		"--input-tokens", "500",
		"--output-tokens", "200",
		"--total-tokens", "700",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--is-streamed",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, receivedBody, "squadId")
	assert.NotContains(t, receivedBody, "squadName")
	assert.NotContains(t, receivedBody, "squadRole")
}

func TestMeterCompletionSquadPresentWhenPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-790", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "gpt-4",
		"--provider", "openai",
		"--input-tokens", "500",
		"--output-tokens", "200",
		"--total-tokens", "700",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--is-streamed",
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

func TestMeterCompletionSkillFieldsAbsentWhenNotPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-792", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs(requiredCompletionArgs())
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, receivedBody, "skillName")
	assert.NotContains(t, receivedBody, "skillInvocationTrigger")
	assert.NotContains(t, receivedBody, "skillSource")
	assert.NotContains(t, receivedBody, "skillKind")
	assert.NotContains(t, receivedBody, "skillPluginName")
	assert.NotContains(t, receivedBody, "skillMarketplaceName")
}

func TestMeterCompletionSkillFieldsPresentWhenPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-791", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	args := append(requiredCompletionArgs(),
		"--skill-name", "code-reviewer",
		"--skill-invocation-trigger", "explicit",
		"--skill-source", "marketplace",
		"--skill-kind", "plugin",
		"--skill-plugin-name", "review-plugin",
		"--skill-marketplace-name", "anthropic-marketplace",
	)
	c.SetArgs(args)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "code-reviewer", receivedBody["skillName"])
	assert.Equal(t, "explicit", receivedBody["skillInvocationTrigger"])
	assert.Equal(t, "marketplace", receivedBody["skillSource"])
	assert.Equal(t, "plugin", receivedBody["skillKind"])
	assert.Equal(t, "review-plugin", receivedBody["skillPluginName"])
	assert.Equal(t, "anthropic-marketplace", receivedBody["skillMarketplaceName"])
}

func TestMeterCompletionMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--model", "gpt-4"})
	err := c.Execute()

	assert.Error(t, err)
}

// requiredCompletionArgs returns the base set of required flags shared by
// every TestMeterCompletionNewOptionalFields subtest.
func requiredCompletionArgs() []string {
	return []string{
		"--model", "gpt-4",
		"--provider", "openai",
		"--input-tokens", "500",
		"--output-tokens", "200",
		"--total-tokens", "700",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--is-streamed",
	}
}

func TestMeterCompletionNewOptionalFields(t *testing.T) {
	cases := []struct {
		flag     string
		args     []string
		bodyKey  string
		expected interface{}
	}{
		{"response-quality-score", []string{"--response-quality-score", "0.87"}, "responseQualityScore", 0.87},
		{"cache-creation-token-cost", []string{"--cache-creation-token-cost", "0.02"}, "cacheCreationTokenCost", 0.02},
		{"cache-read-token-cost", []string{"--cache-read-token-cost", "0.01"}, "cacheReadTokenCost", 0.01},
		{"cost-type", []string{"--cost-type", "AI"}, "costType", "AI"},
		{"mediation-latency", []string{"--mediation-latency", "250"}, "mediationLatency", float64(250)},
		{"system-fingerprint", []string{"--system-fingerprint", "fp_44709d6fcb"}, "systemFingerprint", "fp_44709d6fcb"},
		{"error-reason", []string{"--error-reason", "rate limited"}, "errorReason", "rate limited"},
		{"error-code", []string{"--error-code", "429"}, "errorCode", float64(429)},
		{"middleware-source", []string{"--middleware-source", "litellm"}, "middlewareSource", "litellm"},
		{"operation-subtype", []string{"--operation-subtype", "streaming"}, "operationSubtype", "streaming"},
		{"parent-transaction-id", []string{"--parent-transaction-id", "txn-parent-1"}, "parentTransactionId", "txn-parent-1"},
		{"transaction-name", []string{"--transaction-name", "checkout-flow"}, "transactionName", "checkout-flow"},
		{"retry-number", []string{"--retry-number", "2"}, "retryNumber", float64(2)},
		{"trace-name", []string{"--trace-name", "agent-run-7"}, "traceName", "agent-run-7"},
		{"prompts-truncated", []string{"--prompts-truncated"}, "promptsTruncated", true},
		{"billing-skipped", []string{"--billing-skipped"}, "billingSkipped", true},
		{"skip-reason", []string{"--skip-reason", "FREE_TIER"}, "skipReason", "FREE_TIER"},
		{"pricing-tier", []string{"--pricing-tier", "STANDARD"}, "pricingTier", "STANDARD"},
		{"requested-service-tier", []string{"--requested-service-tier", "priority"}, "requestedServiceTier", "priority"},
		{"actual-service-tier", []string{"--actual-service-tier", "default"}, "actualServiceTier", "default"},
		{"subscription-tier", []string{"--subscription-tier", "pro"}, "subscriptionTier", "pro"},
		{"cost-multiplier", []string{"--cost-multiplier", "1.5"}, "costMultiplier", 1.5},
		{"coding-assistant-account-uuid", []string{"--coding-assistant-account-uuid", "uuid-123"}, "codingAssistantAccountUuid", "uuid-123"},
		{"cache-creation-5m-tokens", []string{"--cache-creation-5m-tokens", "10"}, "cacheCreation5mTokenCount", float64(10)},
		{"cache-creation-1h-tokens", []string{"--cache-creation-1h-tokens", "20"}, "cacheCreation1hTokenCount", float64(20)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.flag+"_absent", func(t *testing.T) {
			var receivedBody map[string]interface{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &receivedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, `{"id": "cmp-abs", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
			}))
			defer srv.Close()

			var buf bytes.Buffer
			cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			c := newCompletionCmd()
			c.SetOut(&buf)
			c.SetArgs(requiredCompletionArgs())
			err := c.Execute()

			require.NoError(t, err)
			assert.NotContains(t, receivedBody, tc.bodyKey)
		})

		t.Run(tc.flag+"_present", func(t *testing.T) {
			var receivedBody map[string]interface{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &receivedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, `{"id": "cmp-pres", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
			}))
			defer srv.Close()

			var buf bytes.Buffer
			cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			c := newCompletionCmd()
			c.SetOut(&buf)
			args := append(requiredCompletionArgs(), tc.args...)
			c.SetArgs(args)
			err := c.Execute()

			require.NoError(t, err)
			assert.Equal(t, tc.expected, receivedBody[tc.bodyKey])
		})
	}
}

func TestMeterCompletionSubscriberAbsentWhenNotPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-sub-abs", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs(requiredCompletionArgs())
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, receivedBody, "subscriber")
}

func TestMeterCompletionSubscriberPresentWhenPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-sub-pres", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	args := append(requiredCompletionArgs(), "--subscriber-id", "sub-1", "--subscriber-email", "sub@example.com")
	c.SetArgs(args)
	err := c.Execute()

	require.NoError(t, err)
	require.Contains(t, receivedBody, "subscriber")
	subscriber, ok := receivedBody["subscriber"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "sub-1", subscriber["id"])
	assert.Equal(t, "sub@example.com", subscriber["email"])
}

func TestMeterCompletionJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "cmp-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newCompletionCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "gpt-4",
		"--provider", "openai",
		"--input-tokens", "500",
		"--output-tokens", "200",
		"--total-tokens", "700",
		"--stop-reason", "END",
		"--request-time", "2024-01-15T10:00:00Z",
		"--completion-start-time", "2024-01-15T10:00:01Z",
		"--response-time", "2024-01-15T10:00:05Z",
		"--request-duration", "5000",
		"--is-streamed",
	})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "cmp-123", result["id"])
}
