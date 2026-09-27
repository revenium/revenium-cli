package teams

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

// ---------------------------------------------------------------------------
// TEAM-09 — `revenium teams pr-health set <team-id> --aging-days N --rotting-days M`.
//
// Every test carries the TestPrHealthSet prefix: the plan's gate runs
// `-run TestPrHealthSet` and counts PASS lines, and a selector matching nothing
// exits 0 while proving nothing. For the same reason no test here uses t.Run —
// `go test -v` prints an extra indented PASS line per subtest, which the gate's
// substring grep would also count.
//
// requestCounter (mutex-guarded, and the mutex is load-bearing under -race) is
// declared once in pr_health_get_test.go and shared by both files.
// ---------------------------------------------------------------------------

// prHealthSetStub is a hermetic stub for the pr-health settings path that keeps
// SEPARATE per-method counters. The GET counter is what makes D-31-03's
// no-read-before-write claim directly assertable rather than merely asserted in
// prose: a GET-then-merge regression turns it non-zero.
type prHealthSetStub struct {
	srv          *httptest.Server
	gets         requestCounter
	puts         requestCounter
	total        requestCounter
	receivedBody map[string]interface{}
	receivedPath string
}

func newPrHealthSetStub(t *testing.T, responseBody string) *prHealthSetStub {
	t.Helper()
	stub := &prHealthSetStub{}
	stub.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.total.inc()
		switch r.Method {
		case http.MethodGet:
			stub.gets.inc()
		case http.MethodPut:
			stub.puts.inc()
		}
		stub.receivedPath = r.URL.Path
		if body, err := io.ReadAll(r.Body); err == nil && len(body) > 0 {
			_ = json.Unmarshal(body, &stub.receivedBody)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, responseBody)
	}))
	t.Cleanup(stub.srv.Close)
	return stub
}

