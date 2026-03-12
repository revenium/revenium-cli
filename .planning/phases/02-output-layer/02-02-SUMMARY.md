---
phase: 02-output-layer
plan: 02
subsystem: cli
tags: [cobra, flags, json, output, formatter]

# Dependency graph
requires:
  - phase: 02-output-layer
    provides: output.Formatter with JSON/table rendering
provides:
  - Global --json and --quiet flags on root command
  - cmd.Output package-level Formatter accessible to all subcommands
  - cmd.JSONMode() for error rendering decisions in main.go
  - JSON error output to stderr when --json is active
affects: [03-sources-crud, 04-api-keys, 05-assets, 06-metering, 07-monetization, 08-analytics, 09-metrics, 10-policies, 11-polish]

# Tech tracking
tech-stack:
  added: []
  patterns: [PersistentPreRunE formatter initialization, JSONMode() accessor for cross-package flag state]

key-files:
  created: []
  modified: [cmd/root.go, cmd/root_test.go, main.go]

key-decisions:
  - "Output formatter initialized before config/version skip so all commands have access"
  - "JSONMode() exported function avoids main.go needing to import output package for flag check"
  - "errors.As used for idiomatic APIError type assertion in main.go"

patterns-established:
  - "cmd.Output: subcommands access the formatter via package-level var"
  - "cmd.JSONMode(): main.go checks JSON mode without direct flag access"

requirements-completed: [FNDN-09, FNDN-10, FNDN-16, FNDN-17]

# Metrics
duration: 2min
completed: 2026-03-12
---

# Phase 2 Plan 02: Root Command Integration Summary

**Global --json and --quiet flags wired into root command with Formatter init in PersistentPreRunE and JSON error rendering in main.go**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-12T05:23:27Z
- **Completed:** 2026-03-12T05:25:03Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added --json and --quiet/-q persistent flags to root command
- Output formatter initialized in PersistentPreRunE for all commands (including config/version)
- main.go renders errors as JSON to stderr when --json is active, preserving styled rendering otherwise
- All existing tests continue to pass with 6 new tests added

## Task Commits

Each task was committed atomically:

1. **Task 1: Add --json and --quiet flags (TDD RED)** - `c6e85f0` (test)
2. **Task 1: Add --json and --quiet flags (TDD GREEN)** - `7afca8c` (feat)
3. **Task 2: Update main.go for JSON error rendering** - `ac9cfbd` (feat)

## Files Created/Modified
- `cmd/root.go` - Added Output var, jsonMode/quiet vars, --json/--quiet flags, JSONMode() function, Formatter init in PersistentPreRunE
- `cmd/root_test.go` - Added 6 tests for new flags, Output initialization, and JSONMode()
- `main.go` - Added JSON error rendering path using errors.As and output.RenderJSONError

## Decisions Made
- Output formatter initialized before config/version skip check so all commands have access to it
- JSONMode() exported as a function rather than exposing the var directly, keeping jsonMode private
- Used errors.As for idiomatic Go error type assertion instead of direct type cast

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- cmd.Output is available for all subcommands to use for table/JSON rendering
- Phase 3 (Sources CRUD) can import cmd.Output and call Render()/RenderJSON() directly
- JSON error rendering works end-to-end through main.go

## Self-Check: PASSED

All 3 files found, all 3 commits verified.

---
*Phase: 02-output-layer*
*Completed: 2026-03-12*
