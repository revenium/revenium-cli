package jobs

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// sc5Case is one row of the SC5 table: a command that must refuse before the
// wire when no team resolves, and must put teamId on the wire when one does.
type sc5Case struct {
	// name is the subtest name and reads as the operator-facing command.
	name string

	// ctor is the constructor that builds this command. It is not
	// documentation: the tests assert that the command actually reached by
	// path is the one this constructor produces, so a row whose args drift
	// onto a different command fails rather than silently testing the wrong
	// RunE.
	ctor func() *cobra.Command

	// path is the command's position under jobs.Cmd.
	path []string

	// prepare resets this command's sticky flag state and returns the extra
	// arguments it needs to reach its guard. A cobra command registered once
	// at package init carries its flag values and pflag Changed bits across
	// every test in the binary (27-03, 27-05), so a row that sets flags must
	// clear them first or it inherits whatever the previous test left.
	prepare func(t *testing.T) []string

	// respond serves the request(s) this command issues on the happy path.
	respond func(w http.ResponseWriter, r *http.Request)

	// escapedPath is the WIRE-FORMAT path this command must produce for the
	// job type id "acme/review" — the id segment escaped, the separators
	// around it left literal.
	escapedPath string
}

// sc5Args builds the full argument slice for a row, for the given job type id.
// prepare runs here because a row that sets flags must clear the sticky state
// the previous test left before it sets its own.
func sc5Args(t *testing.T, tc sc5Case, id string) []string {
	t.Helper()
	args := append([]string{}, tc.path...)
	args = append(args, id)
	if tc.prepare != nil {
		args = append(args, tc.prepare(t)...)
	}
	return args
}

// sc5Cases covers ALL FOUR commands Phase 27 adds, one row each.
//
// One representative case would not do. requireTeam() is called per-RunE, so
// four RunE bodies need four cases; a single case leaves three commands free
// to lose their guard and stay green — which is exactly the shape Phase 26
// found five times, controls sitting green under the mutations they existed to
// catch.
//
// THE RESIDUAL, stated so a future reader sees it rather than inferring it was
// handled: this table closes FOUR INSTANCES of the missing-team class, not the
// class. internal/api/client.go appends teamId to the query string only when
// Client.TeamID is non-empty, so any command in any of the roughly twenty other
// packages that reaches a team-scoped endpoint still issues its request
// team-less and fails in whatever way the server chooses — silently, from the
// operator's point of view. A repo-wide guard needs a per-endpoint allowlist,
// because some endpoints legitimately require no team, and that is a Deferred
// Idea with its own phase (T-27-18, accepted). Adding a fifth command under
// `jobs types` without a guard is caught here by
// TestSC5TableCoversEveryGuardedCommand; adding one anywhere else is not caught
// by anything.
var sc5Cases = []sc5Case{
	{
		name:        "economics get",
		ctor:        newEconomicsGetCmd,
		path:        []string{"types", "economics", "get"},
		escapedPath: "/v2/api/jobs/types/acme%2Freview/economics",
		respond: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, economicsResourceBody)
		},
	},
	{
		name:        "economics set",
		ctor:        newEconomicsSetCmd,
		path:        []string{"types", "economics", "set"},
		escapedPath: "/v2/api/jobs/types/acme%2Freview/economics",
		prepare: func(t *testing.T) []string {
			t.Helper()
			resetEconomicsSetFlags(t)
			// A VALID, non-destructive document. The point of the negative
			// half is that the guard refuses before the file is ever read —
			// so if the guard were moved after readEconomicsInput this test
			// would still pass, and deleting the guard is what makes it
			// fail. A deliberately broken file would refuse for its own
			// reasons and pin nothing.
			//
			// It is the full Resource shape, which strips back to a document
			// identical to what the stub serves as the current contract, so
			// the positive half diffs as non-destructive and needs no --yes.
			return []string{"--file", writeEconomicsFile(t, economicsResourceBody)}
		},
		respond: func(w http.ResponseWriter, r *http.Request) {
			// The command issues a preflight GET and then the PUT. Both are
			// served, and both are checked for teamId by the caller.
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, economicsResourceBody)
		},
	},
	{
		name:        "baselines list",
		ctor:        newBaselinesListCmd,
		path:        []string{"types", "baselines", "list"},
		escapedPath: "/v2/api/jobs/types/acme%2Freview/baselines",
		respond: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"version":1,"costPerUnit":4.25,"currency":"USD","provenance":"MEASURED","declaredBy":"ops@acme.example","effectiveFrom":"2026-01-01T00:00:00Z"}]`)
		},
	},
	{
		name:        "baselines append",
		ctor:        newBaselinesAppendCmd,
		path:        []string{"types", "baselines", "append"},
		escapedPath: "/v2/api/jobs/types/acme%2Freview/baselines",
		prepare: func(t *testing.T) []string {
			t.Helper()
			resetAppendFlags(t)
			// --provenance is a LEGAL value on purpose. An illegal one would
			// be refused by validateBaselineProvenance, and the negative half
			// would then pass on an error the guard had nothing to do with.
			return []string{
				"--cost-per-unit", "4.25",
				"--provenance", "MEASURED",
				"--effective-from", "2026-01-07T00:00:00Z",
			}
		},
		respond: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"version":7,"costPerUnit":4.25,"currency":"USD","effectiveFrom":"2026-01-07T00:00:00Z"}`)
		},
	},
	{
		// Phase 28 / JOBS-15. TestSC5TableCoversEveryGuardedCommand is what
		// required this row the moment `jobs types facts` was registered —
		// the guard Phase 27 built for exactly this, working as designed.
		name:        "facts append",
		ctor:        newFactsAppendCmd,
		path:        []string{"types", "facts", "append"},
		escapedPath: "/v2/api/jobs/types/acme%2Freview/facts",
		prepare: func(t *testing.T) []string {
			t.Helper()
			resetFactsAppendFlags(t)
			// A VALID entry array on purpose. An invalid one would be
			// refused by validateFactEntries before any request, and the
			// negative half of SC5 would then pass on an error the team
			// guard had nothing to do with.
			return []string{"--file", writeTempFactsFile(t, twoFactEntries)}
		},
		respond: func(w http.ResponseWriter, r *http.Request) {
			// 201 and NOTHING else: this endpoint declares no content block
			// at all. Writing a body here would make the row pass against a
			// facts_append.go that decoded the response.
			w.WriteHeader(http.StatusCreated)
		},
	},
}

