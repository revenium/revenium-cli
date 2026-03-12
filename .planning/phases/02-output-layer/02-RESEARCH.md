# Phase 2: Output Layer - Research

**Researched:** 2026-03-12
**Domain:** Terminal output rendering (tables, JSON, TTY detection) with Lip Gloss v2
**Confidence:** HIGH

## Summary

Phase 2 builds the output infrastructure that every resource command (Phase 3+) will consume. The core deliverable is an `internal/output` package providing styled table rendering via Lip Gloss v2's built-in table package, JSON passthrough via `encoding/json`, and intelligent TTY/color detection via the `colorprofile` package (already an indirect dependency). The output layer must handle four output modes: styled tables (default), JSON (`--json`), quiet (`--quiet`), and verbose (`--verbose`, already exists).

Lip Gloss v2 is already imported at `charm.land/lipgloss/v2` v2.0.2 and used in `internal/errors/errors.go` with `RoundedBorder()`. The table sub-package at `charm.land/lipgloss/v2/table` (import path uses the v2 module) provides a fluent API with `StyleFunc` for per-cell styling, `Border()`, `Width()`, and `Wrap()`. The `colorprofile` package (v0.4.2, already an indirect dependency) handles TTY detection, `NO_COLOR`, `TERM=dumb`, and automatic ANSI color downsampling -- no need for `mattn/go-isatty` or manual TTY checks.

**Primary recommendation:** Create `internal/output/` with three files: `table.go` (styled table rendering), `json.go` (JSON output + JSON error output), and `styles.go` (shared Lip Gloss styles matching Phase 1 error box aesthetic). Add `--json` and `--quiet` as persistent flags on the root command. Use `colorprofile.Detect()` for TTY/color detection and `lipgloss.Println()` for automatic color downsampling.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions
- Rounded borders (Lip Gloss `lipgloss.RoundedBorder()`) -- matches error box style from Phase 1
- Bold/colored headers, colored status values (green=active, red=inactive, yellow=pending), plain data cells
- Long values truncated at ~40 chars with ellipsis (...)
- Single resource `get` commands use same table format as `list` (single-row table, not key-value pairs)
- `--json` passes through raw API response -- no wrapping, no transformation
- Pretty-printed (indented) by default
- List commands output a JSON array `[{...}, {...}]` -- simple, works with `jq '.[]'`
- When `--json` is set and an error occurs, errors are also JSON: `{"error": "Invalid API key", "status": 401}` on stderr
- Exit codes remain non-zero on errors even in JSON mode

### Claude's Discretion
- Output package internal API (function signatures, how resource commands register columns)
- TTY detection implementation (lipgloss.HasDarkBackground, isatty, etc.)
- NO_COLOR and TERM=dumb handling
- How --quiet interacts with --json (--quiet suppresses styled output, --json still outputs)
- How --verbose integrates with the output layer (already exists on root command from Phase 1)
- Table column width calculation and terminal width detection

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FNDN-08 | Styled table output using Lip Gloss v2 as default display format | Lip Gloss v2 table package API fully documented; StyleFunc pattern enables per-cell styling; RoundedBorder matches Phase 1 |
| FNDN-09 | `--json` flag on all output commands for machine-readable output | Persistent flag on root command; `encoding/json.MarshalIndent` for pretty-print; raw API passthrough pattern documented |
| FNDN-10 | TTY detection -- disable colors/styling when output is piped, respect `NO_COLOR` env var | `colorprofile.Detect()` handles TTY, NO_COLOR, TERM=dumb automatically; `lipgloss.Println()` auto-downsamples |
| FNDN-16 | `--quiet` / `-q` flag to suppress non-error output | Persistent flag on root; output functions check quiet flag and skip rendering; errors still go to stderr |
| FNDN-17 | `--verbose` / `-v` flag to show HTTP request/response details for debugging | Already exists on root command; output layer needs to respect it but not reimplement; verbose logging is in api.Client |

