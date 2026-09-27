package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// Every fixture in this file is written from the VERBATIM property sets in
// 30-RESEARCH.md § The API Contract → 2 — NOT from the rendering code. Writing
// a fixture from the code is Pitfall 2: the table then renders whatever the
// code already reads, the test is green, and the field names may not appear in
// any schema at all. That is the defect vcs_prs.go:45-46 shipped.
//
// The spec declares example values only for the three QUERY parameters; the
// response schema carries none. The response values below are therefore
// plausible rather than quoted, and they were chosen to be mutually
// distinguishable so an assertion on a totals cell cannot be satisfied by a
// digit that belongs to some other cell or to a date.

// prHealthFixture is a fully-populated response.
//
// The engineers are supplied OUT of alphabetical order on purpose: the spec
// states no ordering for that array, so the two output modes must diverge —
// the table sorts ascending by authorLogin, --json stays the server's document
// in the server's order.
//
// Three nullability states are represented across the engineer rows, because
// they must stay distinguishable in the render: zoe-park carries an explicitly
// null mappedEmail and no oldestInactiveDays key at all, while marco-diaz
// carries a measured oldestInactiveDays of 0.
const prHealthFixture = `{
	"source": "github",
	"startDate": "2026-05-17",
	"endDate": "2026-08-17",
	"agingDays": 14,
	"rottingDays": 45,
	"totals": {
		"openPrs": 128,
		"draftPrs": 9,
		"agingPrs": 31,
		"rottingPrs": 12,
		"rottingPrsAssisted": 5,
		"closedUnmerged": 23,
		"closedUnmergedAssisted": 8,
		"avgCostPerMergedPr": 137.5,
		"lastSyncedAt": "2026-08-17T04:15:00Z"
	},
	"engineers": [
		{
			"authorLogin": "zoe-park",
			"mappedEmail": null,
			"openPrs": 6,
			"agingPrs": 2,
			"rottingPrs": 1,
			"closedUnmerged": 3
		},
		{
			"authorLogin": "marco-diaz",
			"mappedEmail": "marco@example.com",
			"openPrs": 11,
			"agingPrs": 4,
			"rottingPrs": 0,
			"closedUnmerged": 2,
			"oldestInactiveDays": 0
		},
		{
			"authorLogin": "alice-chen",
			"mappedEmail": "alice@example.com",
			"openPrs": 25,
			"agingPrs": 7,
			"rottingPrs": 3,
			"closedUnmerged": 4,
			"oldestInactiveDays": 34
		}
	],
	"oldest": [
		{
			"repoName": "revenium/platform",
			"prNumber": 4821,
			"title": "Add seat sync",
			"url": "https://example.invalid/revenium/platform/pull/4821",
			"authorLogin": "zoe-park",
			"mappedEmail": null,
			"codingToolAssisted": true,
			"reviewDecision": "CHANGES_REQUESTED",
			"ageDays": 63,
			"inactiveDays": 47,
			"createdAtVcs": "2026-06-15T10:02:00Z",
			"updatedAtVcs": "2026-07-01T09:30:00Z",
			"lastCommitAtVcs": null,
			"lastSyncedAt": "2026-08-17T04:15:00Z",
			"draft": false
		}
	]
}`

