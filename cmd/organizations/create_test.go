package organizations

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

// TestCreateOrganization verifies that `revenium organizations create --name X`
// POSTs to /v2/api/organizations with a body containing only "name" — proving
// that the c.Flags().Changed gating omits the optional --external-id and
// --parent-id fields when not supplied (RESEARCH D-05 LOCKED contract).
func TestCreateOrganization(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/organizations", r.URL.Path)
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))

		// Required field present with the CLI-passed value.
		assert.Equal(t, "Acme Corporation", receivedBody["name"])

		// Optional fields MUST be absent (proves Flags().Changed gating).
		_, hasExternal := receivedBody["externalId"]
		assert.False(t, hasExternal, "externalId must not be sent when --external-id not provided")
		_, hasParent := receivedBody["parentId"]
		assert.False(t, hasParent, "parentId must not be sent when --parent-id not provided")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"org-new","name":"Acme Corporation","label":"Acme Corporation","_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "Acme Corporation"})

	require.NoError(t, c.Execute())

	// renderOrg should surface the created resource's id.
	assert.Contains(t, buf.String(), "org-new")
}

// TestCreateOrganizationMissingRequired is the LOAD-BEARING ORGS-03 assertion:
// omitting --name must error at the Cobra MarkFlagRequired layer BEFORE any HTTP
// round-trip. Configures an unreachable BaseURL so an accidental HTTP attempt
// would surface a connection error instead of the expected Cobra one.
func TestCreateOrganizationMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	// Deliberately omit --name; pass --external-id to make sure the failure
	// is specifically about the required flag, not "no args".
	c.SetArgs([]string{"--external-id", "ext-1"})

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name",
		"missing-flag error must name 'name' so users know what to add")
}

// TestCreateOrganizationDryRun pins the dry-run output contract by invoking
// dryrun.Render DIRECTLY with the exact path/verb/resource/body shape that
// create.go's RunE passes when cmd.DryRun() is true.
//
// Precedent: cmd/guardrails/budget_rules_create_test.go TestBudgetRulesCreateDryRun
// — cmd.DryRun() has no exported setter, so we test the dry-run render contract
// rather than the global toggle. The branch itself is a trivial single `if`;
// the integration is covered by manual smoke per the phase VALIDATION doc.
func TestCreateOrganizationDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact body create.go's RunE constructs when only --name is supplied.
	body := map[string]interface{}{
		"name": "Acme Corporation",
	}

	err := dryrun.Render(out, "create", "organization", "/v2/api/organizations", body)
	require.NoError(t, err)

	rendered := buf.String()
	// Header line — verifies verb + resource words survive the dryrun format.
	assert.Contains(t, rendered, "Dry run: create organization")
	// Path round-trip.
	assert.Contains(t, rendered, "/v2/api/organizations")
	// Footer — matches internal/dryrun/dryrun.go.
	assert.Contains(t, rendered, "No changes were made.")
}

// TestCreateOrganizationOptionalFlags is the inverse of TestCreateOrganization:
// when --external-id and --parent-id ARE passed, those keys MUST appear in the
// request body with the supplied values. Proves c.Flags().Changed gating
// correctly distinguishes "user passed the flag" from "flag default".
func TestCreateOrganizationOptionalFlags(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/organizations", r.URL.Path)
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))

		// Required field still present (sanity).
		assert.Equal(t, "Acme Corporation", receivedBody["name"])
		// Optional fields ARE present this time with the supplied values.
		assert.Equal(t, "ext-123", receivedBody["externalId"])
		assert.Equal(t, "parent-org-456", receivedBody["parentId"])

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"org-new2","name":"Acme Corporation","label":"Acme Corporation","_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{
		"--name", "Acme Corporation",
		"--external-id", "ext-123",
		"--parent-id", "parent-org-456",
	})

	require.NoError(t, c.Execute())
	assert.Contains(t, buf.String(), "org-new2")
}
