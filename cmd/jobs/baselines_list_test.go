package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// ansiRE matches the SGR escape sequences lipgloss emits for cell styling.
// output.NewWithWriter deliberately skips the colorprofile wrapper that
// strips them in production, so a test that parses cells must strip them
// itself or every cell comes back wrapped in escape bytes.
var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// tableRows parses a rendered table into its cell rows, header row first.
//
// Substring assertions cannot express "the FIRST data row carries version 5" —
// they only prove a value appears somewhere. Row order is the whole substance
// of JOBS-13, so it has to be asserted positionally.
func tableRows(t *testing.T, out string) [][]string {
	t.Helper()
	var rows [][]string
	for _, line := range strings.Split(ansiRE.ReplaceAllString(out, ""), "\n") {
		if !strings.Contains(line, "│") {
			continue
		}
		parts := strings.Split(line, "│")
		// The leading and trailing borders produce empty edge fields.
		cells := make([]string, 0, len(parts))
		for _, p := range parts[1 : len(parts)-1] {
			cells = append(cells, strings.TrimSpace(p))
		}
		rows = append(rows, cells)
	}
	require.NotEmpty(t, rows, "no bordered table rows found in output:\n%s", out)
	return rows
}

// baselinesStub serves body at the baselines endpoint, asserting the method,
// the escaped path and that teamId reached the wire.
func baselinesStub(t *testing.T, body string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v2/api/jobs/types/acme-review/baselines", r.URL.Path)
		assert.Equal(t, "team-27", r.URL.Query().Get("teamId"),
			"teamId is required by the endpoint and must reach the wire")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}
}

// outOfOrderBaselines is the load-bearing fixture for the sort assertions.
//
// The version literals appear here in the order 2, 5, 1, 3 — deliberately NOT
// already descending. A fixture that arrived sorted would keep
// TestBaselinesListSortsDescending green with the sort deleted, which would
// make the test a statement about the stub rather than about our output. The
// disorder IS the assertion's substance.
const outOfOrderBaselines = `[
  {"version":2,"costPerUnit":4.25,"minutesPerUnit":18,"qualityRate":0.955,"hourlyRate":85.5,"currency":"USD","provenance":"CUSTOMER_DECLARED","declaredBy":"v2@acme.example","evidenceUrl":"https://acme.example/e/2","effectiveFrom":"2026-01-02T00:00:00Z"},
  {"version":5,"costPerUnit":3.10,"minutesPerUnit":12,"qualityRate":0.980,"hourlyRate":90.0,"currency":"USD","provenance":"MEASURED","declaredBy":"v5@acme.example","evidenceUrl":"https://acme.example/e/5","effectiveFrom":"2026-01-05T00:00:00Z"},
  {"version":1,"costPerUnit":5.00,"minutesPerUnit":22,"qualityRate":0.900,"hourlyRate":80.0,"currency":"USD","provenance":"CUSTOMER_DECLARED","declaredBy":"v1@acme.example","evidenceUrl":"https://acme.example/e/1","effectiveFrom":"2026-01-01T00:00:00Z"},
  {"version":3,"costPerUnit":4.00,"minutesPerUnit":16,"qualityRate":0.960,"hourlyRate":86.0,"currency":"USD","provenance":"SIGNED_OFF","declaredBy":"v3@acme.example","evidenceUrl":"https://acme.example/e/3","effectiveFrom":"2026-01-03T00:00:00Z"}
]`

// TestBaselinesListSortsDescending is JOBS-13's criterion stated as a property
// of our output: row 1 is the version in force.
//
// Mutation pin: deleting the sort.SliceStable call in baselines_list.go must
// turn this test red on the first-row version.
func TestBaselinesListSortsDescending(t *testing.T) {
	srv := newStubServer(t, baselinesStub(t, outOfOrderBaselines))

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf, "types", "baselines", "list", "acme-review"))

	rows := tableRows(t, buf.String())
	require.Len(t, rows, 5, "one header row plus four data rows")
	assert.Equal(t, "Version", rows[0][0], "row 0 is the header")

	assert.Equal(t, "5", rows[1][0],
		"the first data row must carry the newest version — the version in force (JOBS-13)")
	assert.Equal(t, "1", rows[4][0],
		"the last data row must carry the oldest version")
	assert.Equal(t, []string{"5", "3", "2", "1"},
		[]string{rows[1][0], rows[2][0], rows[3][0], rows[4][0]},
		"the whole table must be descending, not merely its first row")
}