// prHealthServer returns an httptest server serving body, plus a pointer to
// the count of requests it received. The counter is what proves the enum
// refusal happens BEFORE the request rather than after it.
func prHealthServer(t *testing.T, body string) (*httptest.Server, *int) {
	t.Helper()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

// TestPrHealthRequestPath pins the request half: the exact path, this
// endpoint's own source/startDate/endDate wire keys, the absence of the
// fromDate/toDate spelling `billing seats` uses, and that exactly one request
// leaves the process.
//
// The negative half is the load-bearing one. `/v2/api/billing/seats` is the
// one endpoint in this repository that spells its range fromDate/toDate; a
// reader moving between the two files could "correct" this one to match, and
// the resulting 400 reads like a date-format problem rather than a wrong
// parameter name.
func TestPrHealthRequestPath(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "/v2/api/billing/users/vcs-pr-health", r.URL.Path)
		assert.Equal(t, "github", r.URL.Query().Get("source"))
		assert.Equal(t, "2026-05-17", r.URL.Query().Get("startDate"))
		assert.Equal(t, "2026-08-17", r.URL.Query().Get("endDate"))
		assert.Empty(t, r.URL.Query().Get("fromDate"),
			"this endpoint spells the range startDate/endDate — fromDate belongs to billing seats alone")
		assert.Empty(t, r.URL.Query().Get("toDate"),
			"this endpoint spells the range startDate/endDate — toDate belongs to billing seats alone")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, prHealthFixture)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVcsPrHealthCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--source", "github", "--from", "2026-05-17", "--to", "2026-08-17"})
	require.NoError(t, c.Execute())

	assert.Equal(t, 1, calls, "the command must issue exactly one GET")

	// All three parameters are required: true in the spec, so a request
	// missing one would be refused by the server anyway — but sending it
	// spends an authenticated round trip to learn what cobra already knew.
	//
	// --to is the arm exercised here because it lands on cobra's own
	// required-flag error. Omitting --source instead lands on the enum
	// refusal, because cobra runs PreRunE before ValidateRequiredFlags; that
	// is equally a refusal at zero HTTP calls, and it is why
	// validateVcsSource carries no !f.Changed early return.
	missingSrv, missingCalls := prHealthServer(t, prHealthFixture)

	var missingBuf bytes.Buffer
	cmd.APIClient = api.NewClient(missingSrv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&missingBuf, &missingBuf, false, false)

	missingCmd := newVcsPrHealthCmd()
	missingCmd.SetOut(&missingBuf)
	missingCmd.SetErr(&missingBuf)
	missingCmd.SetArgs([]string{"--source", "github", "--from", "2026-05-17"})
	err := missingCmd.Execute()

	require.Error(t, err, "a missing required date must be refused")
	assert.Contains(t, err.Error(), "to", "the error must name the missing flag")
	assert.Equal(t, 0, *missingCalls, "the refusal must precede the request")
}

// TestPrHealthRejectsSourceOutsideEnum is the T-30-02 assertion.
//
// The zero-call assertion is the one that matters. An error message alone
// would stay green on a validator that ran after the request had already been
// sent; only the counter proves the refusal preceded it.
func TestPrHealthRejectsSourceOutsideEnum(t *testing.T) {
	srv, calls := prHealthServer(t, prHealthFixture)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVcsPrHealthCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--source", "svn", "--from", "2026-05-17", "--to", "2026-08-17"})
	err := c.Execute()

	require.Error(t, err, "a source outside the spec's two prose values must be refused")
	assert.Contains(t, err.Error(), "github", "the refusal must name the accepted values")
	assert.Contains(t, err.Error(), "gitlab", "the refusal must name the accepted values")
	assert.Equal(t, 0, *calls, "the refusal must precede the request")
}

// TestPrHealthAcceptsEveryLegalSource is the other direction of the same gate:
// a validator that refused everything would pass the refusal test above, so
// every value the spec's prose declares legal is exercised here.
//
// It iterates validVcsSources itself rather than a second hand-written list,
// so widening the enum cannot leave the positive test behind.
//
// Deliberately NOT written as t.Run subtests: 30-02-PLAN.md's verify gates
// count `--- PASS: TestPrHealth` lines and require exact totals, and go test
// emits an additional indented PASS line per subtest.
func TestPrHealthAcceptsEveryLegalSource(t *testing.T) {
	require.NotEmpty(t, validVcsSources, "the enum must not be empty")

	for _, source := range validVcsSources {
		srv, calls := prHealthServer(t, prHealthFixture)

		var buf bytes.Buffer
		cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
		cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

		c := newVcsPrHealthCmd()
		c.SetOut(&buf)
		c.SetErr(&buf)
		c.SetArgs([]string{"--source", source, "--from", "2026-05-17", "--to", "2026-08-17"})

		require.NoError(t, c.Execute(), "source %q is declared legal by the spec and must be accepted", source)
		assert.Equal(t, 1, *calls, "source %q must reach the server", source)
	}
}

// prHealthFixtureEmptyEngineers is the response for a window in which the
// organization has a populated roll-up but no per-engineer rows. The spec
// declares engineers as a plain array with no minimum, so an empty one is a
// legitimate success rather than an error.
const prHealthFixtureEmptyEngineers = `{
	"source": "gitlab",
	"startDate": "2026-05-17",
	"endDate": "2026-08-17",
	"agingDays": 14,
	"rottingDays": 45,
	"totals": {
		"openPrs": 128,
		"draftPrs": 9,
		"agingPrs": 31,
		"rottingPrs": 12,
		"rottingPrsAssisted": 5,
		"closedUnmerged": 23,
		"closedUnmergedAssisted": 8,
		"avgCostPerMergedPr": null,
		"lastSyncedAt": null
	},
	"engineers": [],
	"oldest": []
}`

