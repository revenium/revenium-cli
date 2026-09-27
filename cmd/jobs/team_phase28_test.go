package jobs

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// ---------------------------------------------------------------------------
// D-28-12 — the team guard on ALL THREE surfaces Phase 28 adds, pinned at
// runtime by a request counter.
//
// WHY THIS TABLE EXISTS BESIDE sc5Cases, AND WHAT IT DELIBERATELY OVERLAPS.
// Phase 27's TestSC5TableCoversEveryGuardedCommand discovers guarded commands
// by walking typesCmd.Commands() — that is, `jobs types` sub-resources only.
// Of this phase's three new surfaces:
//
//   - `types facts append` IS discovered there, and 28-01 added its sc5Cases
//     row when the guard fired. Its rows below are therefore a DELIBERATE
//     overlap, kept rather than dropped so that this table reads as the
//     complete D-28-12 statement — "each of the three, the same way" — instead
//     of as two of them plus a cross-reference a later reader has to chase.
//     The cost is one duplicated subtest per test; the benefit is that
//     deleting the sc5Cases row cannot silently uncover this command.
//   - `outcome-metrics` is a direct child of Cmd, so sc5Cases cannot reach it
//     and does not. Before this table its requireTeam() was pinned only by a
//     source-order criterion on the file's text. These rows are the first
//     runtime evidence for it.
//   - `delete` is likewise a direct child of Cmd. Its row carries TWO ids on
//     purpose: the guard sits on the BULK arm only, so a one-id row would
//     drive the deliberately-unguarded single-item arm and pass for entirely
//     the wrong reason.
//
// THE SINGLE-ID DELETE ARM IS DELIBERATELY NOT GUARDED (operator resolution of
// RESEARCH Open Question 1). deleteOneJob keeps D-28-03's byte-for-byte
// promise, and the evidence that the placement holds is that the three shipped
// single-id tests — TestDeleteJobWithYes, TestDeleteJobQuiet and
// TestDeleteJobJSONMode — still pass unmodified against an EMPTY team id. If
// the guard ever migrates up into the dispatcher, those three turn red.
// ---------------------------------------------------------------------------

// phase28TeamCase is one row: a command surface Phase 28 adds that must refuse
// before the wire with no team, and must put teamId on the wire with one.
type phase28TeamCase struct {
	// name is the subtest name and reads as the operator-facing command.
	name string

	// path is the command's position under jobs.Cmd. It is asserted against
	// CommandPath() so a row cannot drift onto a partial match of its parent.
	path []string

	// wantPath is the URL path this command must reach on the happy path.
	wantPath string

	// args resets this command's sticky state and returns the FULL argument
	// slice. A cobra command registered once at package init carries its flag
	// values and pflag Changed bits across every test in the binary, so a row
	// that sets flags must clear them first or it inherits whatever the
	// previous test left behind.
	args func(t *testing.T) []string

	// respond serves the request this command issues on the happy path.
	respond func(w http.ResponseWriter, r *http.Request)
}

// bulkDeletePhase28OK confirms BOTH named ids, so reportBulkDelete's
// set-membership shortfall check finds nothing missing and the happy-path row
// is not red for a reason that has nothing to do with the team guard.
const bulkDeletePhase28OK = `{"requestedCount":2,"deletedCount":2,` +
	`"deletedIds":["loan-app-1","loan-app-2"],"message":"Deleted"}`

var phase28TeamCases = []phase28TeamCase{
	{
		name:     "types facts append",
		path:     []string{"types", "facts", "append"},
		wantPath: "/v2/api/jobs/types/acme-review/facts",
		args: func(t *testing.T) []string {
			t.Helper()
			resetFactsAppendFlags(t)
			// A VALID entry array on purpose. An invalid one would be refused
			// by validateFactEntries before any request, and the negative half
			// would then pass on an error the team guard had nothing to do
			// with — the exact green-for-nothing shape the counter exists to
			// refuse.
			return []string{
				"types", "facts", "append", "acme-review",
				"--file", writeTempFactsFile(t, twoFactEntries),
			}
		},
		respond: func(w http.ResponseWriter, r *http.Request) {
			// 201 and NOTHING else: this endpoint declares no content block at
			// all. Writing a body here would make the row pass against a
			// facts_append.go that decoded the response.
			w.WriteHeader(http.StatusCreated)
		},
	},
	{
		name:     "outcome-metrics",
		path:     []string{"outcome-metrics"},
		wantPath: "/v2/api/jobs/loan-app-12345/outcome/metrics",
		args: func(t *testing.T) []string {
			t.Helper()
			resetOutcomeMetricsFlags(t)
			return []string{
				"outcome-metrics", "loan-app-12345",
				"--file", writeTempFactsFile(t, twoOutcomeMetricEntries),
			}
		},
		respond: func(w http.ResponseWriter, r *http.Request) {
			// Same empty-201 contract as facts append.
			w.WriteHeader(http.StatusCreated)
		},
	},
	{
		name:     "delete (bulk arm, two ids)",
		path:     []string{"delete"},
		wantPath: "/v2/api/jobs",
		args: func(t *testing.T) []string {
			t.Helper()
			// --yes is a ROOT persistent flag; jobs.Cmd has no parent in this
			// test binary, so it cannot be passed as an argument and must be
			// written through the root's pflag.Value — the same variable
			// cmd.YesMode() reads.
			setRootFlag(t, "yes", "true")
			// A pipe is never a terminal, which makes the confirmation rung a
			// FACT of this test rather than an accident of how `go test` was
			// invoked. Without it, a toolchain that forwards a terminal to the
			// test binary would drop the happy-path row into the interactive
			// prompt and block on a read.
			withNonTTYStdin(t)
			// TWO ids: the guard is on the bulk arm. One id would exercise the
			// deliberately-unguarded single-item arm.
			return []string{"delete", "loan-app-1", "loan-app-2"}
		},
		respond: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(bulkDeletePhase28OK))
		},
	},
}

