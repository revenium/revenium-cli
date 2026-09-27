package jobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEconomicsRegisteredUnderJobsTypes closes the gap every other cmd/jobs
// test leaves open: they all construct their command directly, so the package
// initializer that wires the tree is never exercised. A nil-parent or
// lexical-ordering regression in that wiring would ship with a fully green
// suite and surface only as `unknown command` in an operator's terminal.
//
// This test walks the package-level Cmd — the same object main.go registers —
// so a registration regression turns cmd/jobs red instead.
func TestEconomicsRegisteredUnderJobsTypes(t *testing.T) {
	resolved, _, err := Cmd.Find([]string{"types", "economics", "get"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "get", resolved.Name())
	// CommandPath starts at "jobs" here, not "revenium": main.go attaches Cmd
	// to the root command, and that attachment is outside this package.
	assert.Equal(t, "jobs types economics get", resolved.CommandPath(),
		"the child must hang off jobs types, not off some other parent")

	// A permissive parent would resolve anything, so pin the negative too.
	// Find returns the deepest command it could match — here `types`, with
	// "bogus" left over — and delivers no error itself; the rejection comes
	// from that parent's Args validator, which is exactly the cobra.NoArgs
	// declaration CF-17 pins.
	parent, leftover, err := Cmd.Find([]string{"types", "bogus"})
	require.NoError(t, err)
	assert.Equal(t, "types", parent.Name())
	assert.Equal(t, []string{"bogus"}, leftover)
	assert.Error(t, parent.ValidateArgs(leftover),
		`"jobs types bogus" must be rejected, not silently accepted`)
}

// TestJobsTypesStillTakesNoArgs guards CF-17 against a future edit that
// "fixes" NoArgs away now that `jobs types` has children. It needs no
// restructuring — a NoArgs parent dispatches children just fine, and bare
// `jobs types` still runs — so this is a pin, not a workaround.
func TestJobsTypesStillTakesNoArgs(t *testing.T) {
	types, _, err := Cmd.Find([]string{"types"})
	require.NoError(t, err)
	require.Equal(t, "types", types.Name())
	require.NotNil(t, types.Args, "jobs types must keep an explicit Args validator (CF-17)")

	assert.NoError(t, types.Args(types, []string{}),
		"bare `jobs types` must still list job types")
	assert.Error(t, types.Args(types, []string{"foo"}),
		"`jobs types foo` must stay rejected (CF-17)")

	assert.True(t, types.HasSubCommands(),
		"jobs types must dispatch children while keeping NoArgs")
}

// ---------------------------------------------------------------------------
// Phase 28 — the tree pins for the two new command surfaces, plus the pin that
// this phase did NOT convert a released leaf into a parent.
//
// Named with the TestRegistration prefix deliberately: 28-VALIDATION.md
// prescribes the run selector `go test ./cmd/jobs/ -run TestRegistration`, and
// no shipped test in this package carries that prefix. A selector that matches
// nothing exits 0 while proving nothing, which is the one outcome a
// registration pin must never produce.
// ---------------------------------------------------------------------------

// TestRegistrationFactsAppend pins `jobs types facts append` (JOBS-15) at its
// exact command path.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a name-only
// assertion stays green on a command hung off the wrong parent — and the
// registration in types.go is exactly the wiring that could put it there,
// because facts.go declares no initializer of its own.
func TestRegistrationFactsAppend(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"types", "facts", "append"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "append", resolved.Name())
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	// CommandPath starts at "jobs" here, not "revenium": main.go attaches Cmd
	// to the root command, and that attachment is outside this package.
	assert.Equal(t, "jobs types facts append", resolved.CommandPath(),
		"append must hang off jobs types facts, not off some other parent")

	// The negative half: a permissive tree would resolve anything. `bogus` must
	// stay LEFT OVER on the facts parent rather than being absorbed by the one
	// runnable child, which is what proves `append` is reachable only by its own
	// name.
	//
	// Deliberately NOT asserted here: that facts.ValidateArgs(leftover) errors.
	// factsCmd declares no Args — like its two shipped siblings `baselines` and
	// `economics` — and cobra's ValidateArgs falls back to ArbitraryArgs, which
	// returns nil for any input (cobra v1.10.2, command.go). See the summary's
	// deviation record and deferred-items.md.
	parent, bogusLeftover, err := Cmd.Find([]string{"types", "facts", "bogus"})
	require.NoError(t, err)
	assert.Equal(t, "facts", parent.Name())
	assert.Equal(t, []string{"bogus"}, bogusLeftover,
		"`bogus` must not resolve onto append — the parent keeps it as an unmatched argument")
	assert.False(t, parent.Runnable(),
		"the facts parent has no RunE, so an unmatched argument cannot execute anything")

	// The reachable rejection: the LEAF's own arity validator, reached through
	// the real registration path rather than through a directly constructed
	// command. A second positional is refused.
	assert.NoError(t, resolved.ValidateArgs([]string{"acme-review"}))
	assert.Error(t, resolved.ValidateArgs([]string{"acme-review", "extra"}),
		"`jobs types facts append <type>` takes exactly one job type")
}

// TestRegistrationOutcomeMetrics pins `jobs outcome-metrics` (JOBS-16) at its
// exact command path.
//
// This command registers with its own init() on the package-level Cmd, the
// opposite of facts.go. The CommandPath() equality is what tells the flat
// hyphenated sibling apart from a child of the shipped `outcome` leaf — the
// two are indistinguishable to a Name() assertion.
func TestRegistrationOutcomeMetrics(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"outcome-metrics"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "outcome-metrics", resolved.Name())
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "jobs outcome-metrics", resolved.CommandPath(),
		"outcome-metrics must be a direct child of jobs, not a child of jobs outcome")

	assert.NoError(t, resolved.ValidateArgs([]string{"loan-app-12345"}))
	assert.Error(t, resolved.ValidateArgs([]string{"loan-app-12345", "extra"}),
		"`jobs outcome-metrics <agenticJobId>` takes exactly one job id")
}

