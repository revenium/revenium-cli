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

// TestEnforcementRulesGet exercises the happy-path render of a populated rules[] payload.
// Fixture sourced from RESEARCH §"Test fixture shape — verified" (For enforcement-rules get).
func TestEnforcementRulesGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/enforcement-rules/team-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"compiledAt":"2026-05-12T18:42:15Z","rules":[{"ruleId":12345,"name":"Q3 OpenAI Budget","action":"BLOCK","shadowMode":false,"metricType":"TOTAL_COST","threshold":1000.0,"currentValue":423.5,"percentUsed":42.35}]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEnforcementRulesGetCmd()
	c.SetArgs([]string{"team-1"})
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	out := buf.String()
	// Metadata header line per RESEARCH D-11
	assert.Contains(t, out, "Compiled at: 2026-05-12T18:42:15Z")
	// Multi-row table contents
	assert.Contains(t, out, "12345")
	assert.Contains(t, out, "Q3 OpenAI Budget")
	assert.Contains(t, out, "BLOCK")
	// shadowMode=false → "enforce" per renderEnforcementRules logic
	assert.Contains(t, out, "enforce")
}

// TestEnforcementRulesGetEmpty exercises the empty-rules branch — metadata header still renders,
// table header does NOT render, "No enforcement rules compiled." phrase appears.
func TestEnforcementRulesGetEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/enforcement-rules/team-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"compiledAt":"2026-05-12T18:42:15Z","rules":[]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newEnforcementRulesGetCmd()
	c.SetArgs([]string{"team-1"})
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "No enforcement rules compiled.")
	// Table header MUST NOT render when rules[] is empty
	assert.NotContains(t, out, "Rule ID")
	// Metadata header still renders even when rules[] is empty (per RESEARCH D-11)
	assert.Contains(t, out, "Compiled at:")
}

// TestEnforcementRulesGetJSON exercises the --json passthrough — full payload (compiledAt + rules)
// preserved verbatim per RESEARCH D-11.
func TestEnforcementRulesGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"compiledAt":"2026-05-12T18:42:15Z","rules":[{"ruleId":12345,"name":"Q3 OpenAI Budget","action":"BLOCK","shadowMode":false,"metricType":"TOTAL_COST","threshold":1000.0,"currentValue":423.5,"percentUsed":42.35}]}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newEnforcementRulesGetCmd()
	c.SetArgs([]string{"team-1"})
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	// Full payload pass-through — both compiledAt and rules keys present
	_, hasCompiledAt := parsed["compiledAt"]
	_, hasRules := parsed["rules"]
	assert.True(t, hasCompiledAt, "JSON output must contain compiledAt key")
	assert.True(t, hasRules, "JSON output must contain rules key")
}
