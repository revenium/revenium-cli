package alerts

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

// TestAlertsEvents is the load-bearing regression guard against Pitfall 5:
// it asserts the request path is the AI Alert resource
// (/v2/api/sources/ai/alert), NOT the anomaly-rule resource
// (/v2/api/sources/ai/anomaly) that the sibling alerts/list.go incorrectly
// calls.
func TestAlertsEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/alert", r.URL.Path)
		assert.NotEqual(t, "/v2/api/sources/ai/anomaly", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"alert-1","label":"High Cost","triggeredTimestamp":"2026-07-19T00:00:00Z","resolved":false,"triggeredValue":42.5}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "alert-1")
	assert.Contains(t, out, "High Cost")
}

// TestAlertsEventsResolvedFlagPresent asserts --resolved adds a `resolved`
// query param.
func TestAlertsEventsResolvedFlagPresent(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--resolved"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, gotQuery, "resolved=true")
}

// TestAlertsEventsResolvedFlagAbsent asserts omitting --resolved sends no
// `resolved` query param.
func TestAlertsEventsResolvedFlagAbsent(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.NotContains(t, gotQuery, "resolved")
}

// TestAlertsEventsEmpty asserts the empty-state phrase fires for a
// zero-result response.
func TestAlertsEventsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/alert", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No alert events found.")
}

// TestAlertsEventsEmptyJSON asserts --json mode on an empty fixture emits
// the canonical empty array `[]`.
func TestAlertsEventsEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newEventsCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Empty(t, result)
}

// TestAlertsEventsGet asserts the sibling get verb hits
// /v2/api/sources/ai/alert/{id}.
func TestAlertsEventsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/alert/alert-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"alert-1","label":"High Cost","triggeredTimestamp":"2026-07-19T00:00:00Z","resolved":false,"triggeredValue":42.5}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEventsGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"alert-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "alert-1")
}
