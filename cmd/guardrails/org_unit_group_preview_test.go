package guardrails

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	apierrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
)

// EVERY TEST IN THIS FILE IS FLAT ON PURPOSE — no t.Run subtests.
//
// The verification gate counts `--- PASS: TestOrgUnitGroupPreview` lines and
// demands an exact figure — nine when 31-04 shipped this file, thirteen after
// 31-07 added four. `go test -v` prints an ADDITIONAL indented `--- PASS:` line
// per subtest, so a single t.Run would make the gate measure one too many and
// fail against a correct implementation. Plans 31-01, 31-02 and 31-03 each hit
// this. Multi-case tests below therefore iterate inline. Do not "tidy" them
// back into subtests.

// previewRequestLog records every request a stub server observed, as
// "METHOD PATH" strings plus the decoded JSON bodies.
//
// Increments are mutex-guarded: httptest serves each request on its own
// goroutine, so an unguarded log would trip the race detector and report a
// data race instead of the write-after-preview regression these tests exist to
// diagnose.
//
// The log records the PATH as well as the count because the load-bearing claim
// of GRDR-08 is not "one request happened" but "no request to any OTHER path
// happened". A command that previewed and then POSTed a budget rule to
// /v2/api/ai/cost-controls would satisfy every rendering assertion in this
// file; only a per-path log catches it.
type previewRequestLog struct {
	mu     sync.Mutex
	calls  []string
	bodies []map[string]interface{}
}

func (l *previewRequestLog) record(r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, r.Method+" "+r.URL.Path)
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		l.bodies = append(l.bodies, body)
	} else {
		l.bodies = append(l.bodies, nil)
	}
}

func (l *previewRequestLog) snapshot() ([]string, []map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	calls := append([]string(nil), l.calls...)
	bodies := append([]map[string]interface{}(nil), l.bodies...)
	return calls, bodies
}

// count returns the number of recorded requests.
func (l *previewRequestLog) count() int {
	calls, _ := l.snapshot()
	return len(calls)
}

// otherThan returns how many recorded requests went to a path other than want.
//
// The comparison is an EQUALITY on the path half, not a suffix test (code
// review WR-06). A suffix test counted any path merely ENDING WITH wantPath as
// the expected one, so a command that previewed and then wrote to a
// differently-prefixed path sharing that tail — the very "previewed, then
// POSTed a budget rule" scenario this file's header calls the load-bearing
// GRDR-08 claim — would have satisfied the assertion it exists to defeat.
//
// calls stays as "METHOD PATH" strings so
// TestOrgUnitGroupPreviewSendsOneRequestAndCreatesNothing's assert.Equal on
// calls[0] keeps asserting the method too; the split happens here instead. A
// record with no space at all cannot be attributed to a path, so it counts as
// an other-path request rather than being silently forgiven.
func (l *previewRequestLog) otherThan(wantPath string) int {
	calls, _ := l.snapshot()
	n := 0
	for _, c := range calls {
		_, path, found := strings.Cut(c, " ")
		if !found || path != wantPath {
			n++
		}
	}
	return n
}

