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
		"--agentic-job-id", "loan-app-12345",
		"--agentic-job-name", "Process Loan",
	})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "vid-123")
	assert.Equal(t, "veo", receivedBody["model"])
	assert.Equal(t, "google", receivedBody["provider"])
	assert.Equal(t, float64(10), receivedBody["durationSeconds"])
	assert.Equal(t, "PER_SECOND", receivedBody["billingUnit"])
	assert.Equal(t, "loan-app-12345", receivedBody["agenticJobId"])
	assert.Equal(t, "Process Loan", receivedBody["agenticJobName"])
}

func TestMeterVideoSquadAbsentWhenNotPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "vid-789", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
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
	assert.NotContains(t, receivedBody, "squadId")
	assert.NotContains(t, receivedBody, "squadName")
	assert.NotContains(t, receivedBody, "squadRole")
}

func TestMeterVideoSquadPresentWhenPassed(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "vid-790", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
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

// requiredVideoArgs are the required flags every `meter video` invocation needs,
// independent of the optional fields under test.
func requiredVideoArgs() []string {
	return []string{
		"--model", "veo",
		"--provider", "google",
		"--request-time", "2024-01-15T10:00:00Z",
		"--response-time", "2024-01-15T10:01:00Z",
		"--request-duration", "60000",
		"--duration-seconds", "10",
		"--billing-unit", "PER_SECOND",
	}
}

// runMeterVideo executes `meter video` with the given extra args and returns the
// captured request body along with any execution error.
func runMeterVideo(t *testing.T, extraArgs ...string) (map[string]interface{}, error) {
	t.Helper()
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "vid-new", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVideoCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	args := append(requiredVideoArgs(), extraArgs...)
	c.SetArgs(args)
	err := c.Execute()
	return receivedBody, err
}

func TestMeterVideoNewOptionalFields(t *testing.T) {
	cases := []struct {
		flag     string
		args     []string
		bodyKey  string
		expected interface{}
	}{
		{"cost-type", []string{"--cost-type", "AI"}, "costType", "AI"},
		{"parent-transaction-id", []string{"--parent-transaction-id", "ptx-1"}, "parentTransactionId", "ptx-1"},
		{"transaction-name", []string{"--transaction-name", "txn-name"}, "transactionName", "txn-name"},
		{"retry-number", []string{"--retry-number", "3"}, "retryNumber", float64(3)},
		{"trace-type", []string{"--trace-type", "distributed"}, "traceType", "distributed"},
		{"trace-name", []string{"--trace-name", "trace-1"}, "traceName", "trace-1"},
		{"error-reason", []string{"--error-reason", "timeout"}, "errorReason", "timeout"},
		{"error-code", []string{"--error-code", "429"}, "errorCode", float64(429)},
		{"middleware-source", []string{"--middleware-source", "proxy"}, "middlewareSource", "proxy"},
		{"source-transaction-id", []string{"--source-transaction-id", "src-1"}, "sourceTransactionId", "src-1"},
		{"output-response", []string{"--output-response", "done"}, "outputResponse", "done"},
		{"prompts-truncated", []string{"--prompts-truncated"}, "promptsTruncated", true},
		{"billing-skipped", []string{"--billing-skipped"}, "billingSkipped", true},
		{"skip-reason", []string{"--skip-reason", "FREE_TIER"}, "skipReason", "FREE_TIER"},
		{"pricing-tier", []string{"--pricing-tier", "STANDARD"}, "pricingTier", "STANDARD"},
		{"requested-service-tier", []string{"--requested-service-tier", "gold"}, "requestedServiceTier", "gold"},
		{"actual-service-tier", []string{"--actual-service-tier", "gold"}, "actualServiceTier", "gold"},
		{"priority-tier", []string{"--priority-tier", "on_demand"}, "priorityTier", "on_demand"},
	}

	for _, tc := range cases {
		t.Run(tc.flag+"_absent", func(t *testing.T) {
			receivedBody, err := runMeterVideo(t)
			require.NoError(t, err)
			assert.NotContains(t, receivedBody, tc.bodyKey)
		})
		t.Run(tc.flag+"_present", func(t *testing.T) {
			receivedBody, err := runMeterVideo(t, tc.args...)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, receivedBody[tc.bodyKey])
		})
	}
}

func TestMeterVideoInputMessagesField(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		receivedBody, err := runMeterVideo(t)
		require.NoError(t, err)
		assert.NotContains(t, receivedBody, "inputMessages")
	})
	t.Run("present", func(t *testing.T) {
		raw := `[{"role":"user","content":"hi"}]`
		receivedBody, err := runMeterVideo(t, "--input-messages", raw)
		require.NoError(t, err)
		// Stored as the RAW string, not the parsed array — spec type is "string".
		assert.Equal(t, raw, receivedBody["inputMessages"])
	})
}

func TestMeterVideoInputMessagesInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVideoCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	args := append(requiredVideoArgs(), "--input-messages", "{not json")
	c.SetArgs(args)

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be valid JSON array")
}

func TestMeterVideoSubscriberField(t *testing.T) {
	t.Run("absent_when_neither_flag_passed", func(t *testing.T) {
		receivedBody, err := runMeterVideo(t)
		require.NoError(t, err)
		assert.NotContains(t, receivedBody, "subscriber")
	})

	t.Run("present_with_id_only", func(t *testing.T) {
		receivedBody, err := runMeterVideo(t, "--subscriber-id", "sub-123")
		require.NoError(t, err)
		require.Contains(t, receivedBody, "subscriber")
		subscriber, ok := receivedBody["subscriber"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "sub-123", subscriber["id"])
		assert.NotContains(t, subscriber, "email")
	})

	t.Run("present_with_email_only", func(t *testing.T) {
		receivedBody, err := runMeterVideo(t, "--subscriber-email", "sub@example.com")
		require.NoError(t, err)
		require.Contains(t, receivedBody, "subscriber")
		subscriber, ok := receivedBody["subscriber"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "sub@example.com", subscriber["email"])
		assert.NotContains(t, subscriber, "id")
	})

	t.Run("present_with_both", func(t *testing.T) {
		receivedBody, err := runMeterVideo(t, "--subscriber-id", "sub-123", "--subscriber-email", "sub@example.com")
		require.NoError(t, err)
		require.Contains(t, receivedBody, "subscriber")
		subscriber, ok := receivedBody["subscriber"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "sub-123", subscriber["id"])
		assert.Equal(t, "sub@example.com", subscriber["email"])
		// credential sub-fields are deliberately excluded (D-08)
		assert.NotContains(t, subscriber, "credential")
	})
}
