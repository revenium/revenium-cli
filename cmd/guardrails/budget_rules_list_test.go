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

// TestBudgetRulesList asserts the list verb hits the RESEARCH-corrected
// /v2/api/ai/cost-controls path and renders ID / Name / Status rows from the
// HATEOAS `_embedded.objectList` envelope. The "active" assertion is the
// load-bearing proof that boolStatus(r,"enabled") mapped enabled:true → "active"
// per RESEARCH D-03 — no "status" enum exists on CostControlResource_Read.
func TestBudgetRulesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/cost-controls", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// Verified HATEOAS fixture per RESEARCH §"Test fixture shape — verified".
		fmt.Fprint(w, `{
			"_embedded": {
				"objectList": [
					{"id":"jR2kmLs","name":"Q3 OpenAI Budget","enabled":true,"shadowMode":false,"action":"BLOCK","metricType":"TOTAL_COST","windowType":"MONTHLY","hardLimit":1000.0,"warnThreshold":800.0}
				]
			},
			"_links": {},
			"page": {"size": 20, "totalElements": 1, "totalPages": 1, "number": 0}
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "jR2kmLs")
	assert.Contains(t, out, "Q3 OpenAI Budget")
	// Load-bearing: proves boolStatus(r,"enabled") mapped true → "active" (RESEARCH D-03).
	assert.Contains(t, buf.String(), "active")
}

// TestBudgetRulesListEmpty asserts the empty-state phrase fires for the
// plain-array `[]` fixture (DoList tries plain decode first per client.go:372-374)
// and that NO table header is rendered.
func TestBudgetRulesListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No budget rules found.")
	assert.NotContains(t, buf.String(), "ID")
}

// TestBudgetRulesListJSON asserts --json mode emits parseable JSON containing
// the rule id. Uses the plain-array path for parity with jobs TestListJobsJSON.
func TestBudgetRulesListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"jR2kmLs","name":"Q3 OpenAI Budget","enabled":true,"shadowMode":false,"action":"BLOCK","metricType":"TOTAL_COST","windowType":"MONTHLY","hardLimit":1000.0,"warnThreshold":800.0}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newBudgetRulesListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "jR2kmLs")
	var parsed []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Len(t, parsed, 1)
	assert.Equal(t, "jR2kmLs", parsed[0]["id"])
	assert.Equal(t, "Q3 OpenAI Budget", parsed[0]["name"])
}

// TestBudgetRulesListEmptyJSON asserts --json mode on an empty fixture emits
// the canonical empty array `[]`.
func TestBudgetRulesListEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newBudgetRulesListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	var result []interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Empty(t, result)
}