// newPreviewStub starts a hermetic stub that logs every request and replies
// with status and body.
func newPreviewStub(t *testing.T, log *previewRequestLog, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.record(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// runPreview wires cmd.APIClient and cmd.Output at the stub, builds a fresh
// command and runs it with args.
//
// The third positional argument of api.NewClient is TeamID. A preview test that
// passes an empty one fails against requireTeam() rather than against the
// behaviour under test, so every test here passes a real team except the one
// that tests its absence.
func runPreview(baseURL, teamID string, jsonMode bool, buf *bytes.Buffer, args ...string) error {
	cmd.APIClient = api.NewClient(baseURL, "test-key", teamID, "", "", false)
	cmd.Output = output.NewWithWriter(buf, buf, jsonMode, false)

	c := newOrgUnitGroupPreviewCmd()
	c.SetOut(buf)
	c.SetErr(buf)
	// SetArgs is always called, even for the empty case: cobra falls back to
	// os.Args[1:] when args is nil, which under `go test` is the test binary's
	// own flags.
	c.SetArgs(args)
	return c.Execute()
}

// previewFixture is a hand-built OrgUnitBudgetGroupPreviewResult: targetCount
// plus a targets[] array of ResourceMetadata. It is never a decode of the
// cached OpenAPI document and never a frozen schema snapshot — a fixture that
// drifts silently proves the CLI still agrees with itself rather than with the
// contract.
const previewFixture = `{
  "targetCount": 3,
  "targets": [
    {"id": "JMwX9g4", "resourceType": "TEAM", "label": "Platform Engineering"},
    {"id": "K2pQr7z", "resourceType": "TEAM", "label": "Data Science"},
    {"id": "L8mNb3c", "resourceType": "TEAM", "label": "Developer Relations"}
  ]
}`

// previewANSIRE matches the SGR escapes lipgloss emits around table borders
// and headers.
var previewANSIRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// previewTableCells returns the rendered table as trimmed cells — index 0 is
// the header row, index 1 the first DATA row. Ported from tableCells in
// cmd/sessions/attribution_test.go.
//
// The ANSI strip is load-bearing, not cosmetic: output.NewWithWriter
// deliberately skips the colorprofile wrapper that strips escapes in
// production, so a test that wants a CELL rather than a substring has to strip
// them itself. That is the difference between an identity ("the first data row
// IS a/Team A") and a coincidence (a substring that some border glyph or a
// neighbouring column could also satisfy).
func previewTableCells(out string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(previewANSIRE.ReplaceAllString(out, ""), "\n") {
		if !strings.Contains(line, "│") {
			continue
		}
		parts := strings.Split(line, "│")
		if len(parts) < 3 {
			continue
		}
		parts = parts[1 : len(parts)-1] // drop the empty fields outside the borders
		cells := make([]string, len(parts))
		for i, p := range parts {
			cells[i] = strings.TrimSpace(p)
		}
		rows = append(rows, cells)
	}
	return rows
}

// previewTwoTargetsNoCount is the exact payload the phase verifier reproduced
// CR-01 with: two real sub-teams and NO targetCount key at all, a shape
// OrgUnitBudgetGroupPreviewResult permits because neither property is declared
// required.
const previewTwoTargetsNoCount = `{
  "targets": [
    {"id": "a", "resourceType": "TEAM", "label": "Team A"},
    {"id": "b", "resourceType": "TEAM", "label": "Team B"}
  ]
}`

// previewTwoTargetsZeroCount is the same list under an explicit zero count —
// the other half of the CR-01 shape, where the server's figure is present but
// contradicts the list it arrived with.
const previewTwoTargetsZeroCount = `{
  "targetCount": 0,
  "targets": [
    {"id": "a", "resourceType": "TEAM", "label": "Team A"},
    {"id": "b", "resourceType": "TEAM", "label": "Team B"}
  ]
}`

// TestOrgUnitGroupPreviewRendersTargetsWhenTargetCountAbsent is the CR-01
// regression: before 31-07 the empty-state branch ORed str(result,
// "targetCount") == "" into its condition, so this body printed "no direct
// sub-teams" — the strongest possible invented answer — and discarded both
// rows.
func TestOrgUnitGroupPreviewRendersTargetsWhenTargetCountAbsent(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, previewTwoTargetsNoCount)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "40"))

	// Asserted first: render assertions must not be satisfiable by a command
	// that stopped issuing the request altogether.
	require.Equal(t, 1, log.count(), "the preview must still issue exactly one request")

	out := buf.String()
	assert.NotContains(t, out, "no direct sub-teams",
		"the server sent two sub-teams; the CLI may not report the opposite fact")
	assert.Contains(t, out, "cap 2 sub-teams",
		"with no usable count the figure is the length of the list the server actually sent")

	cells := previewTableCells(out)
	require.Len(t, cells, 3, "one header row and two data rows")
	assert.Equal(t, []string{"ID", "Label"}, cells[0])
	assert.Equal(t, []string{"a", "Team A"}, cells[1])
	assert.Equal(t, []string{"b", "Team B"}, cells[2])
}