// requireRegisteredCommand checks that the command reached by tc.path is the
// one tc.ctor builds, so each row's constructor is load-bearing rather than a
// comment. A row whose args resolved to some other command would otherwise
// report a perfectly green result about a RunE nobody meant to test.
func requireRegisteredCommand(t *testing.T, tc sc5Case) {
	t.Helper()
	registered, _, err := Cmd.Find(tc.path)
	require.NoError(t, err, "the command path %v must resolve", tc.path)
	require.Equal(t, tc.ctor().Use, registered.Use,
		"the registered command at %v must be the one this row's constructor builds", tc.path)
	require.Equal(t, "jobs "+strings.Join(tc.path, " "), registered.CommandPath())
}

// TestRequireTeam is the negative half of SC5, over ALL FOUR commands: with no
// team resolved from flag, env or config, each must refuse AND issue zero HTTP
// requests.
//
// The load-bearing assertion is the request counter, not the error. A bare
// require.Error passes with the guard deleted — 27-01 ran exactly that
// mutation and watched require.Error stay GREEN while the team-less request
// reached the stub and failed with "failed to decode response: EOF". An
// unrelated error satisfying the same assertion is not evidence.
//
// Mutation contract: deleting the requireTeam() call from any ONE of the four
// RunE bodies must turn that command's subtest red on its count assertion.
func TestRequireTeam(t *testing.T) {
	for _, tc := range sc5Cases {
		t.Run(tc.name, func(t *testing.T) {
			requireRegisteredCommand(t, tc)

			var hits requestCounter
			srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
				hits.inc()
				assert.Fail(t, "no HTTP request may be issued when no team resolves",
					"the guard let %s %s through", r.Method, r.URL.Path)
			})

			args := sc5Args(t, tc, "acme-review")

			var buf bytes.Buffer
			cmd.APIClient = stubClient(srv.URL, "")
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			err := execJobs(&buf, args...)

			// The counter asserts FIRST and with assert, not require: under a
			// mutation that lets the request through, a leading require.Error
			// aborts the subtest and the only assertion that ever runs is
			// "an error is expected but got nil" — which says nothing about
			// whether a request escaped (27-05, deviation 4).
			//
			// assert.Equal, not assert.Zero: Zero drops the expected/actual
			// pair, which is the diagnostic content this test delivers.
			assert.Equal(t, 0, hits.count(),
				"the guard must refuse before any request is issued")

			require.Error(t, err, "a team-less invocation must fail, not proceed")
			// The root sets SilenceUsage and SilenceErrors, so cobra appends
			// no usage block — the message has to name every resolution
			// source itself.
			assert.Contains(t, err.Error(), "--team-id")
			assert.Contains(t, err.Error(), "REVENIUM_TEAM_ID")
			assert.Contains(t, err.Error(), "config set team-id")
		})
	}
}

