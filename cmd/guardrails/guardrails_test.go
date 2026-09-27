package guardrails

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParentCommandStructure asserts the guardrails parent Cmd has exactly the
// four expected sub-commands — three sub-parents (budget-rules,
// enforcement-rules, enforcement-events) plus one leaf
// (org-unit-group-preview) — proving the explicit initFn helpers and the leaf's
// AddCommand in guardrails.go init() are all mounted regardless of filename-asc
// init ordering (D-01). Regression catch: T-14-01-02.
//
// The exact count is the assertion that catches a command silently failing to
// register, so it is kept. It is taken over the DECLARED sub-commands only:
// cobra's ExecuteC injects `help` and `completion` into whichever command it
// treats as root, so a raw len(Cmd.Commands()) would become order-dependent the
// moment any test in this package executes Cmd itself.
func TestParentCommandStructure(t *testing.T) {
	require.Equal(t, "guardrails", Cmd.Use)

	var subUses []string
	for _, c := range Cmd.Commands() {
		if c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		subUses = append(subUses, c.Use)
	}

	assert.Contains(t, subUses, "budget-rules")
	assert.Contains(t, subUses, "enforcement-rules")
	assert.Contains(t, subUses, "enforcement-events")
	assert.Contains(t, subUses, "org-unit-group-preview")
	require.Len(t, subUses, 4,
		"guardrails must have exactly 4 declared sub-commands: 3 sub-parents plus the org-unit-group-preview leaf")
}