</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/lipgloss/v2` | v2.0.2 | Styled output and table rendering | Already imported; provides `table` sub-package with fluent API, `RoundedBorder()`, per-cell `StyleFunc` |
| `charm.land/lipgloss/v2/table` | v2.0.2 | Table rendering | Built-in to Lip Gloss v2; `New().Headers().Rows().StyleFunc().Border()` pattern |
| `github.com/charmbracelet/colorprofile` | v0.4.2 | TTY/color profile detection | Already an indirect dependency; handles `NO_COLOR`, `TERM=dumb`, non-TTY detection |
| `github.com/charmbracelet/x/term` | v0.2.2 | Terminal size detection | Already an indirect dependency; `term.GetSize()` for terminal width |
| `encoding/json` (stdlib) | stdlib | JSON serialization | `MarshalIndent` for pretty-printed `--json` output |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os` (stdlib) | stdlib | Stdout/Stderr, Fd() for TTY check | TTY detection via `os.Stdout.Fd()` |
| `io` (stdlib) | stdlib | `io.Discard` for quiet mode | Redirect output to discard when `--quiet` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `colorprofile.Detect()` | `mattn/go-isatty` | colorprofile is already a dependency and handles more (NO_COLOR, color downsampling) |
| `term.GetSize()` | `golang.org/x/term` | `charmbracelet/x/term` is already in the dependency tree via Lip Gloss |
| Lip Gloss v2 table | `olekukonko/tablewriter` | Lip Gloss table is built-in, matches styling ecosystem, avoids extra dependency |

**No new dependencies needed.** Everything required is already in the dependency tree via Lip Gloss v2.

## Architecture Patterns

### Recommended Project Structure
```
internal/output/
├── table.go      # RenderTable() - styled table rendering with Lip Gloss
├── json.go       # RenderJSON() - pretty-printed JSON, RenderJSONError() - JSON errors to stderr
├── styles.go     # Shared style definitions (header, status colors, border)
└── output.go     # Formatter type, TTY detection, mode resolution (table/json/quiet)
```

### Pattern 1: Formatter Type with Mode Resolution

**What:** A `Formatter` struct that resolves the output mode (table, JSON, quiet) and provides rendering methods. Created once per command execution in `PersistentPreRunE`.

**When to use:** Every command that produces output.

**Example:**
```go
// internal/output/output.go
package output

import (
    "io"
    "os"

    "github.com/charmbracelet/colorprofile"
    "github.com/charmbracelet/x/term"
)

// Formatter handles output rendering based on the active mode.
type Formatter struct {
    writer    io.Writer       // os.Stdout or io.Discard
    errWriter io.Writer       // os.Stderr (always, even in quiet mode)
    jsonMode  bool
    quiet     bool
    width     int             // terminal width, 0 if not a TTY
    isTTY     bool
}

// New creates a Formatter, detecting TTY and terminal width.
func New(jsonMode, quiet bool) *Formatter {
    f := &Formatter{
        writer:    os.Stdout,
        errWriter: os.Stderr,
        jsonMode:  jsonMode,
        quiet:     quiet,
    }

    // Detect TTY
    if file, ok := os.Stdout.(interface{ Fd() uintptr }); ok {
        f.isTTY = term.IsTerminal(file.Fd())
    }

    // Get terminal width
    if f.isTTY {
        if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil {
            f.width = w
        }
    }

    // Quiet mode: suppress non-error output (but not --json)
    if quiet && !jsonMode {
        f.writer = io.Discard
    }

    return f
}
```

### Pattern 2: StyleFunc for Per-Cell Table Styling

**What:** Lip Gloss v2 table's `StyleFunc(func(row, col int) lipgloss.Style)` enables different styles for headers, data cells, and status columns. Use `table.HeaderRow` constant (-1) to detect header row.

**When to use:** All table rendering.

**Example:**
```go
// internal/output/table.go
package output

import (
    "strings"

    "charm.land/lipgloss/v2"
    "charm.land/lipgloss/v2/table"
)

// TableDef defines columns for a resource table.
type TableDef struct {
    Headers      []string
    StatusColumn int    // index of the status column, -1 if none
}

// RenderTable renders data as a styled table.
func (f *Formatter) RenderTable(def TableDef, rows [][]string) error {
    if f.quiet && !f.jsonMode {
        return nil
    }

    t := table.New().
        Border(lipgloss.RoundedBorder()).
        BorderStyle(borderStyle).
        Headers(def.Headers...).
        Rows(rows...).
        StyleFunc(func(row, col int) lipgloss.Style {
            if row == table.HeaderRow {
                return headerStyle
            }
            if col == def.StatusColumn && row >= 0 {
                return statusStyle(rows[row][col])
            }
            return cellStyle
        })

    // Set width if we know terminal width
    if f.width > 0 {
        t.Width(f.width)
    }

    lipgloss.Println(t)
    return nil
}
```

