package skills

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// newPeriodValidatingRoot builds a throwaway parent command that supplies a
// fresh, non-singleton root for the query-shape tests, so each can drive an
// isolated newListCmd()/newGetCmd() instance without mutating the real
// package-level Cmd between tests.
//
// This is NOT the coverage of record for the T-22-01 --period gate, and must
// not be read as such: it is a look-alike, and a mitigation deleted from
// skills.go would leave it green. The gate is covered against the shipped Cmd
// by TestSkillsCmdRejectsInvalidPeriod and TestSkillsCmdDelegatesToAttachedRoot
// in cmd_test.go, both of which fail if cmd.ValidatePeriod(periodFlag) is
// weakened. What survives here is only the isolation these query-shape tests
// want; the period validation it performs is incidental to that.
//
// It lives here rather than in list_test.go only because get_test.go is the
// first file in the package to need it; it is package-scoped and both test
// files use it.
func newPeriodValidatingRoot(sub *cobra.Command) *cobra.Command {
	root := &cobra.Command{
		Use:           "skills",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			return cmd.ValidatePeriod(periodFlag)
		},
	}
	root.PersistentFlags().StringVar(&periodFlag, "period", "", "period")
	root.AddCommand(sub)
	return root
}

// fixtureSkillDetail is a real-schema SkillUsageResource_Read object: the
// flat (non-HATEOAS) body getSkillDetail returns, carrying all 13 verified
// properties from the cached platform spec. Unlike the list envelope this is
// NOT wrapped in `_embedded`, which is why the command uses Do, not DoList.
const fixtureSkillDetail = `{
	"id": "JMwX9g4",
	"resourceType": "skill",
	"name": "code-review",
	"originCategory": "VENDOR",
	"source": "bundled",
	"kind": "workflow",
	"pluginName": "revenium-tools",
	"marketplaceName": "anthropics",
	"totalCost": 12.47,
	"callCount": 342,
	"traceCount": 57,
	"firstSeen": "2026-06-01T12:00:00Z",
	"lastSeen": "2026-06-29T18:30:00Z"
}`

// fixtureSkillDetailNulls is the same object with every property the spec
// marks nullable set to JSON null. The server is entitled to send this for a
// skill that carries no plugin/marketplace attribution, so the renderer must
// survive it: `<nil>` in a cell is a rendering bug, a panic is worse.
const fixtureSkillDetailNulls = `{
	"id": "JMwX9g4",
	"resourceType": "skill",
	"name": "code-review",
	"originCategory": null,
	"source": null,
	"kind": null,
	"pluginName": null,
	"marketplaceName": null,
	"totalCost": 12.47,
	"callCount": 342,
	"traceCount": 57,
	"firstSeen": null,
	"lastSeen": null
}`

func TestSkillsGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"JMwX9g4"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/skills/JMwX9g4", gotPath)
	out := buf.String()
	assert.Contains(t, out, "code-review") // name
	assert.Contains(t, out, "workflow")    // kind
	assert.Contains(t, out, "342")         // callCount
	assert.Contains(t, out, "57")          // traceCount
	assert.Contains(t, out, "$12.47")      // totalCost
}

func TestSkillsGetWithPeriod(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""
	defer func() { periodFlag = "" }()

	root := newPeriodValidatingRoot(newGetCmd())
	root.SetOut(&buf)
	root.SetArgs([]string{"get", "JMwX9g4", "--period", "SEVEN_DAYS"})
	err := root.Execute()

	require.NoError(t, err)
	assert.Equal(t, "SEVEN_DAYS", gotQuery.Get("period"))
	assert.Empty(t, gotQuery.Get("startDate"), "the platform surface takes period, not a date range")
	assert.Empty(t, gotQuery.Get("endDate"), "the platform surface takes period, not a date range")
}

func TestSkillsGetRejectsPathTraversal(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SilenceErrors = true
	c.SilenceUsage = true
	c.SetArgs([]string{"../../etc/passwd"})
	err := c.Execute()

	// T-22-02: a mitigation that merely garbles the path is not a mitigation.
	// The request must never leave the process.
	require.Error(t, err)
	assert.Equal(t, 0, requestCount, "path traversal must be rejected before any HTTP request")
}

func TestSkillsGetNullableProperties(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillDetailNulls)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"JMwX9g4"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "code-review", "non-null properties still render")
	assert.NotContains(t, out, "<nil>", "a nil property must render as an empty cell")
	assert.NotContains(t, out, "null", "a nil property must render as an empty cell")
}

func TestSkillsGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	periodFlag = ""

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"JMwX9g4"})
	err := c.Execute()

	require.NoError(t, err)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

	var want map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(fixtureSkillDetail), &want))

	assert.Equal(t, want, got, "--json must emit the response object unmodified")
	// resourceType is deliberately absent from the table; --json is the
	// full-fidelity path, so it must survive.
	assert.Equal(t, "skill", got["resourceType"])
	assert.True(t, strings.Contains(buf.String(), "resourceType"))
}
