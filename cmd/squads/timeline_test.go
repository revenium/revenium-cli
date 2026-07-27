package squads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// populatedTimelineBody is a real-schema SquadTimelineResource_Read +
// SquadTimelineEvent_Read fixture (RESEARCH).
const populatedTimelineBody = `{
	"squadId": "squad-loan-proc-12345",
	"squadName": "Loan Processing",
	"startTime": "2025-09-01T00:00:00Z",
	"endTime": "2025-09-01T00:01:30Z",
	"totalDuration": 90000,
	"events": [
		{"id": "txn-1", "traceId": "trace-abc-123", "agent": "DocExtractor", "role": "Document Extraction",
		 "timestamp": 1693526400000, "startTime": "2025-09-01T00:00:00Z", "endTime": "2025-09-01T00:00:05Z", "duration": 5000,
		 "model": "gpt-4", "provider": "OpenAI", "operationType": "CHAT",
		 "inputTokens": 500, "outputTokens": 200, "totalTokens": 700, "cost": 0.0024}
	]
}`

// emptyTimelineBody preserves the wrapper shape with a zero-length events array.
const emptyTimelineBody = `{
	"squadId": "squad-loan-proc-12345",
	"squadName": "Loan Processing",
	"startTime": "2025-09-01T00:00:00Z",
	"endTime": "2025-09-01T00:00:00Z",
	"totalDuration": 0,
	"events": []
}`

func TestSquadsTimeline(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, populatedTimelineBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newTimelineCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/squads/squad-loan-proc-12345/timeline", gotPath)
	out := buf.String()
	assert.Contains(t, out, "DocExtractor")
	assert.Contains(t, out, "Document Extraction")
	assert.Contains(t, out, "gpt-4")
}

func TestSquadsTimelineRejectsPathTraversal(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, populatedTimelineBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newTimelineCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"../etc"})
	err := c.Execute()

	require.Error(t, err)
	assert.Equal(t, 0, requestCount)
}

func TestSquadsTimelineWithPeriod(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, populatedTimelineBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = "SEVEN_DAYS"
	defer func() { periodFlag = "" }()

	c := newTimelineCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "SEVEN_DAYS", gotQuery, "--period must thread onto timeline identically to get")
}

func TestSquadsTimelineJSONPreservesWrapper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, populatedTimelineBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	periodFlag = ""

	c := newTimelineCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "JSON output must decode as a wrapper object (map), not a bare array")
	require.Contains(t, result, "events")
	require.Contains(t, result, "squadName")
}

func TestSquadsTimelineEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, emptyTimelineBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newTimelineCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No timeline events found.")
}

func TestSquadsTimelineEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, emptyTimelineBody)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	periodFlag = ""

	c := newTimelineCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"squad-loan-proc-12345"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "empty state must preserve the wrapper object, not emit bare []")
	require.Contains(t, result, "events")
	eventsAny, ok := result["events"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, eventsAny)
}
