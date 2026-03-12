---
phase: 08-anomalies-alerts
plan: 02
subsystem: api
tags: [cobra, alerts, budget, currency, crud, ai]

requires:
  - phase: 08-anomalies-alerts
    provides: Anomalies CRUD package (cmd/anomalies/) with Cmd export for registration
  - phase: 06-products-tools
    provides: CRUD command pattern (products.go, list.go, get.go, create.go, update.go, delete.go)
provides:
  - cmd/alerts/ package with AI alert list/get/create and nested budget full CRUD
  - Budget currency formatting with $1,000.00 style display
  - main.go registration of both anomalies.Cmd and alerts.Cmd
affects: [09-metrics-reporting]

tech-stack:
  added: []
  patterns: [budget CRUD proxying to anomaly API with CUMULATIVE_USAGE type, formatCurrency helper for monetary display]

key-files:
  created:
    - cmd/alerts/alerts.go
    - cmd/alerts/list.go
    - cmd/alerts/list_test.go
    - cmd/alerts/get.go
    - cmd/alerts/get_test.go
    - cmd/alerts/create.go
    - cmd/alerts/create_test.go
    - cmd/alerts/budget.go
    - cmd/alerts/budget_test.go
    - cmd/alerts/budget_list.go
    - cmd/alerts/budget_list_test.go
    - cmd/alerts/budget_get.go
    - cmd/alerts/budget_get_test.go
    - cmd/alerts/budget_create.go
    - cmd/alerts/budget_create_test.go
    - cmd/alerts/budget_update.go
    - cmd/alerts/budget_update_test.go
    - cmd/alerts/budget_delete.go
    - cmd/alerts/budget_delete_test.go
  modified:
    - main.go

key-decisions:
  - "Alert create posts to anomaly endpoint since alerts are generated from anomaly rules"
  - "Budget create/update/delete proxy to anomaly API with CUMULATIVE_USAGE type for full CRUD UX"
  - "formatCurrency uses $ prefix for USD/empty, currency code prefix for others (EUR 1,000.00)"

patterns-established:
  - "Budget CRUD proxying pattern: CLI exposes full CRUD, proxies create/update/delete to anomaly API"
  - "Currency formatting helper localized to alerts package"

requirements-completed: [ALRT-06, ALRT-07, ALRT-08]

duration: 2min
completed: 2026-03-12
---

# Phase 8 Plan 2: Alerts and Budget Commands Summary

**AI alert list/get/create with nested budget full CRUD (list, get, create, update, delete) featuring $1,000.00 currency formatting, all proxying budget writes to anomaly API**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-12T18:05:05Z
- **Completed:** 2026-03-12T18:08:00Z
- **Tasks:** 3
- **Files modified:** 20

## Accomplishments
- Complete cmd/alerts/ package with 3 alert commands (list, get, create) and 5 budget subcommands (list, get, create, update, delete)
- 18 passing tests covering table output, JSON output, empty handling, currency formatting, and all HTTP methods
- Budget create/update/delete proxy to anomaly API with CUMULATIVE_USAGE type for full CRUD consistency
- formatCurrency helper with comma-separated thousands and 2 decimal places ($1,000.00)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create alerts package with list, get, create commands** - `d05ed73` (feat)
2. **Task 2: Add budget subcommands with full CRUD and currency formatting** - `38be1ac` (feat)
3. **Task 3: Register anomalies and alerts commands in main.go** - `1615766` (feat)

## Files Created/Modified
- `cmd/alerts/alerts.go` - Package root with Cmd, alertTableDef, toAlertRows, str, renderAlert
- `cmd/alerts/list.go` - List all AI alerts (GET /v2/api/sources/ai/alert)
- `cmd/alerts/list_test.go` - List tests: table, empty, JSON, empty JSON
- `cmd/alerts/get.go` - Get alert by ID (GET /v2/api/sources/ai/alert/{id})
- `cmd/alerts/get_test.go` - Get tests: table, JSON
- `cmd/alerts/create.go` - Create alert rule (POST /v2/api/sources/ai/anomaly)
- `cmd/alerts/create_test.go` - Create tests: verify POST and body
- `cmd/alerts/budget.go` - Budget subcommand root with formatCurrency, floatVal, toBudgetRows
- `cmd/alerts/budget_test.go` - formatCurrency unit tests (9 cases)
- `cmd/alerts/budget_list.go` - List budget alerts via portfolio endpoint
- `cmd/alerts/budget_list_test.go` - Budget list tests: table with currency, empty, JSON
- `cmd/alerts/budget_get.go` - Get budget progress for anomaly
- `cmd/alerts/budget_get_test.go` - Budget get tests: table with currency, JSON
- `cmd/alerts/budget_create.go` - Create budget alert (POST anomaly with CUMULATIVE_USAGE)
- `cmd/alerts/budget_create_test.go` - Budget create tests: verify POST body has type and threshold
- `cmd/alerts/budget_update.go` - Update budget alert (PUT anomaly)
- `cmd/alerts/budget_update_test.go` - Budget update tests: verify PUT, no-fields error
- `cmd/alerts/budget_delete.go` - Delete budget alert with ConfirmDelete
- `cmd/alerts/budget_delete_test.go` - Budget delete tests: --yes, JSON auto-confirm
- `main.go` - Added anomalies.Cmd and alerts.Cmd registrations

## Decisions Made
- Alert create posts to `/v2/api/sources/ai/anomaly` since alerts are generated from anomaly rules (per CONTEXT.md)
- Budget create/update/delete proxy to the anomaly API with CUMULATIVE_USAGE type, giving users full CRUD UX while working within API constraints
- formatCurrency uses `$` prefix for USD or empty currency, currency code + space for others (e.g., "EUR 1,000.00")
- Budget get takes anomaly-id (not alert-id) per API design and research pitfall #2

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed test assertion for percent formatting**
- **Found during:** Task 2
- **Issue:** Test expected "75.1%" but 75.05 formatted with %.1f rounds to "75.0%"
- **Fix:** Corrected test assertion to match actual formatting behavior
- **Files modified:** cmd/alerts/budget_get_test.go
- **Committed in:** 38be1ac (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor test assertion correction. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 8 (Anomalies & Alerts) fully complete with both packages registered
- All 18 alert tests and 13 anomaly tests pass
- Full test suite green with no regressions
- Ready for Phase 9 (Metrics & Reporting)

---
*Phase: 08-anomalies-alerts*
*Completed: 2026-03-12*
