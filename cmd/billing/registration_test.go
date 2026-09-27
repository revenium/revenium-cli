package billing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Phase 30 — the command-tree pins for the new billing surfaces (D-30-04).
//
// This file closes the gap every other cmd/billing test leaves open: they all
// construct their command directly (newVcsPrsCmd(), newDailyCmd(), …), so the
// package initializer that wires the tree is never exercised. A mis-parented or
// unregistered command would therefore ship with a fully green suite and
// surface only as `unknown command` in an operator's terminal.
//
// These tests walk the package-level Cmd — the same object main.go registers —
// so a registration regression turns cmd/billing red instead.
//
// Named with the TestRegistration prefix deliberately: 30-VALIDATION.md
// prescribes the run selector `go test ./cmd/billing/ -run TestRegistration`,
// and no shipped test in this package carries that prefix. A selector that
// matches nothing exits 0 while proving nothing, which is the one outcome a
// registration pin must never produce.
// ---------------------------------------------------------------------------

// TestRegistrationSeats pins `billing seats` (BILL-04, D-30-04) at its exact
// command path.
//
// CommandPath() is the load-bearing half. Cmd.Find returns the DEEPEST partial
// match it could reach and reports no error of its own, so a name-only
// assertion stays green on a command hung off the wrong parent.
func TestRegistrationSeats(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"seats"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "seats", resolved.Name())
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	// CommandPath starts at "billing" here, not "revenium": main.go attaches
	// Cmd to the root command, and that attachment is outside this package.
	assert.Equal(t, "billing seats", resolved.CommandPath(),
		"seats must be a direct child of billing, not a child of some other verb")
}

// TestRegistrationVcsPrsStaysALeaf is the negative pin D-30-04 earns: the
// released `billing vcs-prs` command is still a LEAF.
//
// The failure this catches is not hypothetical, and it is the third time this
// repo has met the fork — D-28-02 refused nesting `metrics` under `jobs
// outcome`, D-29-01 refused nesting `summary` under `jobs roi`, and nesting a
// `health` or `by-org-unit` child under `vcs-prs` is the same tidier-looking
// tree. Converting a released leaf into a parent changes how a bare
// `billing vcs-prs` invocation resolves its arguments — a behaviour change on a
// shipped, documented, scripted-against command that no other test here would
// report.
func TestRegistrationVcsPrsStaysALeaf(t *testing.T) {
	vcsPrs, leftover, err := Cmd.Find([]string{"vcs-prs"})
	require.NoError(t, err)
	require.NotNil(t, vcsPrs)
	require.Equal(t, "vcs-prs", vcsPrs.Name())
	assert.Empty(t, leftover)
	assert.Equal(t, "billing vcs-prs", vcsPrs.CommandPath())

	assert.False(t, vcsPrs.HasSubCommands(),
		"billing vcs-prs must stay a leaf — Phase 30 registers its new verbs as flat siblings for exactly this reason (D-30-04)")
	assert.Empty(t, vcsPrs.Commands())
}

// TestRegistrationVcsPrHealth pins `billing vcs-pr-health` (BILL-05, D-30-04)
// at its exact command path.
//
// CommandPath() is the load-bearing half here for the same reason it is on
// seats: Cmd.Find returns the DEEPEST partial match it could reach and reports
// no error of its own, so a name-only assertion stays green on a command hung
// off the wrong parent.
func TestRegistrationVcsPrHealth(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"vcs-pr-health"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "vcs-pr-health", resolved.Name())
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	// CommandPath starts at "billing" here, not "revenium": main.go attaches
	// Cmd to the root command, and that attachment is outside this package.
	// Do not "fix" the expected string to include the binary name.
	assert.Equal(t, "billing vcs-pr-health", resolved.CommandPath(),
		"vcs-pr-health must be a direct child of billing, not a child of some other verb")
}

// TestRegistrationVcsPrsByOrgUnit pins `billing vcs-prs-by-org-unit` (BILL-06,
// D-30-04) at its exact command path.
//
// This is the command of the three that would most plausibly have been hung
// under the shipped `vcs-prs` leaf — `billing vcs-prs by-org-unit` is the
// tidier-looking tree, and it is the same fork D-28-02 and D-29-01 refused.
// D-30-04 chose the flat hyphenated sibling instead, and this CommandPath()
// equality is what makes a silent re-parenting a red test: Name() would read
// "vcs-prs-by-org-unit" either way, so a name-only assertion could not tell the
// two trees apart.
//
// CommandPath starts at "billing" here, not "revenium", for the reason recorded
// above: main.go attaches Cmd to the root command outside this package.
func TestRegistrationVcsPrsByOrgUnit(t *testing.T) {
	resolved, leftover, err := Cmd.Find([]string{"vcs-prs-by-org-unit"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "vcs-prs-by-org-unit", resolved.Name())
	assert.Empty(t, leftover, "the whole path must be consumed by the command tree")
	assert.Equal(t, "billing vcs-prs-by-org-unit", resolved.CommandPath(),
		"vcs-prs-by-org-unit must hang off billing directly, not under vcs-prs (D-30-04)")
}