// TestBaselinesListJSONOrder proves the table and --json come from the same
// sorted slice: they cannot disagree about which version is first.
//
// It unmarshals the captured buffer rather than substring-matching, because a
// substring assertion cannot express array order at all.
func TestBaselinesListJSONOrder(t *testing.T) {
	srv := newStubServer(t, baselinesStub(t, outOfOrderBaselines))

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	require.NoError(t, execJobs(&buf, "types", "baselines", "list", "acme-review"))

	var decoded []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	require.Len(t, decoded, 4)

	got := make([]float64, len(decoded))
	for i, b := range decoded {
		got[i] = output.FloatVal(b, "version")
	}
	assert.Equal(t, []float64{5, 3, 2, 1}, got,
		"--json must be emitted from the SAME sorted slice the table renders")
}

// equalVersionBaselines carries two entries both at version 4, distinguishable
// only by declaredBy. json.Unmarshal decodes both versions to the identical
// float64, so a non-stable sort is free to swap them and a merging or
// deduplicating implementation is free to drop one.
const equalVersionBaselines = `[
  {"version":4,"costPerUnit":4.25,"currency":"USD","provenance":"MEASURED","declaredBy":"first@acme.example","effectiveFrom":"2026-01-04T00:00:00Z"},
  {"version":4,"costPerUnit":4.75,"currency":"USD","provenance":"MEASURED","declaredBy":"second@acme.example","effectiveFrom":"2026-01-04T12:00:00Z"}
]`

// TestBaselinesListStableOnEqualVersions pins the "Stable" in SliceStable.
// Equal versions keep the server's relative order and are never merged or
// dropped.
func TestBaselinesListStableOnEqualVersions(t *testing.T) {
	srv := newStubServer(t, baselinesStub(t, equalVersionBaselines))

	var buf bytes.Buffer
	cmd.APIClient = stubClient(srv.URL, "team-27")
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	require.NoError(t, execJobs(&buf, "types", "baselines", "list", "acme-review"))

	rows := tableRows(t, buf.String())
	require.Len(t, rows, 3, "one header row plus BOTH equal-version rows — neither may be dropped")

	declaredBy := rows[0]
	col := -1
	for i, h := range declaredBy {
		if h == "Declared By" {
			col = i
		}
	}
	require.NotEqual(t, -1, col, "the table must carry a Declared By column")

	assert.Equal(t, "4", rows[1][0])
	assert.Equal(t, "4", rows[2][0])
	assert.Equal(t, "first@acme.example", rows[1][col],
		"equal versions must retain the server's relative order")
	assert.Equal(t, "second@acme.example", rows[2][col])
}

// TestBaselinesListEmpty pins the deliberate empty state in both modes.
func TestBaselinesListEmpty(t *testing.T) {
	t.Run("table", func(t *testing.T) {
		srv := newStubServer(t, baselinesStub(t, `[]`))

		var buf bytes.Buffer
		cmd.APIClient = stubClient(srv.URL, "team-27")
		cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

		require.NoError(t, execJobs(&buf, "types", "baselines", "list", "acme-review"))
		assert.Contains(t, buf.String(), "No baseline versions declared for this job type.",
			"an empty result is stated as a fact, never rendered as a blank screen")
	})

	t.Run("json", func(t *testing.T) {
		srv := newStubServer(t, baselinesStub(t, `[]`))

		var buf bytes.Buffer
		cmd.APIClient = stubClient(srv.URL, "team-27")
		cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

		require.NoError(t, execJobs(&buf, "types", "baselines", "list", "acme-review"))

		// The literal matters: a nil Go slice marshals to `null`, which is a
		// different value to a consumer than `[]`. The typed-empty-slice
		// branch in the command is what prevents it.
		assert.Equal(t, "[]", strings.TrimSpace(buf.String()),
			"an empty result must be a typed empty array, never null")

		var decoded []map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
		assert.NotNil(t, decoded, "unmarshalling `null` would leave this nil")
		assert.Empty(t, decoded)
	})
}
