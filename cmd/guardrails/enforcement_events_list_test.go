package guardrails

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

// enforcementEventsFixture is the verified RESEARCH fixture (For enforcement-events list).
// _embedded.objectList shape exercises DoList's HATEOAS unwrap.
const enforcementEventsFixture = `{
  "_embedded": {
    "objectList": [
      {
        "ruleName": "Q3 OpenAI Budget",
        "action": "BLOCK",
        "metricType": "TOTAL_COST",
        "currentValue": 1023.5,
        "threshold": 1000.0,
        "usagePercent": 102.35,
        "tenantId": "5jLgdv",
        "created": "2026-05-12T18:42:15Z"
      }
    ]
  },
  "_links": {},
  "page": {"size": 20, "totalElements": 1, "totalPages": 1, "number": 0}
}`

// TestEnforcementEventsList exercises the happy-path 5-col render against the verified fixture.
func TestEnforcementEventsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/enforcement-events", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, enforcementEventsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEnforcementEventsListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "Q3 OpenAI Budget") // Rule column
	assert.Contains(t, out, "BLOCK")            // Action column (StatusColumn 1)
	assert.Contains(t, out, "TOTAL_COST")       // Metric column
	// Composite "Used" column (currentValue/threshold) from toEnforcementEventRows
	assert.Contains(t, out, "1023.5/1000")
	// Time column sourced via eventTime helper's first candidate "created"
	assert.Contains(t, out, "2026-05-12T18:42:15Z")
}

// TestEnforcementEventsListEmpty exercises the empty-list branch — plain JSON [] from server,
// "No enforcement events found." phrase rendered in non-JSON mode.
func TestEnforcementEventsListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEnforcementEventsListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	assert.Contains(t, buf.String(), "No enforcement events found.")
}

// TestEnforcementEventsListJSON exercises the --json passthrough — list payload preserved verbatim.
func TestEnforcementEventsListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, enforcementEventsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newEnforcementEventsListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	assert.Contains(t, buf.String(), "Q3 OpenAI Budget")
	// Confirm parseable JSON
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "Q3 OpenAI Budget", parsed[0]["ruleName"])
}

// TestEnforcementEventsListSinceFilter proves the --since flag value is URL-encoded into the
// request URL as ?since=... before DoList adds page/size (RESEARCH D-12).
func TestEnforcementEventsListSinceFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/enforcement-events", r.URL.Path)
		assert.Equal(t, "2026-05-01T00:00:00Z", r.URL.Query().Get("since"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, enforcementEventsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEnforcementEventsListCmd()
	c.SetArgs([]string{"--since", "2026-05-01T00:00:00Z"})
	c.SetOut(&buf)
	require.NoError(t, c.Execute())
}

// TestEnforcementEventsListRuleIDFilter proves the --rule-id flag value is propagated as the
// camelCase ?ruleId=... query param per OAS (NOT "rule-id" — RESEARCH D-12).
func TestEnforcementEventsListRuleIDFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/enforcement-events", r.URL.Path)
		assert.Equal(t, "jR2kmLs", r.URL.Query().Get("ruleId"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, enforcementEventsFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEnforcementEventsListCmd()
	c.SetArgs([]string{"--rule-id", "jR2kmLs"})
	c.SetOut(&buf)
	require.NoError(t, c.Execute())
}
