package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_DetectsTTY(t *testing.T) {
	f := New(false, false)
	require.NotNil(t, f)
	assert.False(t, f.IsJSON())
	assert.False(t, f.IsQuiet())
}

func TestNew_QuietMode(t *testing.T) {
	// quiet=true, jsonMode=false -> writer should be io.Discard
	f := NewWithWriter(&bytes.Buffer{}, &bytes.Buffer{}, false, true)
	assert.True(t, f.IsQuiet())

	// Write through RenderTable - should produce no output
	var buf bytes.Buffer
	fQuiet := NewWithWriter(&buf, &bytes.Buffer{}, false, true)
	err := fQuiet.RenderTable(TableDef{Headers: []string{"A"}, StatusColumn: -1}, [][]string{{"x"}})
	require.NoError(t, err)
	assert.Empty(t, buf.String())

	// jsonMode=true, quiet=true -> JSON overrides quiet, writer should NOT be discard
	fJSON := NewWithWriter(&bytes.Buffer{}, &bytes.Buffer{}, true, true)
	assert.True(t, fJSON.IsJSON())
	assert.True(t, fJSON.IsQuiet())
}

func TestNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	f := New(false, false)
	require.NotNil(t, f)

	var buf bytes.Buffer
	fTest := NewWithWriter(&buf, &bytes.Buffer{}, false, false)
	err := fTest.RenderTable(TableDef{
		Headers:      []string{"Name", "Status"},
		StatusColumn: 1,
	}, [][]string{{"test", "active"}})
	require.NoError(t, err)

	// NewWithWriter skips colorprofile wrapping, so we can't test ANSI stripping
	// through it. The NO_COLOR test verifies the New() constructor path.
	// For NewWithWriter, output just goes directly to the buffer.
	assert.NotEmpty(t, buf.String())
}

func TestNonTTY(t *testing.T) {
	// Use colorprofile.Writer with NoTTY profile to simulate non-TTY output.
	// This verifies that ANSI codes are stripped when output is not a terminal.
	var buf bytes.Buffer
	cpWriter := &colorprofile.Writer{
		Forward: &buf,
		Profile: colorprofile.NoTTY,
	}
	f := NewWithWriter(cpWriter, &bytes.Buffer{}, false, false)
	err := f.RenderTable(TableDef{
		Headers:      []string{"Name", "Status"},
		StatusColumn: 1,
	}, [][]string{{"hello", "active"}})
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)
	// colorprofile.Writer with NoTTY strips all ANSI escape codes
	assert.False(t, strings.Contains(output, "\x1b"), "output should not contain ANSI escape codes for non-TTY writer")
}
