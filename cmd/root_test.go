package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommandExists(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "revenium", rootCmd.Use)
}

func TestRootHasGroups(t *testing.T) {
	groups := rootCmd.Groups()
	assert.Len(t, groups, 3)

	groupIDs := make([]string, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID
	}
	assert.Contains(t, groupIDs, "resources")
	assert.Contains(t, groupIDs, "monitoring")
	assert.Contains(t, groupIDs, "config")
}

func TestRootHasExamples(t *testing.T) {
	assert.NotEmpty(t, rootCmd.Example)
}

func TestSilenceFlags(t *testing.T) {
	assert.True(t, rootCmd.SilenceErrors)
	assert.True(t, rootCmd.SilenceUsage)
}

func TestExecuteWithoutConfig(t *testing.T) {
	// Execute with no args just shows help and returns no error
	err := rootCmd.Execute()
	assert.NoError(t, err)
}
