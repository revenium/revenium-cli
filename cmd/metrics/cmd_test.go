package metrics

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// fixtureMetricsResponse is a minimal valid JSON body for the fixture server.
// It exists only so that a REACHED handler fails on the request counter rather
// than on a transport error — the counter reading 0 is the actual assertion.
const fixtureMetricsResponse = `{"content":[],"totalElements":0}`

// TestMetricsCmdRejectsInvalidFromDateUnattached drives the shipped,
// package-level Cmd unattached — the regression test for the self-recursion
// defect carried in the Windows ledger as id 1 since Phase 22 (SAFE-02).
//
// Without the `root != Cmd` term in Cmd.PersistentPreRunE this does not fail,
// it CRASHES the test binary with `fatal error: stack overflow`, because Cmd
// is its own root when executed unattached and the root hook it delegates to
// is that same closure.
//
// A LEAF must be executed, never the bare parent: cobra's `!c.Runnable()` bail
// (command.go:955) returns flag.ErrHelp before the PersistentPreRunE loop at
// :985, so `Cmd.Execute()` with no args never enters the hook at all and would
// pass identically with the guard mutated away. `ai --from bogus` is the right
// trigger because normalizeDateFlag rejects the value inside the hook itself,
// so an error naming the value proves the hook demonstrably ran.
//
// cmd/metrics/dimensions.go's own tests exercise newDimensionsCmd() unattached,
// where root == c and its leaf-form guard is false, so they are unaffected by
// this change.
//
// Mutation proof (Task 3, mutation A) — observed, not asserted. Dropping the
// `root != Cmd` term from cmd/metrics/metrics.go and running
// `go test ./cmd/metrics/ -run TestMetricsCmdRejectsInvalidFromDateUnattached
// -count=1` printed:
//
//	runtime: goroutine stack exceeds 1000000000-byte limit
//	runtime: sp=0x14020260390 stack=[0x14020260000, 0x14040260000]
//	fatal error: stack overflow
//	FAIL	github.com/revenium/revenium-cli/cmd/metrics	1.433s
//
// exit 1. The test binary died rather than failing, which is exactly the
// unrecoverable process-level failure this guard exists to prevent.
func TestMetricsCmdRejectsInvalidFromDateUnattached(t *testing.T) {
	prevSilenceErrors := Cmd.SilenceErrors
	prevSilenceUsage := Cmd.SilenceUsage
	Cmd.SilenceErrors = true
	Cmd.SilenceUsage = true
	fromFlag = ""
	toFlag = ""
	t.Cleanup(func() {
		Cmd.SilenceErrors = prevSilenceErrors
		Cmd.SilenceUsage = prevSilenceUsage
		fromFlag = ""
		toFlag = ""
		Cmd.SetArgs(nil)
	})

	// requestCount is written on the httptest server's goroutine and read on
	// this test's goroutine, and `defer srv.Close()` runs AFTER the assertion
	// below — so there is no happens-before edge between the write and the
	// read. In the passing case no request occurs and nothing is reported; in
	// exactly the failure case this assertion exists to detect, a plain int
	// would surface under -race as a data race in this file instead of the
	// message the test was written to deliver, and the diagnosis would be lost
	// at the moment it is needed (26-REVIEW.md WR-04).
	//
	// An atomic is taken over WR-04's other offered fix — asserting after an
	// explicit early srv.Close() — because `defer srv.Close()` is the idiom
	// every sibling guard test in cmd/ uses, and moving one test off it would
	// make this file diverge from cmd/squads/cmd_test.go, the shipped model
	// D-26-09 says these tests mirror, for a reason no reader could infer from
	// the diff. An atomic changes the counter's type and nothing else about
	// the test's shape.
	var requestCount atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureMetricsResponse)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"ai", "--from", "bogus"})
	err := Cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "not valid")
	assert.Equal(t, int64(0), requestCount.Load(), "an invalid --from must fail before any HTTP request")
}