// TestOrgUnitGroupPreviewRendersTargetsWhenTargetCountZero is the same
// regression under an explicit zero: a count of 0 standing over a non-empty
// list is unusable, not authoritative.
func TestOrgUnitGroupPreviewRendersTargetsWhenTargetCountZero(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, previewTwoTargetsZeroCount)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "40"))

	require.Equal(t, 1, log.count(), "the preview must still issue exactly one request")

	out := buf.String()
	assert.NotContains(t, out, "no direct sub-teams")
	assert.Contains(t, out, "cap 2 sub-teams")

	cells := previewTableCells(out)
	require.Len(t, cells, 3, "one header row and two data rows")
	assert.Equal(t, []string{"ID", "Label"}, cells[0])
	assert.Equal(t, []string{"a", "Team A"}, cells[1])
	assert.Equal(t, []string{"b", "Team B"}, cells[2])
}

// TestOrgUnitGroupPreviewCountClaimedWithoutTargets records the deliberate
// asymmetry of the CR-01 fix, so a later reader does not mistake it for the
// same bug pointing the other way.
//
// The CLI reports the list it can enumerate. A count of 2 with no rows behind
// it is a server-contract disagreement the CLI cannot resolve INTO a table —
// there is nothing to render — so the empty-state sentence stands and the
// count is not promoted into a claim about rows nobody can see. Plan 31-10
// logs the disagreement in deferred-items.md rather than inventing a
// resolution here.
func TestOrgUnitGroupPreviewCountClaimedWithoutTargets(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, `{"targetCount":2,"targets":[]}`)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "40"))

	require.Equal(t, 1, log.count(), "the preview must still issue exactly one request")

	out := buf.String()
	assert.Contains(t, out, "no direct sub-teams")
	assert.NotContains(t, out, "Label", "no target table header may be emitted with no rows to carry")
}

// TestOrgUnitGroupPreviewSendsOneRequestAndCreatesNothing is the assertion
// GRDR-08's success criterion names by hand: exactly one request, to the
// preview path, and none to any other path with any method.
func TestOrgUnitGroupPreviewSendsOneRequestAndCreatesNothing(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, previewFixture)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "40"))

	calls, bodies := log.snapshot()
	require.Len(t, calls, 1, "the preview must issue exactly one request")
	assert.Equal(t, "POST "+orgUnitGroupPreviewPath, calls[0])
	// The load-bearing assertion: a command that previewed and then wrote a
	// budget rule would pass every rendering check in this file.
	assert.Equal(t, 0, log.otherThan(orgUnitGroupPreviewPath),
		"no request of any method may follow the preview")

	require.Len(t, bodies, 1)
	body := bodies[0]
	require.NotNil(t, body)
	assert.Equal(t, "40", body["parentOrgUnitId"])
	assert.Equal(t, "team-1", body["teamId"],
		"teamId must be written into the BODY from the resolved client; query injection does not populate a body")
	assert.Len(t, body, 2, "the body carries exactly parentOrgUnitId and teamId")
}

// TestPreviewRequestLogOtherThanComparesFullPath is the mutation proof for
// WR-06: it exercises the helper directly rather than through the command, so
// the property under test is the comparison itself.
//
// Under the shipped suffix comparison this test measures 0 other-path requests
// and fails — the decoy ENDS WITH the preview path, so the old code counted it
// as the expected call. That failure is exactly the confidence the GRDR-08
// no-second-write assertion was missing.
func TestPreviewRequestLogOtherThanComparesFullPath(t *testing.T) {
	log := &previewRequestLog{}
	log.record(httptest.NewRequest(http.MethodPost, orgUnitGroupPreviewPath, strings.NewReader(`{}`)))
	log.record(httptest.NewRequest(http.MethodPost, "/decoy"+orgUnitGroupPreviewPath, strings.NewReader(`{}`)))

	assert.Equal(t, 2, log.count(), "both requests must be recorded")
	assert.Equal(t, 1, log.otherThan(orgUnitGroupPreviewPath),
		"a path that merely ends with the preview path is a DIFFERENT path")
}