// requirePhase28Registration checks that the row's path resolves to the command
// the row means to drive. Cmd.Find returns the deepest partial match it could
// reach, so without this a row whose path drifted would report a perfectly
// green result about a RunE nobody meant to test.
func requirePhase28Registration(t *testing.T, tc phase28TeamCase) {
	t.Helper()
	registered, leftover, err := Cmd.Find(tc.path)
	require.NoError(t, err, "the command path %v must resolve", tc.path)
	require.Empty(t, leftover, "the whole path %v must be consumed by the command tree", tc.path)
	require.Equal(t, "jobs "+strings.Join(tc.path, " "), registered.CommandPath(),
		"this row must drive the command at %v, not a partial match on its parent", tc.path)
}

// TestRequireTeamPhase28 is the negative half of D-28-12 over all three new
// surfaces: with no team resolved from flag, env or config, each must refuse
// AND issue zero HTTP requests.
//
// THE LOAD-BEARING ASSERTION IS THE COUNTER, NOT THE ERROR. A bare
// require.Error is satisfied by any unrelated failure — a missing fixture file,
// a malformed entry array, a decode error on a team-less request the server
// answered badly. 27-01 ran exactly that mutation and watched require.Error
// stay GREEN while the team-less request reached the stub. Only "no request was
// issued" is evidence that the guard fired before the wire.
//
// Mutation contract (28-VALIDATION.md): deleting the requireTeam() call from
// any ONE of the three RunE bodies must turn that command's subtest red on its
// count assertion, and leave the other two green.
func TestRequireTeamPhase28(t *testing.T) {
	for _, tc := range phase28TeamCases {
		t.Run(tc.name, func(t *testing.T) {
			requirePhase28Registration(t, tc)

			var hits requestCounter
			srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
				hits.inc()
				assert.Fail(t, "no HTTP request may be issued when no team resolves",
					"the guard let %s %s through", r.Method, r.URL.Path)
			})

			// A --dry-run leaked from an earlier test would make every one of
			// these rows pass with zero requests for a reason that has nothing
			// to do with the guard. Cleared explicitly, and restored by
			// setRootFlag's own cleanup.
			setRootFlag(t, "dry-run", "false")
			args := tc.args(t)

			var buf bytes.Buffer
			cmd.APIClient = stubClient(srv.URL, "")
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			err := execJobs(&buf, args...)

			// The counter asserts FIRST, and with assert rather than require:
			// under a mutation that lets the request through, a leading
			// require.Error aborts the subtest and the only assertion that ever
			// runs is "an error is expected but got nil", which says nothing
			// about whether a request escaped (27-05 deviation 4, 28-03
			// deviation 2).
			//
			// assert.Equal, not the Zero form: Zero drops the expected/actual
			// pair, which is the whole diagnostic content this test delivers.
			assert.Equal(t, 0, hits.count(),
				"the guard must refuse before any request is issued")

			require.Error(t, err, "a team-less invocation must fail, not proceed")

			// cmd/root.go sets SilenceUsage and SilenceErrors, so cobra appends
			// no usage block — the message has to name every resolution source
			// itself, or the operator is told what is wrong and not how to fix
			// it.
			assert.Contains(t, err.Error(), "--team-id")
			assert.Contains(t, err.Error(), "REVENIUM_TEAM_ID")
			assert.Contains(t, err.Error(), "config set team-id")
		})
	}
}

// TestTeamIDSentPhase28 is the positive half of D-28-12 over the same three
// surfaces: when a team IS resolved, the parameter all three endpoints declare
// required actually reaches the query string.
//
// Refusing without a team proves nothing on its own. A guard that returns nil
// while the client drops the parameter satisfies TestRequireTeamPhase28 and
// still issues an unscoped request — which is why the guard and the wire
// behaviour are two separate tests rather than two assertions in one.
func TestTeamIDSentPhase28(t *testing.T) {
	for _, tc := range phase28TeamCases {
		t.Run(tc.name, func(t *testing.T) {
			requirePhase28Registration(t, tc)

			var hits requestCounter
			var mu sync.Mutex
			var teams, paths []string
			srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
				hits.inc()
				// httptest serves each request on its own goroutine, so the
				// recording is mutex-guarded — an unguarded slice append would
				// trip the race detector and report a data race instead of the
				// guard regression this test exists to diagnose.
				mu.Lock()
				teams = append(teams, r.URL.Query().Get("teamId"))
				paths = append(paths, r.URL.Path)
				mu.Unlock()
				tc.respond(w, r)
			})

			setRootFlag(t, "dry-run", "false")
			args := tc.args(t)

			var buf bytes.Buffer
			cmd.APIClient = stubClient(srv.URL, "team-28")
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			err := execJobs(&buf, args...)

			require.NoError(t, err, "output was: %s", buf.String())
			assert.Equal(t, 1, hits.count(),
				"exactly one request per invocation — none of the three has a per-entry loop")

			mu.Lock()
			gotTeams := append([]string(nil), teams...)
			gotPaths := append([]string(nil), paths...)
			mu.Unlock()

			require.NotEmpty(t, gotTeams, "the command must reach the server")
			for i, team := range gotTeams {
				assert.Equal(t, "team-28", team,
					"request %d of %d carried no teamId on the query string", i+1, len(gotTeams))
			}
			assert.Equal(t, []string{tc.wantPath}, gotPaths,
				"the row must exercise its own endpoint — a teamId on the wrong path is not this row's evidence")
		})
	}
}
