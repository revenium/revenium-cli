package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/internal/build"
)

func TestVersionCommandOutput(t *testing.T) {
	cmd := newVersionCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "revenium")
	assert.Contains(t, output, build.Version)
}