### Pattern 3: Status Color Mapping

**What:** Map status string values to colors for visual scanning of active/inactive/pending resources.

**Example:**
```go
// internal/output/styles.go
package output

import "charm.land/lipgloss/v2"

var (
    // Border style matches Phase 1 error boxes
    borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

    // Header: bold with accent color
    headerStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("99"))

    // Plain data cells
    cellStyle = lipgloss.NewStyle().Padding(0, 1)
)

// statusStyle returns a style colored by status value.
func statusStyle(status string) lipgloss.Style {
    base := lipgloss.NewStyle().Padding(0, 1)
    switch strings.ToLower(status) {
    case "active", "enabled":
        return base.Foreground(lipgloss.Color("42"))  // green
    case "inactive", "disabled", "deleted":
        return base.Foreground(lipgloss.Color("196")) // red
    case "pending", "draft":
        return base.Foreground(lipgloss.Color("214")) // yellow
    default:
        return base
    }
}
```

### Pattern 4: JSON Output with Raw Passthrough

**What:** `--json` outputs the raw API response, pretty-printed. The output layer receives the already-unmarshaled data and re-marshals it with indentation. For errors in JSON mode, emit `{"error": "...", "status": N}` to stderr.

**Example:**
```go
// internal/output/json.go
package output

import (
    "encoding/json"
    "fmt"
)

// RenderJSON writes data as pretty-printed JSON to stdout.
func (f *Formatter) RenderJSON(data interface{}) error {
    enc := json.NewEncoder(f.writer)
    enc.SetIndent("", "  ")
    return enc.Encode(data)
}

// RenderJSONError writes a JSON error to stderr.
func RenderJSONError(msg string, statusCode int) {
    errObj := map[string]interface{}{
        "error":  msg,
        "status": statusCode,
    }
    enc := json.NewEncoder(os.Stderr)
    enc.SetIndent("", "  ")
    enc.Encode(errObj)
}
```

### Pattern 5: Value Truncation

**What:** Truncate long cell values at ~40 characters with an ellipsis to keep tables compact.

**Example:**
```go
// internal/output/table.go

// Truncate truncates s to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
    if len([]rune(s)) <= maxLen {
        return s
    }
    return string([]rune(s)[:maxLen-1]) + "..."
}
```

**Note:** Use `[]rune` for correct Unicode handling, not `len(s)` which counts bytes.

### Pattern 6: Global Flag Registration

**What:** Add `--json` and `--quiet` as persistent flags on the root command, alongside the existing `--verbose`.

**Example:**
```go
// cmd/root.go additions
var (
    verbose  bool  // already exists
    jsonMode bool
    quiet    bool
)

func init() {
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
    rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output as JSON")
    rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
}
```

### Anti-Patterns to Avoid

- **Wrapping API responses in --json mode:** The user explicitly decided `--json` passes raw API response. Do NOT add metadata envelopes like `{"data": [...], "count": N}`.
- **Styled output leaking into JSON:** When `--json` is active, zero styled output should reach stdout. Use `lipgloss.Println()` only in table mode.
- **Forgetting stderr for JSON errors:** In `--json` mode, errors MUST go to stderr as JSON, not stdout. Scripts parsing JSON stdout will break otherwise.
- **Using `fmt.Println` instead of `lipgloss.Println`:** The `lipgloss.Println()` function auto-downsamples colors via `colorprofile`. Using `fmt.Println` bypasses this and may emit raw ANSI codes to non-TTY outputs.
- **Manual TTY detection with `mattn/go-isatty`:** The `colorprofile.Detect()` function already handles TTY detection, `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`, and `TERM=dumb`. Adding a separate TTY library is redundant.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TTY detection | Custom `isatty` checks | `colorprofile.Detect(os.Stdout, os.Environ())` | Handles NO_COLOR, CLICOLOR, CLICOLOR_FORCE, TERM=dumb, non-TTY output |
| Color downsampling | Manual ANSI stripping | `lipgloss.Println()` or `colorprofile.NewWriter()` | Automatically downgrades TrueColor to ANSI256/ANSI/ASCII based on terminal |
| Terminal width | Manual ioctl calls | `term.GetSize(os.Stdout.Fd())` | Cross-platform, already in dependency tree |
| Table rendering | Custom column alignment | `charm.land/lipgloss/v2/table` | Handles Unicode width, wrapping, border rendering |
| Unicode-aware truncation | `s[:40]` byte slicing | `[]rune(s)[:maxLen]` | Byte slicing breaks multi-byte characters |

