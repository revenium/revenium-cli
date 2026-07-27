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

func TestMeterAudio(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/ai/audio", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "aud-123", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAudioCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "whisper-1",
		"--provider", "openai",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:00:10Z",
		"--request-duration", "10000",
		"--billing-unit", "PER_SECOND",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "aud-123")
	assert.Equal(t, "whisper-1", receivedBody["model"])
	assert.Equal(t, "openai", receivedBody["provider"])
	assert.Equal(t, "PER_SECOND", receivedBody["billingUnit"])
}

func TestMeterAudioWithOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "aud-456", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAudioCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "whisper-1",
		"--provider", "openai",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:00:10Z",
		"--request-duration", "10000",
		"--billing-unit", "PER_SECOND",
		"--duration-seconds", "120",
		"--language", "en",
		"--agentic-job-id", "loan-app-12345",
		"--agentic-job-version", "2.1.0",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, float64(120), receivedBody["durationSeconds"])
	assert.Equal(t, "en", receivedBody["language"])
	assert.Equal(t, "loan-app-12345", receivedBody["agenticJobId"])
	assert.Equal(t, "2.1.0", receivedBody["agenticJobVersion"])
}

func TestMeterAudioSquadAbsentWhenNotPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "aud-789", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAudioCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "whisper-1",
		"--provider", "openai",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:00:10Z",
		"--request-duration", "10000",
		"--billing-unit", "PER_SECOND",
	})
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, receivedBody, "squadId")
	assert.NotContains(t, receivedBody, "squadName")
	assert.NotContains(t, receivedBody, "squadRole")
}

func TestMeterAudioSquadPresentWhenPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "aud-790", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAudioCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--model", "whisper-1",
		"--provider", "openai",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:00:10Z",
		"--request-duration", "10000",
		"--billing-unit", "PER_SECOND",
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

func TestMeterAudioMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAudioCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--model", "whisper-1"})
	err := c.Execute()

	assert.Error(t, err)
}

func audioRequiredArgs() []string {
	return []string{
		"--model", "whisper-1",
		"--provider", "openai",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:00:10Z",
		"--request-duration", "10000",
		"--billing-unit", "PER_SECOND",
	}
}

func execAudioCmd(t *testing.T, args []string) map[string]interface{} {
	t.Helper()
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "aud-x", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAudioCmd()
	c.SetOut(&buf)
	c.SetArgs(args)
	err := c.Execute()
	require.NoError(t, err)

	return receivedBody
}

func TestMeterAudioNewOptionalFields(t *testing.T) {
	cases := []struct {
		flag     string
		args     []string
		bodyKey  string
		expected interface{}
	}{
		{"cost-type", []string{"--cost-type", "AI"}, "costType", "AI"},
		{"parent-transaction-id", []string{"--parent-transaction-id", "txn-parent-1"}, "parentTransactionId", "txn-parent-1"},
		{"transaction-name", []string{"--transaction-name", "my-transaction"}, "transactionName", "my-transaction"},
		{"retry-number", []string{"--retry-number", "3"}, "retryNumber", float64(3)},
		{"trace-type", []string{"--trace-type", "agentic"}, "traceType", "agentic"},
		{"trace-name", []string{"--trace-name", "trace-name-1"}, "traceName", "trace-name-1"},
		{"error-reason", []string{"--error-reason", "timeout"}, "errorReason", "timeout"},
		{"error-code", []string{"--error-code", "429"}, "errorCode", float64(429)},
		{"middleware-source", []string{"--middleware-source", "proxy-1"}, "middlewareSource", "proxy-1"},
		{"prompts-truncated", []string{"--prompts-truncated"}, "promptsTruncated", true},
		{"billing-skipped", []string{"--billing-skipped"}, "billingSkipped", true},
		{"skip-reason", []string{"--skip-reason", "FREE_TIER"}, "skipReason", "FREE_TIER"},
		{"pricing-tier", []string{"--pricing-tier", "STANDARD"}, "pricingTier", "STANDARD"},
		{"requested-service-tier", []string{"--requested-service-tier", "priority"}, "requestedServiceTier", "priority"},
		{"actual-service-tier", []string{"--actual-service-tier", "default"}, "actualServiceTier", "default"},
		{"output-response", []string{"--output-response", "some output text"}, "outputResponse", "some output text"},
		{"input-messages", []string{"--input-messages", `[{"role":"user","content":"hi"}]`}, "inputMessages", `[{"role":"user","content":"hi"}]`},
	}

	for _, tc := range cases {
		t.Run(tc.flag+"_absent", func(t *testing.T) {
			receivedBody := execAudioCmd(t, audioRequiredArgs())
			assert.NotContains(t, receivedBody, tc.bodyKey)
		})

		t.Run(tc.flag+"_present", func(t *testing.T) {
			args := append(append([]string{}, audioRequiredArgs()...), tc.args...)
			receivedBody := execAudioCmd(t, args)
			assert.Equal(t, tc.expected, receivedBody[tc.bodyKey])
		})
	}

	t.Run("subscriber_absent", func(t *testing.T) {
		receivedBody := execAudioCmd(t, audioRequiredArgs())
		assert.NotContains(t, receivedBody, "subscriber")
	})

	t.Run("subscriber-id_present", func(t *testing.T) {
		args := append(append([]string{}, audioRequiredArgs()...), "--subscriber-id", "sub-1")
		receivedBody := execAudioCmd(t, args)
		assert.Equal(t, map[string]interface{}{"id": "sub-1"}, receivedBody["subscriber"])
	})

	t.Run("subscriber-email_present", func(t *testing.T) {
		args := append(append([]string{}, audioRequiredArgs()...), "--subscriber-email", "test@example.com")
		receivedBody := execAudioCmd(t, args)
		assert.Equal(t, map[string]interface{}{"email": "test@example.com"}, receivedBody["subscriber"])
	})

	t.Run("subscriber-id-and-email_present", func(t *testing.T) {
		args := append(append([]string{}, audioRequiredArgs()...), "--subscriber-id", "sub-1", "--subscriber-email", "test@example.com")
		receivedBody := execAudioCmd(t, args)
		assert.Equal(t, map[string]interface{}{"id": "sub-1", "email": "test@example.com"}, receivedBody["subscriber"])
	})
}

func TestMeterAudioInputMessagesInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newAudioCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs(append(append([]string{}, audioRequiredArgs()...), "--input-messages", "{not json"))
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be valid JSON array")
}
