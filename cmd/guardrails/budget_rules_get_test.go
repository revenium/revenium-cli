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

// TestBudgetRulesGet asserts GET /v2/api/ai/cost-controls/<id> renders the
// single-row 3-col table (ID / Name / Status). The "active" assertion is the
// load-bearing proof that renderRule's boolStatus(rule,"enabled") mapping
// matches RESEARCH D-03 (no "status" field on CostControlResource_Read).
func TestBudgetRulesGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"Q3 OpenAI Budget","enabled":true,"shadowMode":false,"action":"BLOCK","metricType":"TOTAL_COST","windowType":"MONTHLY","hardLimit":1000.0,"warnThreshold":800.0,"_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "jR2kmLs")
	assert.Contains(t, out, "Q3 OpenAI Budget")
	assert.Contains(t, out, "active")
}

// TestBudgetRulesGetJSON asserts --json mode emits parseable JSON containing
// the rule id and preserves the `_links` HATEOAS field (pass-through per
// RESEARCH §"HATEOAS").
func TestBudgetRulesGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"Q3 OpenAI Budget","enabled":true,"shadowMode":false,"action":"BLOCK","metricType":"TOTAL_COST","windowType":"MONTHLY","hardLimit":1000.0,"warnThreshold":800.0,"_links":{"self":{"href":"/v2/api/ai/cost-controls/jR2kmLs"}}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newBudgetRulesGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "jR2kmLs", result["id"])
	assert.Equal(t, "Q3 OpenAI Budget", result["name"])
	// HATEOAS pass-through (RESEARCH §"HATEOAS").
	assert.Contains(t, result, "_links")
}

// TestBudgetRulesGetWithFilters asserts that filters and notificationChannelIds
// surface in NON-JSON (table-mode) output so users can SEE the rule's scope
// without flipping into --json. The literal-string assertions are the proof
// that renderRule's secondary block iterates and prints the filter triples
// and channel IDs verbatim.
func TestBudgetRulesGetWithFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/cost-controls/jR2kmLs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id":"jR2kmLs",
			"name":"Q3 OpenAI Budget",
			"enabled":true,
			"filters":[{"dimension":"MODEL","operator":"IS","value":"gpt-4"}],
			"notificationChannelIds":["chan-1","chan-2"],
			"_links":{}
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs"})
	require.NoError(t, c.Execute())

	out := buf.String()
	// Existing contract — 3-col table renders the ID/Name/Status row.
	assert.Contains(t, out, "jR2kmLs")
	assert.Contains(t, out, "Q3 OpenAI Budget")
	assert.Contains(t, out, "active")
	// New contract — filters and notification channels are visible in non-JSON mode.
	assert.Contains(t, out, "MODEL", "filter dimension must appear in non-JSON output")
	assert.Contains(t, out, "IS", "filter operator must appear in non-JSON output")
	assert.Contains(t, out, "gpt-4", "filter value must appear in non-JSON output")
	assert.Contains(t, out, "chan-1", "notification channel id must appear in non-JSON output")
	assert.Contains(t, out, "chan-2", "notification channel id must appear in non-JSON output")
}

// TestBudgetRulesGetEmptyFiltersStaysSilent asserts that absent or empty
// filters/notificationChannelIds DO NOT produce an empty "Filters:" or
// "Notification channels:" header — the renderer must stay silent when there
// is nothing to show. This protects the original 3-col table contract for
// rules that have no scope or channels.
func TestBudgetRulesGetEmptyFiltersStaysSilent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// No filters key, empty notificationChannelIds.
		fmt.Fprint(w, `{"id":"x","name":"Silent","enabled":true,"notificationChannelIds":[],"_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"x"})
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.NotContains(t, out, "Filters:", "no Filters: header when field is absent")
	assert.NotContains(t, out, "Notification channels:", "no Notification channels: header when array is empty")
}

// TestBudgetRulesGetJSONWithFilters asserts that --json mode preserves
// filters and notificationChannelIds through the map passthrough — proof
// that the secondary table-mode rendering does not contaminate JSON output.
func TestBudgetRulesGetJSONWithFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id":"jR2kmLs",
			"name":"Q3 OpenAI Budget",
			"enabled":true,
			"filters":[{"dimension":"MODEL","operator":"IS","value":"gpt-4"}],
			"notificationChannelIds":["chan-1","chan-2"],
			"_links":{"self":{"href":"/v2/api/ai/cost-controls/jR2kmLs"}}
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newBudgetRulesGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"jR2kmLs"})
	require.NoError(t, c.Execute())

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	// Both new fields round-trip through the map passthrough.
	require.Contains(t, result, "filters")
	require.Contains(t, result, "notificationChannelIds")
	rawFilters, ok := result["filters"].([]interface{})
	require.True(t, ok)
	require.Len(t, rawFilters, 1)
	f0 := rawFilters[0].(map[string]interface{})
	assert.Equal(t, "MODEL", f0["dimension"])
	assert.Equal(t, "gpt-4", f0["value"])
	rawChans, ok := result["notificationChannelIds"].([]interface{})
	require.True(t, ok)
	require.Len(t, rawChans, 2)
	assert.Equal(t, "chan-1", rawChans[0])
}
