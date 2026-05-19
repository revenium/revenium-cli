package guardrails

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestBudgetRulesCreate verifies that `revenium guardrails budget-rules create`
// POSTs the 8 OAS-required fields to /v2/api/ai/cost-controls and OMITS the 2
// optional fields (shadowMode, enabled) when their flags were not passed. This
// is the canonical proof of the GRDR-03 contract (RESEARCH D-10).
func TestBudgetRulesCreate(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/ai/cost-controls", r.URL.Path)
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &received))

		// All 8 required keys present with the exact CLI-passed values.
		assert.Equal(t, "Q3 OpenAI Budget", received["name"])
		assert.Equal(t, "Caps monthly OpenAI spend", received["description"])
		assert.Equal(t, "TOTAL_COST", received["metricType"])
		assert.Equal(t, "MONTHLY", received["windowType"])
		assert.Equal(t, "BLOCK", received["action"])
		assert.Equal(t, "MODEL", received["groupBy"])
		assert.Equal(t, 800.0, received["warnThreshold"])
		assert.Equal(t, 1000.0, received["hardLimit"])

		// Optional fields must NOT be present (proves Flags().Changed gating).
		_, hasShadow := received["shadowMode"]
		assert.False(t, hasShadow, "shadowMode must not be sent when --shadow-mode not provided")
		_, hasEnabled := received["enabled"]
		assert.False(t, hasEnabled, "enabled must not be sent when --enabled not provided")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"Q3 OpenAI Budget","enabled":true,"_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--name", "Q3 OpenAI Budget",
		"--description", "Caps monthly OpenAI spend",
		"--metric-type", "TOTAL_COST",
		"--window-type", "MONTHLY",
		"--action", "BLOCK",
		"--group-by", "MODEL",
		"--warn-threshold", "800",
		"--hard-limit", "1000",
	})

	require.NoError(t, c.Execute())

	// renderRule should print the created rule's id.
	assert.Contains(t, buf.String(), "jR2kmLs")
}

// TestBudgetRulesCreateMissingRequired is the LOAD-BEARING GRDR-03 assertion:
// omitting any one of the 8 required flags must error at the Cobra
// MarkFlagRequired layer BEFORE any HTTP round-trip. This prevents
// under-specified creates from producing server-side 400 spam (T-14-03-05).
//
// Per RESEARCH §"Required test files", we omit --hard-limit.
func TestBudgetRulesCreateMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	// Configure an unreachable URL — if the test accidentally issues HTTP,
	// the dial would fail with a different error than the Cobra one we expect.
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesCreateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	// Deliberately omit --hard-limit (one of the 8 required flags).
	c.SetArgs([]string{
		"--name", "X",
		"--description", "Y",
		"--metric-type", "TOTAL_COST",
		"--window-type", "MONTHLY",
		"--action", "BLOCK",
		"--group-by", "MODEL",
		"--warn-threshold", "800",
	})

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hard-limit",
		"missing-flag error must name 'hard-limit' so users know what to add")
}

// TestBudgetRulesCreateDryRun pins the dry-run output contract by invoking
// dryrun.Render DIRECTLY with the exact path/verb/resource/body shape that
// budget_rules_create.go's RunE passes in the dry-run branch.
//
// Precedent: cmd/jobs/outcome_test.go TestOutcomeDryRun — the cmd.DryRun()
// global flag has no exported setter, so we test the dry-run render contract
// instead of the cmd.DryRun() toggle. The branch itself is a structurally
// trivial single `if`; its integration is covered by manual smoke per
// 14-VALIDATION.md.
func TestBudgetRulesCreateDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact body budget_rules_create.go's RunE constructs when all 8
	// required flags are supplied and no optional flags are passed.
	body := map[string]interface{}{
		"name":          "Q3 OpenAI Budget",
		"description":   "Caps monthly OpenAI spend",
		"metricType":    "TOTAL_COST",
		"windowType":    "MONTHLY",
		"action":        "BLOCK",
		"groupBy":       "MODEL",
		"warnThreshold": 800.0,
		"hardLimit":     1000.0,
	}

	err := dryrun.Render(out, "create", "budget rule", "/v2/api/ai/cost-controls", body)
	require.NoError(t, err)

	rendered := buf.String()
	// Header line — verifies verb + resource words survive the dryrun format.
	assert.Contains(t, rendered, "Dry run: create budget rule")
	// Path round-trip.
	assert.Contains(t, rendered, "/v2/api/ai/cost-controls")
	// At least one body key visible (proves body payload was rendered).
	assert.Contains(t, rendered, "metricType")
	// Footer — matches internal/dryrun/dryrun.go.
	assert.Contains(t, rendered, "No changes were made.")
}

// TestBudgetRulesCreateOptionalFlags confirms the inverse of TestBudgetRulesCreate:
// when --shadow-mode and --enabled ARE passed, those keys MUST appear in the
// request body with the supplied values. Proves c.Flags().Changed gating
// correctly distinguishes "user passed the flag" from "default value".
func TestBudgetRulesCreateOptionalFlags(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/ai/cost-controls", r.URL.Path)
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &received))

		// Optional flags ARE present this time, with the supplied values.
		assert.Equal(t, true, received["shadowMode"])
		assert.Equal(t, false, received["enabled"])

		// Required fields still present — sanity check that gating optional
		// fields didn't accidentally remove required ones.
		assert.Equal(t, "Q3 OpenAI Budget", received["name"])
		assert.Equal(t, 1000.0, received["hardLimit"])

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"jR2kmLs","name":"Q3 OpenAI Budget","enabled":false,"_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetRulesCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--name", "Q3 OpenAI Budget",
		"--description", "Caps monthly OpenAI spend",
		"--metric-type", "TOTAL_COST",
		"--window-type", "MONTHLY",
		"--action", "BLOCK",
		"--group-by", "MODEL",
		"--warn-threshold", "800",
		"--hard-limit", "1000",
		"--shadow-mode",
		"--enabled=false",
	})

	require.NoError(t, c.Execute())
	assert.Contains(t, buf.String(), "jR2kmLs")
}