// TestPrHealthRendersEngineers asserts BILL-05's own success criterion: the
// per-engineer aging, rotting and closed-unmerged counts, alongside the
// organization roll-up and the server's echoed thresholds.
//
// It pins four separate contracts in one body, because they are only true
// together:
//
//   - ordering is ascending by authorLogin, from a fixture that supplies the
//     engineers out of order;
//   - a value the server withheld and a value it measured as zero are
//     DIFFERENT strings, so a render that turned every count into a marker
//     fails here;
//   - the totals row carries the server's counts AND the top-level echoed
//     thresholds, which live outside the totals object;
//   - no currency symbol appears anywhere. The operation description states
//     dollar estimates are computed client-side as count x avgCostPerMergedPr;
//     this CLI renders no such figure, so no cell in the table is a number the
//     server did not send.
//
// Deliberately NOT written as t.Run subtests, for the reason
// TestPrHealthAcceptsEveryLegalSource records.
func TestPrHealthRendersEngineers(t *testing.T) {
	srv, calls := prHealthServer(t, prHealthFixture)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVcsPrHealthCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--source", "github", "--from", "2026-05-17", "--to", "2026-08-17"})
	require.NoError(t, c.Execute())
	require.Equal(t, 1, *calls)

	out := buf.String()

	assert.Contains(t, out, "Totals", "the roll-up section must be labelled")
	assert.Contains(t, out, "By Engineer", "the per-engineer section must be labelled")

	// The fixture supplies zoe-park, marco-diaz, alice-chen in that order.
	alice := strings.Index(out, "alice-chen")
	marco := strings.Index(out, "marco-diaz")
	zoe := strings.Index(out, "zoe-park")
	require.NotEqual(t, -1, alice)
	require.NotEqual(t, -1, marco)
	require.NotEqual(t, -1, zoe)
	assert.Less(t, alice, marco, "engineer rows must be ordered ascending by authorLogin")
	assert.Less(t, marco, zoe, "engineer rows must be ordered ascending by authorLogin")

	// zoe-park carries an explicitly null mappedEmail and no oldestInactiveDays
	// key at all; both must read as undeclared rather than as an empty string
	// or a confident zero.
	assert.GreaterOrEqual(t, strings.Count(out, undeclared), 2,
		"a null mappedEmail and an absent oldestInactiveDays must both render as the undeclared marker")
	// marco-diaz's oldestInactiveDays was measured as zero and must survive as 0.
	assert.Regexp(t, regexp.MustCompile(`\b0\b`), out,
		"a measured zero must still render as 0, not as the undeclared marker")

	// The roll-up counts BILL-05 names, plus the two echoed thresholds, which
	// come from the TOP LEVEL of the response rather than from totals.
	for _, want := range []string{`\b31\b`, `\b12\b`, `\b23\b`, `\b14\b`, `\b45\b`} {
		assert.Regexp(t, regexp.MustCompile(want), out,
			"the Totals row must carry the server's value matching %s", want)
	}
	assert.Contains(t, out, "137.50", "the nullable average must render at the server's value")

	assert.NotContains(t, out, "$",
		"no currency symbol may be asserted for a schema that declares no currency, and no dollar figure is computed here")

	// Stability: two engineers comparing equal on authorLogin must keep the
	// server's relative order, which is what sort.SliceStable buys over sort.Slice.
	// This fixture is local rather than a package-level const because it exists
	// only to pin that one property.
	const tiedFixture = `{
		"totals": {"openPrs": 2},
		"engineers": [
			{"authorLogin": "same-login", "openPrs": 777, "agingPrs": 0, "rottingPrs": 0, "closedUnmerged": 0, "oldestInactiveDays": 0},
			{"authorLogin": "same-login", "openPrs": 888, "agingPrs": 0, "rottingPrs": 0, "closedUnmerged": 0, "oldestInactiveDays": 0}
		]
	}`
	tiedSrv, tiedCalls := prHealthServer(t, tiedFixture)

	var tiedBuf bytes.Buffer
	cmd.APIClient = api.NewClient(tiedSrv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&tiedBuf, &tiedBuf, false, false)

	tiedCmd := newVcsPrHealthCmd()
	tiedCmd.SetOut(&tiedBuf)
	tiedCmd.SetArgs([]string{"--source", "github", "--from", "2026-05-17", "--to", "2026-08-17"})
	require.NoError(t, tiedCmd.Execute())
	require.Equal(t, 1, *tiedCalls)

	tiedOut := tiedBuf.String()
	first := strings.Index(tiedOut, "777")
	second := strings.Index(tiedOut, "888")
	require.NotEqual(t, -1, first)
	require.NotEqual(t, -1, second)
	assert.Less(t, first, second,
		"engineers comparing equal on authorLogin must keep the order the server sent")
}