// TestRegistrationOutcomeStaysLeaf is the pin that D-28-02 was honoured: the
// released `jobs outcome <agenticJobId>` command is still a one-argument LEAF.
//
// The failure this catches is not hypothetical. Nesting a `metrics` child under
// `outcome` is the tidier-looking tree, and it would convert a released
// ExactArgs(1) command into a parent — after which a bare `jobs outcome <id>`
// begins resolving its first positional as a subcommand name. That is a
// behaviour change on a shipped, documented, scripted-against invocation, and
// no test elsewhere in this package would report it.
func TestRegistrationOutcomeStaysLeaf(t *testing.T) {
	outcome, leftover, err := Cmd.Find([]string{"outcome"})
	require.NoError(t, err)
	require.NotNil(t, outcome)
	require.Equal(t, "outcome", outcome.Name())
	assert.Empty(t, leftover)
	assert.Equal(t, "jobs outcome", outcome.CommandPath())

	assert.False(t, outcome.HasSubCommands(),
		"jobs outcome must stay a leaf — Phase 28 registered outcome-metrics as a sibling for exactly this reason")
	assert.Empty(t, outcome.Commands())

	require.NotNil(t, outcome.Args, "jobs outcome must keep an explicit Args validator")
	assert.NoError(t, outcome.Args(outcome, []string{"loan-app-12345"}),
		"one job id is the shipped invocation and must stay accepted")
	assert.Error(t, outcome.Args(outcome, []string{"loan-app-12345", "metrics"}),
		"a second positional must stay rejected — if it is accepted, outcome has become a parent")
}

// ---------------------------------------------------------------------------
// Phase 29 (JOBS-18) — the tree pins for `jobs roi-summary`, plus the pin that
// this phase did NOT convert the released `jobs roi` leaf into a parent.
//
// Named with the TestRegistration prefix deliberately, for the same reason the
// Phase 28 block above records: 29-VALIDATION.md prescribes the run selectors
// `go test ./cmd/jobs/ -run TestRegistrationROISummary` and
// `-run TestRegistrationROIStaysALeaf`, and a selector that matches nothing
// exits 0 while proving nothing. That is the exact trap 28-VALIDATION.md hit
// against this very file.
// ---------------------------------------------------------------------------

// TestRegistrationROISummary pins `jobs roi-summary` (JOBS-18, D-29-01) at its
// exact command path, and pins that its shorter namesake still resolves to
// itself.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a name-only
// assertion stays green on a command hung off the wrong parent — and hanging
// `summary` under the shipped `roi` leaf is precisely the tidier-looking tree
// D-29-01 refused. Name() cannot tell the two apart; CommandPath() can.
func TestRegistrationROISummary(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"roi-summary"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "roi-summary", resolved.Name())
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	// CommandPath starts at "jobs" here, not "revenium": main.go attaches Cmd
	// to the root command, and that attachment is outside this package.
	assert.Equal(t, "jobs roi-summary", resolved.CommandPath(),
		"roi-summary must be a direct child of jobs, not a child of jobs roi")

	// Two commands now share a name prefix. This is what proves the shorter one
	// still resolves to ITSELF rather than being shadowed by the longer, which
	// is the failure mode a flat hyphenated sibling introduces and which no
	// other test in this package would report.
	roi, roiLeftover, err := Cmd.Find([]string{"roi"})
	require.NoError(t, err)
	require.NotNil(t, roi)
	assert.Equal(t, "roi", roi.Name())
	assert.Empty(t, roiLeftover)
	assert.Equal(t, "jobs roi", roi.CommandPath(),
		"`jobs roi` must still resolve to the shipped leaf, not to its longer namesake")
}

// TestRegistrationROIStaysALeaf is the negative pin D-29-01 earns: the released
// `jobs roi <agenticJobId>` command is still an exact-one-argument LEAF.
//
// The failure this catches is not hypothetical, and it is the third time this
// repo has met the fork — `outcome-history`, `outcome-metrics`, now
// `roi-summary`. Nesting a `summary` child under `roi` is the tidier-looking
// tree, and it would convert a released ExactArgs(1) command into a parent,
// after which a bare `jobs roi <id>` begins resolving its first positional as a
// subcommand name. That is a behaviour change on a shipped, documented,
// scripted-against invocation. D-28-02 refused the move; this asserts it.
//
// Deliberately NOT asserted: the validator's error TEXT. roi.go composes
// ExactArgs(1) with the shared cmd.ValidResourceID checker, whose message is
// not this phase's contract.
func TestRegistrationROIStaysALeaf(t *testing.T) {
	roi, leftover, err := Cmd.Find([]string{"roi"})
	require.NoError(t, err)
	require.NotNil(t, roi)
	require.Equal(t, "roi", roi.Name())
	assert.Empty(t, leftover)
	assert.Equal(t, "jobs roi", roi.CommandPath())

	require.NotNil(t, roi.Args, "jobs roi must keep an explicit Args validator (CF-17)")
	assert.Error(t, roi.Args(roi, []string{}),
		"bare `jobs roi` must stay rejected — the job id is required")
	assert.NoError(t, roi.Args(roi, []string{"loan-app-12345"}),
		"one job id is the shipped invocation and must stay accepted")

	assert.False(t, roi.HasSubCommands(),
		"jobs roi must stay a leaf — Phase 29 registered roi-summary as a flat sibling for exactly this reason (D-29-01)")
	assert.Empty(t, roi.Commands())
}
