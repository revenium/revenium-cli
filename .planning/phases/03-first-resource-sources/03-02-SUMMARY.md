---
phase: 03-first-resource-sources
plan: 02
subsystem: api
tags: [cobra, crud, sources, create, update, delete, partial-update, confirm-delete]

# Dependency graph
requires:
  - phase: 03-first-resource-sources
    plan: 01
    provides: "ConfirmDelete helper, sources list/get, tableDef, toRows, str, renderSource helpers, --yes flag"
  - phase: 02-output-layer
    provides: "Formatter with Render/RenderJSON, TableDef, IsJSON, IsQuiet"
  - phase: 01-project-scaffold-config
    provides: "API Client with Do(), rootCmd with persistent flags"
provides:
  - "sources create command (POST /v2/api/sources)"
  - "sources update command with partial update semantics (PUT /v2/api/sources/{id})"
  - "sources delete command with ConfirmDelete integration (DELETE /v2/api/sources/{id})"
  - "RegisterCommand helper in cmd package for circular-import-safe command registration"
  - "Full CRUD vertical slice for Sources proving pattern for Phases 4-9"
affects: [04-assets, 05-products, 06-contracts, 07-customers, 08-subscriptions]

# Tech tracking
tech-stack:
  added: []
  patterns: [register-command-from-main, partial-update-via-flags-changed, standalone-flag-in-tests]

key-files:
  created:
    - cmd/sources/create.go
    - cmd/sources/create_test.go
    - cmd/sources/update.go
    - cmd/sources/update_test.go
    - cmd/sources/delete.go
    - cmd/sources/delete_test.go
  modified:
    - cmd/sources/sources.go
    - cmd/root.go
    - cmd/root_test.go
    - main.go

key-decisions:
  - "RegisterCommand pattern avoids circular imports between cmd and cmd/sources"
  - "Resource commands registered in main.go init() rather than cmd/root.go init()"
  - "Delete test registers --yes flag locally since inherited persistent flag unavailable in standalone tests"

patterns-established:
  - "RegisterCommand(cmd, groupID) for safe resource command registration from main.go"
  - "Partial update via cmd.Flags().Changed() for all update commands"
  - "Delete commands use ConfirmDelete then check IsQuiet for output"

requirements-completed: [SRCS-03, SRCS-04, SRCS-05]

# Metrics
duration: 4min
completed: 2026-03-12
---

# Phase 3 Plan 2: Create, Update, Delete Source Commands with Root Registration Summary

**Full CRUD commands for sources with partial update semantics, ConfirmDelete integration, and RegisterCommand pattern to avoid circular imports**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-12T05:57:39Z
- **Completed:** 2026-03-12T06:02:05Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Create command with --name (required), --type (required), --description (optional) sending POST to API
- Update command with partial update semantics via Flags().Changed() sending only modified fields
- Delete command with ConfirmDelete integration, --yes flag support, quiet mode suppression
- Sources command registered under "Core Resources" group via RegisterCommand pattern
- 10 new tests covering all create/update/delete behaviors

## Task Commits

Each task was committed atomically:

1. **Task 1: Create, update, and delete source commands with tests** - `09277ba` (test: RED) + `4b3013d` (feat: GREEN)
2. **Task 2: Wire sources command into root.go** - `d557066` (feat)

_Note: TDD tasks have two commits (RED test + GREEN implementation)_

## Files Created/Modified
- `cmd/sources/create.go` - Create command with --name, --type, --description flags and POST API call
- `cmd/sources/create_test.go` - Tests for create with flags, description, missing required flags
- `cmd/sources/update.go` - Update command with partial update via Flags().Changed()
- `cmd/sources/update_test.go` - Tests for partial update body and no-flags error
- `cmd/sources/delete.go` - Delete command with ConfirmDelete and quiet mode
- `cmd/sources/delete_test.go` - Tests for delete with --yes, quiet mode, JSON mode
- `cmd/sources/sources.go` - Updated init() to register create, update, delete subcommands
- `cmd/root.go` - Added RegisterCommand function for circular-import-safe registration
- `cmd/root_test.go` - Tests for RegisterCommand and --yes flag
- `main.go` - Register sources command in init() under "resources" group

## Decisions Made
- Used RegisterCommand pattern in cmd/root.go to avoid circular imports (cmd -> cmd/sources -> cmd). Resource commands that import cmd for APIClient/Output must be registered from main.go, not cmd/root.go.
- Delete tests register --yes flag locally on the command since persistent flags from rootCmd are not inherited when running commands standalone in tests.
- Description flag uses Flags().Changed() in create command too, only including it when explicitly set (consistent with update behavior).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Circular import between cmd and cmd/sources**
- **Found during:** Task 2 (Wire sources into root.go)
- **Issue:** Adding `import "cmd/sources"` to cmd/root.go created a circular import because cmd/sources already imports cmd for APIClient/Output
- **Fix:** Created RegisterCommand(cmd, groupID) function in cmd/root.go, moved sources registration to main.go init()
- **Files modified:** cmd/root.go, main.go
- **Verification:** go build ./... succeeds, go test ./... all pass
- **Committed in:** d557066

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Auto-fix was necessary to resolve Go circular import. RegisterCommand pattern is reusable for all future resource phases (4-9).

## Issues Encountered
None beyond the circular import deviation documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Full CRUD vertical slice for Sources complete (list, get, create, update, delete)
- RegisterCommand pattern established for registering future resource commands
- Partial update pattern with Flags().Changed() ready for replication
- ConfirmDelete integration proven for all delete commands
- All 51 tests pass across entire codebase

---
*Phase: 03-first-resource-sources*
*Completed: 2026-03-12*