**Key insight:** Lip Gloss v2 and its dependency tree (`colorprofile`, `x/term`) already solve every terminal-interaction problem this phase needs. Zero new dependencies required.

## Common Pitfalls

### Pitfall 1: JSON Output Contaminated by Styled Output
**What goes wrong:** When `--json` is active, styled error messages or verbose output accidentally reaches stdout alongside the JSON data.
**Why it happens:** `fmt.Println` or `lipgloss.Println` used without checking `--json` mode first.
**How to avoid:** All output goes through the `Formatter`. In JSON mode, the `Formatter.writer` writes only JSON. Errors always go to stderr. Verbose output always goes to stderr (already the case in `api/client.go`).
**Warning signs:** `revenium sources list --json | jq .` fails with parse errors.

### Pitfall 2: Broken Pipe Panics
**What goes wrong:** CLI panics when piped output is closed early (e.g., `revenium sources list | head -1`).
**Why it happens:** Writing to a closed pipe causes a `SIGPIPE` signal or write error that isn't handled.
**How to avoid:** Go handles `SIGPIPE` by default (exits with code 141). Ensure no panic recovery interferes. Test with `| head -1`.
**Warning signs:** Panic stack traces when piping to `head` or `grep`.

### Pitfall 3: Hardcoded Terminal Width
**What goes wrong:** Tables overflow narrow terminals or waste space on wide terminals.
**Why it happens:** Using a fixed width (e.g., 80) instead of detecting the actual terminal.
**How to avoid:** Use `term.GetSize()` and pass width to `table.Width()`. Fall back to 80 when not a TTY or when detection fails.
**Warning signs:** Tables with ugly wrapping or excessive whitespace.

### Pitfall 4: --quiet Suppressing JSON Output
**What goes wrong:** `--quiet --json` produces no output at all.
**Why it happens:** `--quiet` blanket-suppresses all stdout.
**How to avoid:** `--json` takes precedence over `--quiet`. If both are set, JSON output still flows to stdout. `--quiet` only suppresses styled table output.
**Warning signs:** `revenium sources list --json --quiet` returning empty.

### Pitfall 5: Color Codes in Non-TTY Output
**What goes wrong:** ANSI escape sequences appear in piped output or log files.
**Why it happens:** Using `style.Render()` + `fmt.Println()` instead of `lipgloss.Println()`.
**How to avoid:** Always use `lipgloss.Println()` for styled output, which auto-detects TTY and downsamples. For non-TTY, it strips all ANSI codes.
**Warning signs:** Raw `^[[38;2;...m` sequences in `> output.txt`.

### Pitfall 6: Ellipsis Truncation with Multi-byte Characters
**What goes wrong:** Truncation at byte position N breaks UTF-8 characters, producing garbled output.
**Why it happens:** Using `s[:40]` instead of rune-based truncation.
**How to avoid:** Use `[]rune(s)` for character-aware truncation. Use `lipgloss.Width()` for display-width-aware truncation (handles CJK double-width).
**Warning signs:** Garbled characters at the end of truncated values in API names with Unicode.

## Code Examples

### Complete Table Rendering (verified pattern from official docs)
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/lipgloss/v2/table
import (
    "charm.land/lipgloss/v2"
    "charm.land/lipgloss/v2/table"
)

