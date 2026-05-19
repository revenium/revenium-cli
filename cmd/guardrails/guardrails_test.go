package guardrails

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParentCommandStructure asserts the guardrails parent Cmd has exactly the
// three expected sub-parents (budget-rules, enforcement-rules, enforcement-events),
// proving the explicit initFn helpers in guardrails.go init() are mounted regardless
// of filename-asc init ordering (D-01). Regression catch: T-14-01-02.
func TestParentCommandStructure(t *testing.T) {
	require.Equal(t, "guardrails", Cmd.Use)

	var subUses []string
	for _, c := range Cmd.Commands() {
		subUses = append(subUses, c.Use)
	}

	assert.Contains(t, subUses, "budget-rules")
	assert.Contains(t, subUses, "enforcement-rules")
	assert.Contains(t, subUses, "enforcement-events")
	require.Len(t, Cmd.Commands(), 3, "guardrails must have exactly 3 sub-parents")
}
