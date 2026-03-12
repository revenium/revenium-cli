---
phase: 03-first-resource-sources
plan: 01
subsystem: api
tags: [cobra, crud, sources, confirm-delete, tdd]

# Dependency graph
requires:
  - phase: 02-output-layer
    provides: "Formatter with Render/RenderTable/RenderJSON, TableDef, statusStyle"
  - phase: 01-project-scaffold-config
    provides: "API Client with Do(), rootCmd with persistent flags"
provides:
  - "ConfirmDelete shared helper in internal/resource/"
  - "--yes/-y global persistent flag on rootCmd"
  - "sources list command (GET /v2/api/sources)"
  - "sources get command (GET /v2/api/sources/{id})"
  - "cmd/sources package with tableDef, toRows, str, renderSource helpers"
affects: [03-first-resource-sources, 04-assets, 05-products, 06-contracts, 07-customers, 08-subscriptions]

# Tech tracking
tech-stack:
  added: []
  patterns: [resource-package-per-entity, confirm-delete-helper, map-string-interface-api-responses]

key-files:
  created:
    - internal/resource/resource.go
    - internal/resource/resource_test.go
    - cmd/sources/sources.go
    - cmd/sources/list.go
    - cmd/sources/list_test.go
    - cmd/sources/get.go
    - cmd/sources/get_test.go
  modified:
    - cmd/root.go

key-decisions:
  - "ConfirmDelete uses bufio.NewScanner for prompt, not Huh library -- keeps it simple for y/N"
  - "Sources use map[string]interface{} for API responses -- avoids coupling to exact schema"
  - "Empty list prints message in text mode but renders empty array in JSON mode"

patterns-established:
  - "Resource package pattern: cmd/{resource}/ with sources.go parent, verb files, tableDef, toRows, str helpers"
  - "ConfirmDelete(resourceType, id, skipConfirm, jsonMode) for all delete commands"
  - "renderSource helper for single-resource rendering (get, create, update)"

requirements-completed: [SRCS-01, SRCS-02, SRCS-05]

# Metrics
duration: 3min
completed: 2026-03-12
---

# Phase 3 Plan 1: Shared Resource Helpers and Sources List/Get Summary

**ConfirmDelete helper with --yes flag, sources list with empty-state handling, and sources get with single-row table rendering**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-12T05:52:01Z
- **Completed:** 2026-03-12T05:55:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- ConfirmDelete shared helper with skipConfirm, jsonMode, and non-TTY bypass logic
- --yes/-y persistent flag registered globally on rootCmd with YesMode() accessor
- sources list command with styled table, empty-state message, and JSON mode
- sources get command with single-row table and JSON mode

## Task Commits

Each task was committed atomically:

1. **Task 1: Create shared resource helper and --yes flag** - `c586570` (test: failing tests) + `95364e1` (feat: implementation)
2. **Task 2: Create sources list and get commands** - `4cae927` (test: failing tests) + `027e3df` (feat: implementation)

_Note: TDD tasks have two commits each (RED test + GREEN implementation)_

## Files Created/Modified
- `internal/resource/resource.go` - ConfirmDelete shared helper for all delete commands
- `internal/resource/resource_test.go` - Tests for ConfirmDelete bypass paths
- `cmd/root.go` - Added --yes/-y persistent flag and YesMode() export
- `cmd/sources/sources.go` - Parent command, tableDef, toRows, str, renderSource helpers
- `cmd/sources/list.go` - List all sources with table/JSON rendering
- `cmd/sources/list_test.go` - Tests for list with data, empty, JSON, empty JSON
- `cmd/sources/get.go` - Get single source by ID
- `cmd/sources/get_test.go` - Tests for get table, JSON, missing arg

## Decisions Made
- Used bufio.NewScanner for confirmation prompt (simple y/N does not justify Huh library)
- API responses use map[string]interface{} to avoid coupling to exact schema shape
- Empty list renders "No sources found." in text mode but empty JSON array in JSON mode
- Sources commands not yet registered in root.go (deferred to Plan 02 per plan spec)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Sources list and get commands ready, pending registration in root.go (Plan 02)
- ConfirmDelete helper ready for delete command implementation (Plan 02)
- Resource package pattern established for replication across Phases 4-9

---
*Phase: 03-first-resource-sources*
*Completed: 2026-03-12*
