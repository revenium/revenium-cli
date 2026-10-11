package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaStillPrintsInQuietMode(t *testing.T) {
	oldOutput := Output
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	t.Cleanup(func() {
		Output = oldOutput
		quiet = false
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})

	rootCmd.SetArgs([]string{"schema", "--quiet"})
	require.NoError(t, rootCmd.Execute())

	assert.Contains(t, out.String(), `"commands"`)
}