t := table.New().
    Border(lipgloss.RoundedBorder()).
    BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
    Headers("ID", "NAME", "STATUS").
    Row("abc-123", "My Source", "active").
    Row("def-456", "Another Source", "inactive").
    StyleFunc(func(row, col int) lipgloss.Style {
        switch {
        case row == table.HeaderRow:
            return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Padding(0, 1)
        default:
            return lipgloss.NewStyle().Padding(0, 1)
        }
    }).
    Width(80)

lipgloss.Println(t)
```

### TTY and Color Profile Detection
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/colorprofile
import (
    "os"

    "github.com/charmbracelet/colorprofile"
    "github.com/charmbracelet/x/term"
)

// Detect if stdout is a TTY
isTTY := term.IsTerminal(os.Stdout.Fd())

// Detect color profile (handles NO_COLOR, TERM=dumb, etc.)
profile := colorprofile.Detect(os.Stdout, os.Environ())
// profile is one of: TrueColor, ANSI256, ANSI, ASCII, NoTTY

// For non-TTY or --json mode, skip styled rendering entirely
if profile == colorprofile.NoTTY || profile == colorprofile.ASCII {
    // plain text or JSON only
}
```

### JSON Error to Stderr
```go
// When --json is active and an error occurs
import (
    "encoding/json"
    "os"

    apierrors "github.com/revenium/revenium-cli/internal/errors"
)

func renderError(err error, jsonMode bool) {
    if jsonMode {
        if apiErr, ok := err.(*apierrors.APIError); ok {
            json.NewEncoder(os.Stderr).Encode(map[string]interface{}{
                "error":  apiErr.Message,
                "status": apiErr.StatusCode,
            })
        } else {
            json.NewEncoder(os.Stderr).Encode(map[string]interface{}{
                "error": err.Error(),
            })
        }
    } else {
        fmt.Fprintln(os.Stderr, apierrors.RenderError(err.Error()))
    }
}
```

### Integration with main.go Error Handling
```go
// main.go needs to be updated to check --json mode for error rendering
func main() {
    rootCmd := cmd.NewRootCmd()
    if err := rootCmd.Execute(); err != nil {
        if cmd.JSONMode {
            output.RenderJSONError(err)
        } else {
            fmt.Fprintln(os.Stderr, apierrors.RenderError(err.Error()))
        }
        os.Exit(1)
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `lipgloss.SetColorProfile()` (global) | `colorprofile.Detect()` per-writer | Lip Gloss v2 (Mar 2025) | No more global state; per-output-stream detection |
| `mattn/go-isatty` for TTY detection | `charmbracelet/x/term.IsTerminal()` | Charm ecosystem v2 | One fewer dependency; consistent ecosystem |
| Separate table library (tablewriter) | `lipgloss/v2/table` built-in | Lip Gloss v2 (Mar 2025) | Native styling integration, no extra dependency |
| `fmt.Println()` for styled output | `lipgloss.Println()` auto-downsamples | Lip Gloss v2 (Mar 2025) | Automatic color profile respect |
| Manual `NO_COLOR` env var checking | `colorprofile.Detect()` handles it | colorprofile package | Follows no-color.org spec automatically |

**Deprecated/outdated:**
- `lipgloss.SetColorProfile()`: Global state, removed in v2. Use `colorprofile.Detect()` instead.
- `github.com/charmbracelet/lipgloss` (v1 import): Project uses v2 at `charm.land/lipgloss/v2`.

## Open Questions

1. **API response shape for lists vs. single resources**
   - What we know: API returns JSON that gets unmarshaled into `result interface{}` in `client.Do()`
   - What's unclear: Whether list endpoints return bare arrays `[...]` or wrapped `{"items": [...]}` -- affects JSON passthrough
   - Recommendation: During Phase 3 (Sources), discover the actual shape and adjust. The output layer should handle both patterns.

2. **How to expose Formatter to commands**
   - What we know: `APIClient` is currently a package-level var in `cmd/root.go`, set in `PersistentPreRunE`
   - What's unclear: Whether to add a package-level `Formatter` var alongside `APIClient`, or pass it through command constructors
   - Recommendation: Use the same pattern as `APIClient` -- a package-level `Output` var in `cmd/root.go` initialized in `PersistentPreRunE`. This is consistent with Phase 1's approach and avoids refactoring the command registration pattern.

3. **Terminal width for non-TTY (e.g., CI environments)**
   - What we know: `term.GetSize()` fails for non-TTY
   - What's unclear: Whether to default to 80 or render without width constraint
   - Recommendation: Default to 80 columns when terminal width cannot be detected. This is the POSIX standard default.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify v1.11.1 |
| Config file | None needed (Go convention) |
| Quick run command | `go test ./internal/output/... -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FNDN-08 | Styled table output renders with correct borders and styles | unit | `go test ./internal/output/... -run TestRenderTable -count=1` | Wave 0 |
| FNDN-08 | Single-row table for get commands | unit | `go test ./internal/output/... -run TestSingleRowTable -count=1` | Wave 0 |
| FNDN-08 | Long values truncated at ~40 chars with ellipsis | unit | `go test ./internal/output/... -run TestTruncate -count=1` | Wave 0 |
| FNDN-09 | --json outputs pretty-printed JSON to stdout | unit | `go test ./internal/output/... -run TestRenderJSON -count=1` | Wave 0 |
| FNDN-09 | JSON errors to stderr with correct shape | unit | `go test ./internal/output/... -run TestRenderJSONError -count=1` | Wave 0 |
| FNDN-10 | NO_COLOR disables colors | unit | `go test ./internal/output/... -run TestNoColor -count=1` | Wave 0 |
| FNDN-10 | Non-TTY output strips ANSI | unit | `go test ./internal/output/... -run TestNonTTY -count=1` | Wave 0 |
| FNDN-16 | --quiet suppresses styled output | unit | `go test ./internal/output/... -run TestQuiet -count=1` | Wave 0 |
| FNDN-16 | --quiet + --json still outputs JSON | unit | `go test ./internal/output/... -run TestQuietWithJSON -count=1` | Wave 0 |
| FNDN-17 | --verbose flag exists on root command | unit | `go test ./cmd/... -run TestVerboseFlag -count=1` | Exists (root_test.go) |

