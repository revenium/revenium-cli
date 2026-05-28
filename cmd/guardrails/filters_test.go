package guardrails

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseFilterTriple covers the colon-triple parse used by --filter.
//
// Per PLAN D-05 (locked): split on the FIRST TWO colons only so the value
// segment may itself contain colons (e.g. "AGENT:IS:my:weird:value" yields
// value="my:weird:value"). Error messages MUST include the raw offending input
// so users can locate the bad flag value in long invocations.
func TestParseFilterTriple(t *testing.T) {
	t.Run("simple triple", func(t *testing.T) {
		got, err := parseFilterTriple("MODEL:IS:gpt-4")
		require.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"dimension": "MODEL",
			"operator":  "IS",
			"value":     "gpt-4",
		}, got)
	})

	t.Run("value with embedded colons preserved", func(t *testing.T) {
		// PLAN D-05 locked: split on FIRST TWO colons only.
		got, err := parseFilterTriple("AGENT:IS:my:weird:value")
		require.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"dimension": "AGENT",
			"operator":  "IS",
			"value":     "my:weird:value",
		}, got)
	})

	t.Run("fewer than 3 parts errors with raw input in message", func(t *testing.T) {
		_, err := parseFilterTriple("MODEL:IS")
		require.Error(t, err)
		// Error message must name the offending input so users can locate it.
		assert.Contains(t, err.Error(), "MODEL:IS")
	})

	t.Run("empty input errors", func(t *testing.T) {
		_, err := parseFilterTriple("")
		require.Error(t, err)
	})

	t.Run("empty dimension errors", func(t *testing.T) {
		_, err := parseFilterTriple(":IS:gpt-4")
		require.Error(t, err)
	})

	t.Run("empty operator errors", func(t *testing.T) {
		_, err := parseFilterTriple("MODEL::gpt-4")
		require.Error(t, err)
	})

	t.Run("empty value errors", func(t *testing.T) {
		_, err := parseFilterTriple("MODEL:IS:")
		require.Error(t, err)
	})
}

// TestParseFiltersJSON covers the --filters-json escape hatch.
func TestParseFiltersJSON(t *testing.T) {
	t.Run("valid JSON array round-trips", func(t *testing.T) {
		got, err := parseFiltersJSON(`[{"dimension":"MODEL","operator":"IS","value":"gpt-4"}]`)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "MODEL", got[0]["dimension"])
		assert.Equal(t, "IS", got[0]["operator"])
		assert.Equal(t, "gpt-4", got[0]["value"])
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		_, err := parseFiltersJSON("not json")
		require.Error(t, err)
	})

	t.Run("JSON object (not array) returns error", func(t *testing.T) {
		_, err := parseFiltersJSON(`{"dimension":"MODEL"}`)
		require.Error(t, err)
	})
}

// makeFilterCmd builds a throwaway cobra.Command wired with the same flags
// as the real create/update commands so resolveFilters can be exercised with
// realistic Flags().Changed() state.
func makeFilterCmd(args []string) (*cobra.Command, *[]string, *string) {
	var filterFlags []string
	var filtersJSON string
	c := &cobra.Command{Use: "x", RunE: func(c *cobra.Command, args []string) error { return nil }}
	c.Flags().StringArrayVar(&filterFlags, "filter", nil, "")
	c.Flags().StringVar(&filtersJSON, "filters-json", "", "")
	c.SetArgs(args)
	_ = c.Execute()
	return c, &filterFlags, &filtersJSON
}

// TestResolveFilters covers the orchestration helper used by both create and
// update RunE blocks. Per PLAN D-01 (locked), passing both --filter AND
// --filters-json is a mutual-exclusion error that fires BEFORE any HTTP call.
func TestResolveFilters(t *testing.T) {
	t.Run("both flags set returns mutual-exclusion error naming both", func(t *testing.T) {
		c, ff, fj := makeFilterCmd([]string{
			"--filter", "MODEL:IS:gpt-4",
			"--filters-json", `[{"dimension":"X","operator":"IS","value":"y"}]`,
		})
		_, _, err := resolveFilters(c, *ff, *fj)
		require.Error(t, err)
		// PLAN D-01 locked: both flag names must appear in the error.
		assert.Contains(t, err.Error(), "--filter")
		assert.Contains(t, err.Error(), "--filters-json")
	})

	t.Run("neither flag set returns (nil, false, nil)", func(t *testing.T) {
		c, ff, fj := makeFilterCmd([]string{})
		filters, changed, err := resolveFilters(c, *ff, *fj)
		require.NoError(t, err)
		assert.False(t, changed, "changed must be false when neither flag was passed")
		assert.Nil(t, filters)
	})

	t.Run("only --filter set returns parsed slice + changed=true", func(t *testing.T) {
		c, ff, fj := makeFilterCmd([]string{
			"--filter", "MODEL:IS:gpt-4",
			"--filter", "PROVIDER:IS:openai",
		})
		filters, changed, err := resolveFilters(c, *ff, *fj)
		require.NoError(t, err)
		assert.True(t, changed)
		require.Len(t, filters, 2)
		assert.Equal(t, "MODEL", filters[0]["dimension"])
		assert.Equal(t, "PROVIDER", filters[1]["dimension"])
	})

	t.Run("only --filters-json set returns parsed slice + changed=true", func(t *testing.T) {
		c, ff, fj := makeFilterCmd([]string{
			"--filters-json", `[{"dimension":"MODEL","operator":"IS","value":"gpt-4"}]`,
		})
		filters, changed, err := resolveFilters(c, *ff, *fj)
		require.NoError(t, err)
		assert.True(t, changed)
		require.Len(t, filters, 1)
		assert.Equal(t, "gpt-4", filters[0]["value"])
	})

	t.Run("malformed --filter value surfaces the parser error", func(t *testing.T) {
		c, ff, fj := makeFilterCmd([]string{
			"--filter", "MODEL:IS", // missing value segment
		})
		_, _, err := resolveFilters(c, *ff, *fj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MODEL:IS")
	})
}