// TestOrgUnitGroupPreviewCarriesNoMutationSurface pins D-31-23: the command
// declares no mutating annotation and no confirmation surface of its own. The
// root's persistent --yes is not this command's concern.
func TestOrgUnitGroupPreviewCarriesNoMutationSurface(t *testing.T) {
	c := newOrgUnitGroupPreviewCmd()

	assert.Empty(t, c.Annotations["mutating"],
		"org-unit-group-preview creates nothing; the mutating marker would gate it behind --dry-run and a prompt")
	assert.Nil(t, c.Flags().Lookup("yes"),
		"the command must declare no confirmation flag of its own")
	assert.Contains(t, c.Short, "creates nothing")
	assert.Contains(t, c.Long, "creates nothing")
}

// TestOrgUnitGroupPreviewRequiresTeam proves the guard is free: the counter is
// asserted BEFORE require.Error, because a test that only asserts an error came
// back still passes when the guard is deleted and the server answers 400.
func TestOrgUnitGroupPreviewRequiresTeam(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, previewFixture)

	var buf bytes.Buffer
	err := runPreview(srv.URL, "", false, &buf, "--parent-org-unit-id", "40")

	assert.Equal(t, 0, log.count(), "an unresolved team must be refused before any request")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--team-id")
	assert.Contains(t, err.Error(), "REVENIUM_TEAM_ID")
	assert.Contains(t, err.Error(), "team-id")
}

// TestOrgUnitGroupPreviewRequiresParentFlag proves the required flag is
// enforced at the cobra layer, so an under-specified body is never sent.
func TestOrgUnitGroupPreviewRequiresParentFlag(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, previewFixture)

	var buf bytes.Buffer
	err := runPreview(srv.URL, "team-1", false, &buf)

	assert.Equal(t, 0, log.count(), "a missing required flag must cost zero HTTP")
	require.Error(t, err)
}

// TestOrgUnitGroupPreviewAcceptsNonNumericId proves no digits-only check was
// invented for a property the schema types as a string. A hashid-shaped value
// is sent unchanged and the server decides.
func TestOrgUnitGroupPreviewAcceptsNonNumericId(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, previewFixture)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "a91XJp"))

	calls, bodies := log.snapshot()
	require.Len(t, calls, 1)
	require.Len(t, bodies, 1)
	assert.Equal(t, "a91XJp", bodies[0]["parentOrgUnitId"],
		"the value must reach the server unchanged; a client-side numeric pattern would be our invention")
}

// TestOrgUnitGroupPreviewZeroTargets pins the boundary case: zero is a real
// answer about the org structure, not an empty result set, so it reads as a
// sentence and emits no table header.
func TestOrgUnitGroupPreviewZeroTargets(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, `{"targetCount":0,"targets":[]}`)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "40"))

	out := buf.String()
	assert.Contains(t, out, "no direct sub-teams")
	assert.NotContains(t, out, "Label", "no target table header may be emitted for zero targets")
}

// TestOrgUnitGroupPreview422NamesFeatureFlags pins D-31-27: the enrichment
// names both flags AND the %w wrap survives, so the shared exit-code
// resolution (internal/errors.ExitCodeFor, which unwraps with errors.As) still
// resolves the validation exit code.
func TestOrgUnitGroupPreview422NamesFeatureFlags(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusUnprocessableEntity,
		`{"message":"Department budgets not enabled for this team"}`)

	var buf bytes.Buffer
	err := runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "40")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "org-unit-budgets-enabled")
	assert.Contains(t, err.Error(), "org-unit-attribution-enabled")

	var apiErr *apierrors.APIError
	require.True(t, errors.As(err, &apiErr), "the %w wrap must keep the APIError recoverable")
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
	assert.Equal(t, apierrors.ExitValidation, apierrors.ExitCodeFor(err))
}