// TestPrHealthEmptyEngineers covers the window in which the roll-up is
// populated but no engineer rows come back. It is a success in BOTH output
// modes — a sentence in table mode, the server's own document under --json.
//
// The Totals half of the assertion is the load-bearing one: an implementation
// that returned early on an empty engineers array would suppress a section the
// server did populate.
//
// Deliberately NOT written as t.Run subtests, for the reason
// TestPrHealthAcceptsEveryLegalSource records.
func TestPrHealthEmptyEngineers(t *testing.T) {
	srv, calls := prHealthServer(t, prHealthFixtureEmptyEngineers)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newVcsPrHealthCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--source", "gitlab", "--from", "2026-05-17", "--to", "2026-08-17"})
	require.NoError(t, c.Execute(), "an empty engineer list is a success, not an error")
	require.Equal(t, 1, *calls)

	out := buf.String()
	assert.Contains(t, out, "No engineers are reported for this window.")
	for _, want := range []string{`\b31\b`, `\b12\b`, `\b23\b`, `\b14\b`, `\b45\b`} {
		assert.Regexp(t, regexp.MustCompile(want), out,
			"the Totals section must still render when engineers is empty (%s)", want)
	}
	// avgCostPerMergedPr and lastSyncedAt are both null in this fixture.
	assert.GreaterOrEqual(t, strings.Count(out, undeclared), 2,
		"a null average and a null lastSyncedAt must render as the undeclared marker")
}

// TestPrHealthJSONIsVerbatim proves the table-layout decisions are TABLE
// decisions and not data loss: oldest[] has no section of its own, but it
// survives untouched under --json, and so does every totals key.
//
// It also pins the ordering contract from the other side. The fixture supplies
// the engineers out of alphabetical order; --json must emit them in the
// server's order while the table (asserted in TestPrHealthRendersEngineers)
// emits them ascending. A sort placed above the output-mode branch would pass
// the table half and silently break this one.
func TestPrHealthJSONIsVerbatim(t *testing.T) {
	srv, calls := prHealthServer(t, prHealthFixture)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-1", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newVcsPrHealthCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--source", "github", "--from", "2026-05-17", "--to", "2026-08-17"})
	require.NoError(t, c.Execute())
	require.Equal(t, 1, *calls)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	for _, key := range []string{"source", "startDate", "endDate", "agingDays", "rottingDays", "totals", "engineers", "oldest"} {
		_, ok := parsed[key]
		assert.True(t, ok, "--json must carry the server's %q key", key)
	}

	totals, ok := parsed["totals"].(map[string]interface{})
	require.True(t, ok)
	for _, key := range []string{
		"openPrs", "draftPrs", "agingPrs", "rottingPrs", "rottingPrsAssisted",
		"closedUnmerged", "closedUnmergedAssisted", "avgCostPerMergedPr", "lastSyncedAt",
	} {
		_, ok := totals[key]
		assert.True(t, ok, "--json must carry the server's totals.%s", key)
	}

	// oldest[] is rendered as no table at all, so this is the assertion that
	// the decision costs the operator no data.
	oldest, ok := parsed["oldest"].([]interface{})
	require.True(t, ok)
	require.Len(t, oldest, 1)
	oldestPr, ok := oldest[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "CHANGES_REQUESTED", oldestPr["reviewDecision"],
		"a field no table shows must still survive verbatim under --json")
	assert.Equal(t, float64(4821), oldestPr["prNumber"])

	engineers, ok := parsed["engineers"].([]interface{})
	require.True(t, ok)
	require.Len(t, engineers, 3)
	firstEngineer, ok := engineers[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "zoe-park", firstEngineer["authorLogin"],
		"--json must emit the engineers in the SERVER's order — the fixture supplies zoe-park first")
}
