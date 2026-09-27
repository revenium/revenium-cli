package skills

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// fixtureSkillsPage is a real-schema SkillUsagePagedModel_Read envelope: an
// `_embedded.skillUsageResourceList` array of one
// EntityModelSkillUsageResource_Read carrying all 13 properties from the
// cached platform spec plus its `_links` member, alongside the sibling
// `page` object.
//
// The envelope shape is load-bearing, not decoration. It is the proof that
// the generic `_embedded` unwrap in internal/api/client.go — which iterates
// the `_embedded` map rather than keying on a known resource-list name —
// handles `skillUsageResourceList` with no change to the client.
const fixtureSkillsPage = `{
	"_embedded": {
		"skillUsageResourceList": [{
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
			"lastSeen": "2026-06-29T18:30:00Z",
			"_links": {
				"self": {"href": "https://api.revenium.io/v2/api/skills/JMwX9g4"}
			}
		}]
	},
	"_links": {
		"self": {"href": "https://api.revenium.io/v2/api/skills"}
	},
	"page": {
		"size": 20,
		"totalElements": 1,
		"totalPages": 1,
		"number": 0
	}
}`

func TestSkillsList(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/skills", gotPath)
	out := buf.String()
	assert.Contains(t, out, "code-review") // name
	assert.Contains(t, out, "342")         // callCount
	assert.Contains(t, out, "57")          // traceCount
	assert.Contains(t, out, "$12.47")      // totalCost
}

// fixtureSkillsPageEmpty is the envelope the server returns for a team with
// no recorded skill usage: a present-but-empty resource list, not a missing
// `_embedded` member.
const fixtureSkillsPageEmpty = `{
	"_embedded": {
		"skillUsageResourceList": []
	},
	"_links": {
		"self": {"href": "https://api.revenium.io/v2/api/skills"}
	},
	"page": {
		"size": 20,
		"totalElements": 0,
		"totalPages": 0,
		"number": 0
	}
}`

// skillsPageN renders a one-item envelope whose skill name identifies which
// page served it, declaring totalPages so aggregation is driven by the
// server's own metadata rather than by the requested batch size.
func skillsPageN(name string, totalPages int) string {
	return fmt.Sprintf(`{
	"_embedded": {
		"skillUsageResourceList": [{
			"id": "skill-%[1]s",
			"resourceType": "skill",
			"name": %[1]q,
			"originCategory": "VENDOR",
			"source": "bundled",
			"kind": "workflow",
			"totalCost": 1.5,
			"callCount": 10,
			"traceCount": 2
		}]
	},
	"page": {"size": 100, "totalElements": %[2]d, "totalPages": %[2]d, "number": 0}
}`, name, totalPages)
}

func TestSkillsListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	var got []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 1)

	// The item objects are emitted raw: everything the eight-column table
	// drops must still be here, because --json is the full-fidelity path.
	assert.Equal(t, "JMwX9g4", got[0]["id"])
	assert.Equal(t, "skill", got[0]["resourceType"])
	assert.Equal(t, "revenium-tools", got[0]["pluginName"])
	assert.Equal(t, "anthropics", got[0]["marketplaceName"])
	assert.Equal(t, "2026-06-01T12:00:00Z", got[0]["firstSeen"])
	assert.Equal(t, "2026-06-29T18:30:00Z", got[0]["lastSeen"])
	assert.Contains(t, got[0], "_links")
}

func TestSkillsListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPageEmpty)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())
	assert.Contains(t, buf.String(), "No skills found.")
}

func TestSkillsListEmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPageEmpty)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	var got []interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Empty(t, got, "an empty result set is an empty array, not null and not a table")
}

func TestSkillsListTeamId(t *testing.T) {
	var gotTeamID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTeamID = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	// D-22-03: teamId is auto-injected from resolved config. No --team-id
	// flag exists on this command and none is needed.
	assert.Equal(t, "team-456", gotTeamID)
	assert.Nil(t, c.Flags().Lookup("team-id"), "no package-local --team-id flag (D-22-03)")
}

func TestSkillsListWithPeriod(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""
	defer func() { periodFlag = "" }()

	root := newPeriodValidatingRoot(newListCmd())
	root.SetOut(&buf)
	root.SetArgs([]string{"list", "--period", "SEVEN_DAYS"})
	require.NoError(t, root.Execute())

	assert.Equal(t, "SEVEN_DAYS", gotQuery.Get("period"))
	assert.Empty(t, gotQuery.Get("startDate"), "the platform surface takes period, not a date range")
	assert.Empty(t, gotQuery.Get("endDate"), "the platform surface takes period, not a date range")
}

func TestSkillsListInvalidPeriod(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""
	defer func() { periodFlag = "" }()

	root := newPeriodValidatingRoot(newListCmd())
	root.SetOut(&buf)
	root.SetArgs([]string{"list", "--period", "BOGUS"})
	err := root.Execute()

	// T-22-01: an unknown period is rejected before it can be interpolated
	// into a request URL.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	assert.Equal(t, 0, requestCount, "invalid --period must fail before any HTTP request")
}

func TestSkillsListPagination(t *testing.T) {
	var gotRawQuery string
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--page", "2", "--page-size", "5"})
	require.NoError(t, c.Execute())

	// Asserted on the raw query so a stray extra parameter is caught.
	assert.Equal(t, "page=2&size=5", gotRawQuery)
	// An explicit --page means "exactly this page" and opts out of
	// aggregation — one request, no more.
	assert.Equal(t, 1, requestCount)
}

func TestSkillsListAggregatesAllPages(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		name := "skill-page-one"
		if r.URL.Query().Get("page") == "1" {
			name = "skill-page-two"
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, skillsPageN(name, 2))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	out := buf.String()
	assert.Contains(t, out, "skill-page-one")
	assert.Contains(t, out, "skill-page-two")
	assert.Equal(t, 2, requestCount, "without --page, iteration is driven by the response's own totalPages")
}

// TestSkillsListAggregatesAllPagesJSON records that aggregation is NOT
// table-only. SC1 asks for auto-pagination in table mode; ListOptsFromFlags
// deliberately applies it in every output mode, because silently handing an
// automation caller page one while it believes it has the whole result set is
// the most damaging available default (see the doc comment in cmd/list.go).
// That is a superset of SC1, so this asserts it rather than narrowing it.
func TestSkillsListAggregatesAllPagesJSON(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		name := "skill-page-one"
		if r.URL.Query().Get("page") == "1" {
			name = "skill-page-two"
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, skillsPageN(name, 2))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)
	periodFlag = ""

	c := newListCmd()
	c.SetOut(&buf)
	require.NoError(t, c.Execute())

	var got []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 2)
	assert.Equal(t, "skill-page-one", got[0]["name"])
	assert.Equal(t, "skill-page-two", got[1]["name"])
	assert.Equal(t, 2, requestCount)
}