### Sampling Rate
- **Per task commit:** `go test ./internal/output/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/output/table_test.go` -- covers FNDN-08 (table rendering, truncation, single-row)
- [ ] `internal/output/json_test.go` -- covers FNDN-09 (JSON output, JSON errors)
- [ ] `internal/output/output_test.go` -- covers FNDN-10, FNDN-16 (TTY detection, quiet mode, NO_COLOR)
- [ ] `cmd/root_test.go` -- needs new tests for `--json` and `--quiet` flag registration

*(Test infrastructure exists -- Go testing + testify. Only test files for the new `internal/output/` package are needed.)*

## Sources

### Primary (HIGH confidence)
- [Lip Gloss v2 table package API](https://pkg.go.dev/github.com/charmbracelet/lipgloss/v2/table) - Full table API: New(), Headers(), Rows(), StyleFunc(), Border(), Width(), Wrap()
- [Lip Gloss v2 package API](https://pkg.go.dev/github.com/charmbracelet/lipgloss/v2) - Core API: Println(), HasDarkBackground(), Color(), Border types, Writer
- [colorprofile package API](https://pkg.go.dev/github.com/charmbracelet/colorprofile) - Detect(), Profile type, Writer, NO_COLOR/TERM=dumb handling
- [charmbracelet/x/term package](https://pkg.go.dev/github.com/charmbracelet/x/term) - IsTerminal(), GetSize() for TTY and terminal width detection

### Secondary (MEDIUM confidence)
- [Lip Gloss v2: What's New (Discussion #506)](https://github.com/charmbracelet/lipgloss/discussions/506) - v2 migration guide, writer-based output, color profile changes

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in `go.mod` / indirect dependencies; API verified from pkg.go.dev
- Architecture: HIGH - Patterns derived from existing codebase conventions (Phase 1) and Lip Gloss v2 official examples
- Pitfalls: HIGH - TTY/pipe issues and JSON contamination are well-documented in CLI development
- API surface: MEDIUM - Lip Gloss v2 table import path needs verification (`charm.land/lipgloss/v2/table` vs `github.com/charmbracelet/lipgloss/v2/table`)

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable ecosystem, Lip Gloss v2 is released)