// wirePrHealthSet points the package-level client and formatter at the stub.
func wirePrHealthSet(stub *prHealthSetStub, buf *bytes.Buffer) {
	cmd.APIClient = api.NewClient(stub.srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(buf, buf, false, false)
}

// execPrHealthSet runs the set command with args, capturing all output in buf.
func execPrHealthSet(buf *bytes.Buffer, args ...string) error {
	c := newPrHealthSetCmd()
	c.SetOut(buf)
	c.SetErr(buf)
	c.SilenceUsage = true
	c.SetArgs(args)
	return c.Execute()
}

// TestPrHealthSetSendsBothThresholds is the happy path: one PUT carrying both
// integers, no GET before it, and the PUT's own 200 re-rendered through the
// shared renderer (D-31-05).
func TestPrHealthSetSendsBothThresholds(t *testing.T) {
	stub := newPrHealthSetStub(t, `{"agingDays":14,"rottingDays":30}`)

	var buf bytes.Buffer
	wirePrHealthSet(stub, &buf)

	require.NoError(t, execPrHealthSet(&buf, "team-1", "--aging-days", "14", "--rotting-days", "30"))

	assert.Equal(t, "/v2/api/teams/team-1/settings/pr-health", stub.receivedPath)
	assert.Equal(t, 1, stub.puts.count(), "exactly one PUT")
	assert.Equal(t, 0, stub.gets.count(), "the command must never read before it writes (D-31-03)")

	require.Len(t, stub.receivedBody, 2, "the body carries exactly the two required properties")
	aging, ok := stub.receivedBody["agingDays"].(float64)
	require.True(t, ok, "agingDays must decode as a JSON number, not a string")
	assert.Equal(t, float64(14), aging)
	rotting, ok := stub.receivedBody["rottingDays"].(float64)
	require.True(t, ok, "rottingDays must decode as a JSON number, not a string")
	assert.Equal(t, float64(30), rotting)

	out := buf.String()
	assert.Contains(t, out, "agingDays")
	assert.Contains(t, out, "14")
	assert.Contains(t, out, "rottingDays")
	assert.Contains(t, out, "30")
}

// TestPrHealthSetRejectsAgingGreaterThanRotting is ROADMAP SC1's named guard.
//
// The counter assertion PRECEDES require.Error deliberately (the cmd/jobs
// economics-set precedent): a test that only asserts an error came back still
// passes when the guard is deleted and the server 400s. The zero request count
// is the assertion that names the actual property — refused at zero HTTP cost.
func TestPrHealthSetRejectsAgingGreaterThanRotting(t *testing.T) {
	stub := newPrHealthSetStub(t, `{}`)

	var buf bytes.Buffer
	wirePrHealthSet(stub, &buf)

	err := execPrHealthSet(&buf, "team-1", "--aging-days", "40", "--rotting-days", "30")

	assert.Equal(t, 0, stub.total.count(), "no request of any method may be sent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--aging-days")
	assert.Contains(t, err.Error(), "40")
	assert.Contains(t, err.Error(), "--rotting-days")
	assert.Contains(t, err.Error(), "30")
}

// TestPrHealthSetRejectsEqualThresholds pins the adjacency edge: the schema
// prose is "Must be lower than" / "Must be higher than", and equality satisfies
// neither, so `>=` — not `>` — is the refusal.
func TestPrHealthSetRejectsEqualThresholds(t *testing.T) {
	stub := newPrHealthSetStub(t, `{}`)

	var buf bytes.Buffer
	wirePrHealthSet(stub, &buf)

	err := execPrHealthSet(&buf, "team-1", "--aging-days", "30", "--rotting-days", "30")

	assert.Equal(t, 0, stub.total.count(), "no request of any method may be sent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--aging-days")
	assert.Contains(t, err.Error(), "--rotting-days")
}

// TestPrHealthSetRequiresBothFlags proves an under-specified invocation is
// refused by COBRA's required-flag validation — not by the arithmetic ordering
// guard reading two flags the operator never supplied.
//
// The comment this replaces asserted exactly that, and the assertion was false
// when it was written (code review WR-01). Cobra runs PreRunE at command.go:969
// and ValidateRequiredFlags at command.go:977, so the shipped PreRunE compared
// two unset zeros and answered `--aging-days (0) must be less than
// --rotting-days (0)`: the operator was told two values they never supplied were
// out of order, and both MarkFlagRequired calls were unreachable dead code.
// 31-08 inverts it — the guard yields when either flag was not Changed, so
// cobra's own `required flag(s) ... not set` is what an operator reads, and the
// claim above is true for the first time.
//
// The assertion had to change with it. The shipped test asserted the message
// contained `rotting-days`, which the ARITHMETIC error contains too — it passed
// for the wrong reason and would have kept certifying the guard after the guard
// was deleted. `required flag` is the discriminator the arithmetic error cannot
// contain; the negative assertion on `must be less than` stops that guard
// silently reclaiming either case.
//
// Both under-specified shapes are covered inline as a loop, not as t.Run
// subtests, for the PASS-count reason in this file's header. The zero-request
// assertion precedes require.Error in each, per this file's
// counters-before-require.Error discipline.
func TestPrHealthSetRequiresBothFlags(t *testing.T) {
	for _, args := range [][]string{
		{"team-1", "--aging-days", "14"},
		{"team-1"},
	} {
		stub := newPrHealthSetStub(t, `{}`)

		var buf bytes.Buffer
		wirePrHealthSet(stub, &buf)

		err := execPrHealthSet(&buf, args...)

		assert.Equal(t, 0, stub.total.count(), "no request of any method may be sent for %v", args)

		require.Error(t, err, "%v must be refused", args)
		assert.Contains(t, err.Error(), "required flag", "%v must be refused by cobra's required-flag validation, not by the ordering guard", args)
		assert.NotContains(t, err.Error(), "must be less than", "%v must not reach the arithmetic ordering guard", args)
	}
}

// TestPrHealthSetAllowsZeroAndNegative proves no client-side floor was invented.
// The schema declares no lower bound on either property, so the server owns that
// rule; refusing a call the server would accept is the same invention as
// inventing a default, in the opposite direction.
//
// Written as a loop rather than t.Run subtests for the PASS-count reason above.
func TestPrHealthSetAllowsZeroAndNegative(t *testing.T) {
	for _, aging := range []string{"0", "-1"} {
		stub := newPrHealthSetStub(t, `{"agingDays":0,"rottingDays":5}`)

		var buf bytes.Buffer
		wirePrHealthSet(stub, &buf)

		err := execPrHealthSet(&buf, "team-1", "--aging-days", aging, "--rotting-days", "5")

		require.NoError(t, err, "--aging-days %s must reach the server", aging)
		assert.Equal(t, 1, stub.puts.count(), "--aging-days %s must produce one PUT", aging)
		assert.Equal(t, 0, stub.gets.count(), "no read before the write")
	}
}

// TestPrHealthSetDryRun pins the dry-run render contract. cmd.DryRun() has no
// exported setter (the cmd/teams coding-assistant-filter precedent), so the
// convention is to invoke dryrun.Render directly with the exact verb, resource
// label, path and body the RunE passes.
//
// The Annotations/cmd.DryRun() pairing is an invariant, not a nicety: the
// phase-27 Critical was a command that declared itself mutating and never called
// the gate, so its --dry-run performed the real write.
func TestPrHealthSetDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	body := map[string]interface{}{"agingDays": 14, "rottingDays": 30}

	require.NoError(t, dryrun.Render(out, "update", "PR health settings", "/v2/api/teams/team-1/settings/pr-health", body))

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: update PR health settings")
	assert.Contains(t, rendered, "/v2/api/teams/team-1/settings/pr-health")
	assert.Contains(t, rendered, "No changes were made.")

	// The command really is annotated mutating — without it the global
	// --dry-run flag never reaches this command at all.
	assert.Equal(t, "true", newPrHealthSetCmd().Annotations["mutating"])
}
