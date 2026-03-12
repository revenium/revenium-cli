---
phase: 02-output-layer
plan: 01
subsystem: output
tags: [lipgloss, table, json, tty, colorprofile, ansi]

# Dependency graph
requires:
  - phase: 01-project-scaffold-config
    provides: "Lip Gloss v2, Cobra root command, error rendering with RoundedBorder"
provides:
  - "Formatter type with TTY detection, colorprofile-wrapped writer, and mode resolution"
  - "Styled table rendering with rounded borders, bold headers, and status colors"
  - "JSON rendering with pretty-print and JSON error output to stderr"
  - "Unicode-safe Truncate function with ellipsis (U+2026)"
  - "Render convenience method dispatching to JSON or table based on mode"
affects: [03-source-crud, 04-api-product-crud, 05-metering-crud, 06-analytics-crud, 07-subscription-crud, 08-platform-crud]

# Tech tracking
tech-stack:
  added: [charm.land/lipgloss/v2/table, github.com/charmbracelet/colorprofile, github.com/charmbracelet/x/term]
  patterns: [Formatter-with-mode-resolution, colorprofile-writer-wrapping, StyleFunc-per-cell-styling, NewWithWriter-test-constructor]

key-files:
  created:
    - internal/output/output.go
    - internal/output/styles.go
    - internal/output/table.go
    - internal/output/json.go
    - internal/output/output_test.go
    - internal/output/table_test.go
    - internal/output/json_test.go
  modified: []

key-decisions:
  - "colorprofile.NewWriter wraps stdout for automatic ANSI stripping in New() constructor"
  - "NewWithWriter skips colorprofile wrapping for test isolation with bytes.Buffer"
  - "Non-TTY ANSI stripping tested via colorprofile.Writer with NoTTY profile"
  - "Default terminal width 80 when detection fails"

patterns-established:
  - "Formatter pattern: New(jsonMode, quiet) for production, NewWithWriter(w, errW, jsonMode, quiet) for tests"
  - "Render dispatch: f.Render(def, rows, data) routes to JSON or table based on mode"
  - "Status color mapping: statusStyle() returns green/red/yellow lipgloss.Style based on status string"
  - "Table definition: TableDef{Headers, StatusColumn} passed to RenderTable for per-resource customization"

requirements-completed: [FNDN-08, FNDN-09, FNDN-10, FNDN-16]

# Metrics
duration: 3min
completed: 2026-03-12
---

# Phase 2 Plan 01: Output Package Summary

**Formatter with TTY-aware styled tables, JSON rendering, colorprofile ANSI stripping, and Unicode truncation using Lip Gloss v2**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-12T05:17:22Z
- **Completed:** 2026-03-12T05:20:53Z
- **Tasks:** 2
- **Files created:** 7

## Accomplishments
- Formatter type with TTY detection, colorprofile-wrapped writer, quiet mode, and JSON mode resolution
- Styled table rendering with rounded borders, bold headers (color 99), and per-status color mapping (green/red/yellow)
- JSON rendering with pretty-print (2-space indent) and JSON error output to stderr with {"error", "status"} shape
- Unicode-safe Truncate function using rune-based slicing with single-character ellipsis (U+2026)
- 20 unit tests covering all behaviors including non-TTY ANSI stripping and quiet+JSON interaction

## Task Commits

Each task was committed atomically:

1. **Task 1: Create output package -- Formatter, styles, table rendering, truncation** - `c74b722` (feat)
2. **Task 2: JSON rendering and JSON error output** - `4f7ed40` (feat)

_Note: TDD tasks had tests written first (RED), then implementation (GREEN), committed together per task._

## Files Created/Modified
- `internal/output/output.go` - Formatter type with New() and NewWithWriter() constructors, TTY detection, colorprofile wrapping
- `internal/output/styles.go` - Shared Lip Gloss styles: borderStyle, headerStyle, cellStyle, statusStyle()
- `internal/output/table.go` - TableDef type, RenderTable method, Truncate function
- `internal/output/json.go` - RenderJSON, RenderJSONError, Render convenience method
- `internal/output/output_test.go` - Tests for Formatter creation, quiet mode, NO_COLOR, non-TTY
- `internal/output/table_test.go` - Tests for Truncate (short, long, Unicode, boundary), RenderTable, statusStyle
- `internal/output/json_test.go` - Tests for JSON pretty-print, arrays, objects, quiet+JSON, JSON errors, Render dispatch

## Decisions Made
- Used colorprofile.NewWriter to wrap stdout in New() for automatic ANSI stripping/downsampling -- this handles NO_COLOR, TERM=dumb, and non-TTY automatically at write time
- NewWithWriter skips colorprofile wrapping so tests can use bytes.Buffer directly without TTY dependency
- Tested non-TTY ANSI stripping by constructing a colorprofile.Writer with NoTTY profile explicitly
- Default terminal width set to 80 when term.GetSize fails (POSIX standard default)
- RenderJSONError is a method on Formatter (not standalone function) to use the errWriter field

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Output package complete and ready for Phase 3+ resource commands
- Resource commands will use `output.New(jsonMode, quiet)` in PersistentPreRunE and call `f.Render(def, rows, data)` to render output
- --json and --quiet persistent flags on root command still need to be added (likely Phase 3 when first resource command is built)

---
*Phase: 02-output-layer*
*Completed: 2026-03-12*
