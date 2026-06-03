package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestJSONFlagRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, flag, "--json persistent flag should be registered")
	assert.Equal(t, "false", flag.DefValue)
}

func TestQuietFlagRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("quiet")
	require.NotNil(t, flag, "--quiet persistent flag should be registered")
	assert.Equal(t, "false", flag.DefValue)
}

func TestQuietShortFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().ShorthandLookup("q")
	require.NotNil(t, flag, "-q shorthand should be registered")
	assert.Equal(t, "quiet", flag.Name)
}

func TestOutputFormatterInitialized(t *testing.T) {
	// The Output var should be accessible (non-nil after PersistentPreRunE runs).
	// For config/version commands, it should still be initialized.
	// We test that the var exists and is of the right type by checking it's declared.
	// After running the version command (which skips config loading), Output should be set.
	oldOutput := Output
	defer func() { Output = oldOutput }()

	Output = nil
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.NotNil(t, Output, "Output formatter should be initialized even for version command")
}

func TestVerboseFlagStillWorks(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, flag, "--verbose flag should still be registered")
	assert.Equal(t, "v", flag.Shorthand)
}

func TestRegisterCommand(t *testing.T) {
	// Verify RegisterCommand adds a command with the correct group ID
	testCmd := &cobra.Command{Use: "test-resource", Short: "Test resource"}
	RegisterCommand(testCmd, "resources")

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "test-resource" {
			found = true
			assert.Equal(t, "resources", c.GroupID, "registered command should have the specified group ID")
			break
		}
	}
	assert.True(t, found, "registered command should be found on rootCmd")

	// Clean up: remove the test command
	rootCmd.RemoveCommand(testCmd)
}

func TestYesFlagRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("yes")
	require.NotNil(t, flag, "--yes persistent flag should be registered")
	assert.Equal(t, "false", flag.DefValue)
	assert.Equal(t, "y", flag.Shorthand, "--yes should have -y shorthand")
}

func TestOverrideFlagsRegistered(t *testing.T) {
	for _, name := range []string{"api-key", "api-url", "team-id", "tenant-id", "owner-id"} {
		flag := rootCmd.PersistentFlags().Lookup(name)
		require.NotNil(t, flag, "--%s persistent override flag should be registered", name)
		assert.Equal(t, "", flag.DefValue, "--%s should default to empty string", name)
	}
}

// TestRootHelpDocumentsOverrideFlags asserts that the root command Long help text
// documents the five global override flags section, the four-level precedence
// chain, and the two previously-undocumented env vars — satisfying CFGO-07 and D-05.
// Strings are asserted against the ACTUAL Long content of rootCmd (cmd/root.go).
func TestRootHelpDocumentsOverrideFlags(t *testing.T) {
	long := rootCmd.Long

	// The Global Override Flags section must be present.
	assert.Contains(t, long, "Global Override Flags", "Long help must contain a Global Override Flags section")

	// All five flag names must appear in the Long help.
	for _, flag := range []string{"--api-key", "--api-url", "--team-id", "--tenant-id", "--owner-id"} {
		assert.Contains(t, long, flag, "Long help must document override flag %s", flag)
	}

	// The four-level precedence statement must appear verbatim.
	assert.Contains(t, long, "flag > env var > config file > default",
		"Long help must state the four-level precedence chain")

	// The previously-undocumented tenant-id and owner-id env vars must be present.
	assert.Contains(t, long, "REVENIUM_TENANT_ID", "Long help must document REVENIUM_TENANT_ID env var")
	assert.Contains(t, long, "REVENIUM_OWNER_ID", "Long help must document REVENIUM_OWNER_ID env var")
}

func TestJSONModeFunction(t *testing.T) {
	// JSONMode should return the current state of the jsonMode var
	oldVal := jsonMode
	defer func() { jsonMode = oldVal }()

	jsonMode = false
	assert.False(t, JSONMode())

	jsonMode = true
	assert.True(t, JSONMode())
}
