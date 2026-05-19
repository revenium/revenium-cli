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