// TestTeamIDSent is the positive half of SC5, over the same four commands: when
// a team IS resolved, the parameter the endpoints mark required actually
// reaches the wire.
//
// Refusing without a team proves nothing on its own — Phase 26 found teams
// asserting the refusal and skipping this half, leaving the happy path free to
// send no teamId at all. Every request the command issues is checked, not just
// the first: `economics set` issues a preflight GET and then a PUT, and a
// teamId present on one and absent from the other is still a bug.
func TestTeamIDSent(t *testing.T) {
	for _, tc := range sc5Cases {
		t.Run(tc.name, func(t *testing.T) {
			requireRegisteredCommand(t, tc)

			var hits requestCounter
			var mu sync.Mutex
			var received []string
			srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
				hits.inc()
				mu.Lock()
				received = append(received, r.URL.Query().Get("teamId"))
				mu.Unlock()
				tc.respond(w, r)
			})

			args := sc5Args(t, tc, "acme-review")

			var buf bytes.Buffer
			cmd.APIClient = stubClient(srv.URL, "team-1")
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			err := execJobs(&buf, args...)

			require.NoError(t, err, "output was: %s", buf.String())
			assert.GreaterOrEqual(t, hits.count(), 1, "the command must reach the server")

			mu.Lock()
			got := append([]string{}, received...)
			mu.Unlock()
			require.NotEmpty(t, got)
			for i, teamID := range got {
				assert.Equal(t, "team-1", teamID,
					"request %d of %d carried no teamId on the query string", i+1, len(got))
			}
		})
	}
}

// TestSC5TableCoversEveryGuardedCommand fails when a sub-resource command is
// added under `jobs types` and not added to sc5Cases.
//
// Without it, the table is a snapshot of what happened to exist the day it was
// written: a fifth command could ship with no guard and nothing anywhere would
// report it. This is the one direction of T-27-18 that IS closed — inside the
// `jobs types` sub-resource tree. Outside it, see the residual noted above
// sc5Cases.
func TestSC5TableCoversEveryGuardedCommand(t *testing.T) {
	covered := map[string]bool{}
	for _, tc := range sc5Cases {
		covered[strings.Join(tc.path, " ")] = true
	}

	discovered := 0
	for _, parent := range typesCmd.Commands() {
		// A sub-resource is a child of `jobs types` that has children of its
		// own — economics and baselines today. Discovering them rather than
		// naming them means a THIRD sub-resource is covered too.
		if len(parent.Commands()) == 0 || parent.IsAdditionalHelpTopicCommand() {
			continue
		}
		for _, child := range parent.Commands() {
			if !child.Runnable() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			if child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			discovered++
			key := "types " + parent.Name() + " " + child.Name()
			assert.True(t, covered[key],
				"`jobs %s` reaches a team-scoped endpoint but has no row in sc5Cases — "+
					"add one, or it is free to ship without requireTeam()", key)
		}
	}

	assert.Equal(t, len(sc5Cases), discovered,
		"sc5Cases must have exactly one row per registered `jobs types` sub-resource command")
}

// TestPathEscapedOnEveryCommand is T-27-04's runtime proof, over the same four
// commands.
//
// The threat is not theoretical. internal/api.resolveURL appends teamId after a
// naive `strings.Contains(url, "?")` check, so a job type id that smuggled a
// "?" or "&" through would land a second, argument-chosen teamId on the query
// string — a tenancy selector chosen by the argument rather than by the
// operator's configuration. cmd.ValidResourceID rejects ?, &, #, %, ../, ..\
// and control characters, but it permits a bare "/", and url.PathEscape is what
// stops that "/" from opening a new path segment.
//
// Until this test existed, T-27-04's only evidence in this phase was an
// acceptance grep asserting the string `url.PathEscape(` appears in the source.
// A grep cannot tell whether the call is applied to the id or to something
// else, and it stays green if the result is discarded. This asserts the
// WIRE-FORMAT path, which is the only place the answer is observable.
//
// EscapedPath(), not Path(): server-side, r.URL.Path collapses %2F back to "/",
// so an assertion on Path() passes identically whether PathEscape was applied
// or not — the same green-for-nothing shape this phase exists to refuse. The
// pattern is TestROIPathEscape's (roi_test.go:125).
func TestPathEscapedOnEveryCommand(t *testing.T) {
	for _, tc := range sc5Cases {
		t.Run(tc.name, func(t *testing.T) {
			requireRegisteredCommand(t, tc)

			var mu sync.Mutex
			var escaped []string
			srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				escaped = append(escaped, r.URL.EscapedPath())
				mu.Unlock()
				tc.respond(w, r)
			})

			var buf bytes.Buffer
			cmd.APIClient = stubClient(srv.URL, "team-1")
			cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

			err := execJobs(&buf, sc5Args(t, tc, "acme/review")...)

			require.NoError(t, err, "output was: %s", buf.String())

			mu.Lock()
			got := append([]string{}, escaped...)
			mu.Unlock()
			require.NotEmpty(t, got, "the command must reach the server")
			for i, p := range got {
				assert.Equal(t, tc.escapedPath, p,
					"request %d of %d: the id segment must be escaped (%%2F), "+
						"the separators around it left literal", i+1, len(got))
			}
		})
	}
}
