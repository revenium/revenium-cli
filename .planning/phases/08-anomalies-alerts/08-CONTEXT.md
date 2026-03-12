# Phase 8: Anomalies & Alerts - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

AI anomaly detection rules (full CRUD) and alert management split into two top-level commands: `revenium anomalies` for anomaly rules, and `revenium alerts` for AI alerts plus nested budget alert thresholds. Budget alerts live under `revenium alerts budget` as a subcommand group (like models pricing).

</domain>

<decisions>
## Implementation Decisions

### Command Structure
- Two top-level commands: `revenium anomalies` and `revenium alerts`
- `anomalies` — standard full CRUD (list, get, create, update, delete)
- `alerts` — AI alert CRUD (list, create at minimum per requirements)
- `alerts budget` — nested subcommand group with full CRUD (list, get, create, update, delete), following the models/pricing nesting pattern

### Alert Separation
- `alerts list` shows AI alerts only — homogeneous list
- `alerts budget list` shows budget alerts only — separate view
- No combined "all alerts" view — each type has its own clean list

### Budget Alert Display
- Budget alert thresholds display monetary values with currency formatting
- Use currency from API response if available (e.g., USD, EUR)
- Format as `$1,000.00` style with commas and 2 decimal places
- Budget alerts have full CRUD (list, get, create, update, delete) — consistent with all other resources

### All Phase 3-7 Patterns Apply
- Package per resource: `cmd/anomalies/` and `cmd/alerts/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list → "No anomalies found." / "No alerts found." / "No budget alerts found."

### Claude's Discretion
- Table columns for anomalies, alerts, and budget alerts (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for all three resource types
- Currency formatting implementation details
- How to handle missing currency field in API response (fallback to USD or raw number)

</decisions>

<specifics>
## Specific Ideas

- Budget alerts under `alerts budget` mirrors the `models pricing` nesting pattern from Phase 4
- Currency formatting is new to this phase — a simple `formatCurrency(amount float64, currency string) string` helper in the alerts package
- Anomalies are straightforward CRUD with no special patterns

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/products/` — Standard CRUD template (closest pattern for anomalies)
- `cmd/models/pricing*.go` — Nested subcommand pattern for budget alerts
- `cmd/models/models.go` — `initPricing()` pattern for wiring nested commands
- `internal/resource/resource.go` — `ConfirmDelete()` shared helper
- `cmd/root.go` — `RegisterCommand()`, global flags

### Established Patterns
- Resource package exports `Cmd`, registers subcommands in `init()`
- Nested subcommands via `initNestedCmd()` called from parent init()
- Tests: `httptest.NewServer` + `output.NewWithWriter` with `bytes.Buffer`

### Integration Points
- `cmd.RegisterCommand(anomalies.Cmd, "resources")` in main.go
- `cmd.RegisterCommand(alerts.Cmd, "resources")` in main.go

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-anomalies-alerts*
*Context gathered: 2026-03-12*