// TestMetricsCmdDelegatesToAttachedRoot is the other half of the `root != Cmd`
// guard, and the direction with the worse failure mode. TestMetricsCmdRejects-
// InvalidFromDateUnattached holds the guard against being DROPPED; this holds
// it against being drawn too BROADLY.
//
// When a real root sits above Cmd — as rootCmd does in the shipped binary —
// that root's PersistentPreRunE must still run, because it is what initializes
// config and cmd.APIClient. A guard drawn too broadly would skip it, leaving
// `revenium metrics ai` calling into a nil cmd.APIClient: a nil-pointer panic
// in the shipped binary that no other test in this package observes.
//
// SAFE-01's TestAllCommandPackagesRegistered cannot cover this: it tests
// registration completeness, not whether the root hook fires.
//
// Mutation proof (Task 3, mutation B) — observed, not asserted. Making the
// delegation block unreachable in cmd/metrics/metrics.go printed:
//
//	--- FAIL: TestMetricsCmdDelegatesToAttachedRoot (0.00s)
//	    cmd_test.go:130:
//	        Error:    Should be true
//	        Messages: the attached root's PersistentPreRunE must still run
//	FAIL	github.com/revenium/revenium-cli/cmd/metrics	0.272s
func TestMetricsCmdDelegatesToAttachedRoot(t *testing.T) {
	prevSilenceErrors := Cmd.SilenceErrors
	prevSilenceUsage := Cmd.SilenceUsage
	Cmd.SilenceErrors = true
	Cmd.SilenceUsage = true
	fromFlag = ""
	toFlag = ""
	t.Cleanup(func() {
		Cmd.SilenceErrors = prevSilenceErrors
		Cmd.SilenceUsage = prevSilenceUsage
		fromFlag = ""
		toFlag = ""
		Cmd.SetArgs(nil)
	})

	// BROUGHT TO PARITY WITH THE cmd/meter SIBLING (26-REVIEW.md WR-08). This
	// test used to stand up no fixture server, set neither cmd.APIClient nor
	// cmd.Output, and carry no request counter — while its sibling
	// TestMeterCmdDelegatesToAttachedRoot does all three, and the comment at the
	// head of this file explains at length that the counter exists so a
	// regression produces "the message the test was written to deliver" rather
	// than a lost diagnosis.
	//
	// The cost was not theoretical. The regression this test's neighbour is
	// named for is normalizeDateFlag ceasing to run in the hook; with it removed,
	// the `ai` leaf reached RunE with cmd.APIClient nil and the run printed:
	//
	//	--- FAIL: TestMetricsCmdDelegatesToAttachedRoot (0.00s)
	//	panic: runtime error: invalid memory address or nil pointer dereference
	//	[signal SIGSEGV: segmentation violation code=0x2 addr=0x0]
	//	  api.(*Client).resolveURL(0x0, ...) internal/api/client.go:73
	//	  metrics.newAICmd.func1(...) cmd/metrics/ai.go:45
	//
	// The test binary died. That is the same unrecoverable process-level failure
	// the `root != Cmd` guard exists to prevent, arrived at from the other side —
	// and in file order rather than in isolation it is worse still, because
	// cmd.APIClient then points at whatever the previous test left behind and the
	// leaf makes a real request against a closed listener.
	//
	// With the fixture server in place the same mutation prints the intended
	// diagnosis instead:
	//
	//	--- FAIL: TestMetricsCmdDelegatesToAttachedRoot (0.00s)
	//	    Error:    An error is expected but got nil.
	//	    Messages: normalizeDateFlag must run inside the hook ...
	//
	// atomic.Int64 for the same reason as in
	// TestMetricsCmdRejectsInvalidFromDateUnattached above; see that comment.
	var requestCount atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureMetricsResponse)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	rootRan := false
	parent := &cobra.Command{
		Use:           "revenium",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			rootRan = true
			return nil
		},
	}
	parent.AddCommand(Cmd)
	// Load-bearing: Cmd is package-global, so without this it stays parented
	// under this synthetic root for every subsequent test in the package.
	t.Cleanup(func() { parent.RemoveCommand(Cmd) })

	parent.SetOut(&buf)
	parent.SetErr(&buf)
	parent.SetArgs([]string{"metrics", "ai", "--from", "bogus"})
	err := parent.Execute()

	assert.True(t, rootRan, "the attached root's PersistentPreRunE must still run")
	require.Error(t, err, "normalizeDateFlag must run inside the hook, so an invalid --from fails before the leaf's RunE. Without it the leaf reaches the API client instead, and this assertion is the last thing standing between that and a nil-pointer panic that kills the test binary.")
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "not valid")
	assert.Equal(t, int64(0), requestCount.Load(), "an invalid --from must fail before any HTTP request")
}
