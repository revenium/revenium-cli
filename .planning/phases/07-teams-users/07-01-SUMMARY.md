---
phase: 07-teams-users
plan: 01
subsystem: api
tags: [cobra, teams, crud, prompt-capture, rest-api]

# Dependency graph
requires:
  - phase: 06-products-tools
    provides: "CRUD pattern, RegisterCommand, output.TableDef, resource.ConfirmDelete"
provides:
  - "Teams CRUD commands (list, get, create, update, delete)"
  - "Prompt capture get/set subcommands for team settings"
  - "Teams registered in main.go under resources group"
affects: [07-teams-users]

# Tech tracking
tech-stack:
  added: []
  patterns: [nested-subcommand-with-key-value-table, label-fallback-to-name]

key-files:
  created:
    - cmd/teams/teams.go
    - cmd/teams/list.go
    - cmd/teams/get.go
    - cmd/teams/create.go
    - cmd/teams/update.go
    - cmd/teams/delete.go
    - cmd/teams/prompt_capture.go
    - cmd/teams/prompt_capture_get.go
    - cmd/teams/prompt_capture_set.go
    - cmd/teams/list_test.go
    - cmd/teams/get_test.go
    - cmd/teams/create_test.go
    - cmd/teams/update_test.go
    - cmd/teams/delete_test.go
    - cmd/teams/prompt_capture_get_test.go
    - cmd/teams/prompt_capture_set_test.go
  modified:
    - main.go

key-decisions:
  - "Teams table shows ID and Name (from label field with name fallback); no status column"
  - "Prompt capture settings rendered as sorted key-value table, skipping _links"
  - "Bool flags for prompt-capture set require --enabled=true syntax (cobra bool flag behavior)"

patterns-established:
  - "Key-value table pattern: renderPromptSettings iterates map, sorts by key, renders Setting/Value columns"
  - "Label fallback: use label field for display name, fall back to name if label is empty"

requirements-completed: [TEAM-01, TEAM-02, TEAM-03, TEAM-04, TEAM-05, TEAM-06, TEAM-07]

# Metrics
duration: 3min
completed: 2026-03-12
---

# Phase 7 Plan 1: Teams CRUD and Prompt Capture Summary

**Full teams CRUD with nested prompt-capture get/set subcommands using key-value table rendering**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-12T16:52:23Z
- **Completed:** 2026-03-12T16:55:51Z
- **Tasks:** 2
- **Files modified:** 17

## Accomplishments
- Complete teams CRUD (list, get, create, update, delete) hitting /v2/api/teams endpoints
- Prompt capture nested subcommands (get/set) for /v2/api/teams/{id}/settings/prompts
- 17 tests covering all commands including JSON mode, empty list, quiet mode, delete confirmation
- Teams registered in main.go and available in CLI help

## Task Commits

Each task was committed atomically:

1. **Task 1: Create teams package with CRUD commands and prompt-capture subcommands** - `9435a22` (feat)
2. **Task 2: Add tests and register teams in main.go** - `2d2974e` (feat)

## Files Created/Modified
- `cmd/teams/teams.go` - Parent command, tableDef, toRows, str, renderTeam helpers
- `cmd/teams/list.go` - List all teams with empty-state handling
- `cmd/teams/get.go` - Get single team by ID
- `cmd/teams/create.go` - Create team with --name (required) and --description (optional)
- `cmd/teams/update.go` - Update team with changed-only fields
- `cmd/teams/delete.go` - Delete with confirmation via resource.ConfirmDelete
- `cmd/teams/prompt_capture.go` - Parent subcommand, key-value table rendering
- `cmd/teams/prompt_capture_get.go` - GET prompt capture settings
- `cmd/teams/prompt_capture_set.go` - PUT prompt capture settings with --enabled and --max-prompt-length
- `cmd/teams/*_test.go` - 7 test files with 17 test functions
- `main.go` - Added teams import and RegisterCommand call

## Decisions Made
- Teams table shows ID and Name (using "label" field with "name" fallback) with no status column
- Prompt capture settings rendered as sorted key-value table, filtering out "_links" key
- Bool flags require `--enabled=true` syntax due to cobra bool flag parsing behavior

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed bool flag parsing in prompt-capture set test**
- **Found during:** Task 2 (test creation)
- **Issue:** `--enabled true` parsed as --enabled flag + "true" positional arg by cobra (bool flag behavior)
- **Fix:** Changed test to use `--enabled=true` syntax
- **Files modified:** cmd/teams/prompt_capture_set_test.go
- **Verification:** All 17 tests pass
- **Committed in:** 2d2974e (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor test syntax fix. No scope creep.

## Issues Encountered
None beyond the bool flag parsing addressed above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Teams package complete and tested
- Pattern established for nested key-value settings subcommands
- Ready for remaining 07-teams-users plans (users, etc.)

## Self-Check: PASSED

All 16 source/test files verified present. Both task commits (9435a22, 2d2974e) verified in git log.

---
*Phase: 07-teams-users*
*Completed: 2026-03-12*