// TestOrgUnitGroupPreviewJSONEmitsOneDocument pins the single-branch rule: a
// two-section render that called the shared entry point once per section would
// emit two concatenated JSON documents, which json.Unmarshal rejects.
func TestOrgUnitGroupPreviewJSONEmitsOneDocument(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK, previewFixture)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", true, &buf, "--parent-org-unit-id", "40"))

	var parsed interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed),
		"JSON mode must emit exactly one document")
	doc, ok := parsed.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), doc["targetCount"])
	assert.Len(t, doc["targets"], 3)
}

// TestStrRendersIntegralJSONNumbersAsIntegers pins WR-02 at the helper.
//
// The fixture is DECODED from a JSON literal rather than written as Go values
// on purpose: that is the only way the numbers under test are real
// encoding/json float64s. Go constants would let an int slip in and the test
// would prove the wrong thing — %v on an int never reached exponent form, so
// the defect would be invisible.
//
// The loop is inline, not t.Run subtests, per this file's header rule.
func TestStrRendersIntegralJSONNumbersAsIntegers(t *testing.T) {
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(`{
	  "targetCount": 1234567,
	  "hardLimit": 1000,
	  "ratio": 0.5,
	  "metricType": "TOTAL_COST",
	  "nulled": null
	}`), &decoded))

	cases := []struct {
		key  string
		want string
		why  string
	}{
		{"targetCount", "1234567", "a seven-digit whole number must reach the operator as digits, not 1.234567e+06"},
		{"hardLimit", "1000", "the shipped budget-rules fixture must render exactly as it does today"},
		{"ratio", "0.5", "a fractional value keeps its decimal point and its existing formatting"},
		{"metricType", "TOTAL_COST", "a string passes through untouched"},
		{"nulled", "", "a JSON null is the missing-key contract: the empty string"},
		{"absent", "", "a key that is not present is the empty string"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, str(decoded, tc.key), tc.why)
	}
}

// TestOrgUnitGroupPreviewRendersLargeTargetCountInFull is the end-to-end half
// of WR-02: the figure the operator actually reads.
//
// The fixture's count (1234567) and list length (1) are deliberately unequal.
// This test isolates the FORMAT of the server's own figure, which Task 1
// established is printed verbatim whenever it is usable — the count/list
// disagreement is the subject of
// TestOrgUnitGroupPreviewCountClaimedWithoutTargets, not of this one.
func TestOrgUnitGroupPreviewRendersLargeTargetCountInFull(t *testing.T) {
	log := &previewRequestLog{}
	srv := newPreviewStub(t, log, http.StatusOK,
		`{"targetCount":1234567,"targets":[{"id":"a","resourceType":"TEAM","label":"Team A"}]}`)

	var buf bytes.Buffer
	require.NoError(t, runPreview(srv.URL, "team-1", false, &buf, "--parent-org-unit-id", "40"))

	require.Equal(t, 1, log.count(), "the preview must still issue exactly one request")
	assert.Contains(t, buf.String(), "cap 1234567 sub-teams",
		"the server's own figure must reach the operator in full decimal, not in exponent form")
}

// TestOrgUnitGroupPreviewRegisteredFlat proves the command is a DIRECT child of
// guardrails and not nested under budget-rules (D-31-22). CommandPath() is the
// assertion rather than Name(), because a name matches wherever it is mounted.
func TestOrgUnitGroupPreviewRegisteredFlat(t *testing.T) {
	found, leftover, err := Cmd.Find([]string{"org-unit-group-preview"})
	require.NoError(t, err)
	assert.Empty(t, leftover)
	require.NotNil(t, found)
	assert.Equal(t, "guardrails org-unit-group-preview", found.CommandPath())
}
