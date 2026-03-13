// Package output provides formatted output rendering for the Revenium CLI.
// It handles styled table rendering, JSON output, TTY detection, and color
// profile management for pipe-safe output.
package output

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

// Formatter handles output rendering based on the active mode.
type Formatter struct {
	writer    io.Writer // os.Stdout, colorprofile.Writer, or io.Discard
	errWriter io.Writer // os.Stderr (always, even in quiet mode)
	jsonMode  bool
	quiet     bool
	isTTY bool
}

// New creates a Formatter, detecting TTY and terminal width.
// The writer is wrapped with colorprofile.NewWriter to automatically
// strip or downsample ANSI escape codes based on the detected color
// profile (respects NO_COLOR, TERM=dumb, and non-TTY outputs).
func New(jsonMode, quiet bool) *Formatter {
	f := &Formatter{
		writer:    os.Stdout,
		errWriter: os.Stderr,
		jsonMode:  jsonMode,
		quiet:     quiet,
	}

	// Detect TTY
	f.isTTY = term.IsTerminal(os.Stdout.Fd())

	// Quiet mode: suppress non-error output (but not --json)
	if quiet && !jsonMode {
		f.writer = io.Discard
	} else {
		// Wrap writer with colorprofile for automatic ANSI stripping/downsampling
		f.writer = colorprofile.NewWriter(os.Stdout, os.Environ())
	}

	return f
}

// NewWithWriter creates a Formatter with explicit writers for testing.
// It skips TTY detection and colorprofile wrapping, allowing tests to
// use bytes.Buffer to capture output.
func NewWithWriter(w io.Writer, errW io.Writer, jsonMode, quiet bool) *Formatter {
	f := &Formatter{
		writer:    w,
		errWriter: errW,
		jsonMode:  jsonMode,
		quiet:     quiet,
	}

	// Quiet mode: suppress non-error output (but not --json)
	if quiet && !jsonMode {
		f.writer = io.Discard
	}

	return f
}

// IsJSON returns true if JSON output mode is active.
func (f *Formatter) IsJSON() bool {
	return f.jsonMode
}

// IsQuiet returns true if quiet mode is active.
func (f *Formatter) IsQuiet() bool {
	return f.quiet
}
