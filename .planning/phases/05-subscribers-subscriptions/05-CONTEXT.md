# Phase 5: Subscribers & Subscriptions - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

CRUD for Subscribers (API consumers) and Subscriptions (subscriber-to-source mappings). Subscriptions support both full update (PUT) and partial update (PATCH). Two independent resource packages following the established CRUD pattern.

</domain>

<decisions>
## Implementation Decisions

### All Phase 3-4 Patterns Apply
- Package per resource: `cmd/subscribers/` and `cmd/subscriptions/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list → "No subscribers found." / "No subscriptions found."

### Claude's Discretion
- Table columns for subscribers and subscriptions (discover from API)
- Which flags for create/update on each resource
- How to handle SUBR-05 (PATCH partial update) — could be a `--patch` flag on update, or always use PATCH, or separate `patch` subcommand
- Whether subscriptions show related subscriber/source names or just IDs
- Any shared infrastructure between the two resources (if any)
- API endpoint paths for both resources

</decisions>

<specifics>
## Specific Ideas

- Subscribers and subscriptions are logically related (subscriptions link subscribers to sources) but should be independent command groups
- The subscription update having both PUT and PATCH is the main unique aspect of this phase

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/sources/` — Full CRUD template (list, get, create, update, delete)
- `cmd/models/` — CRUD with PATCH variant (Phase 4 established PATCH pattern)
- `cmd/models/pricing*.go` — Nested subcommand pattern (if subscriptions need nesting)
- `internal/resource/resource.go` — `ConfirmDelete()` shared helper
- `cmd/root.go` — `RegisterCommand()`, global flags (--yes, --json, --quiet, --verbose)

### Established Patterns
- Resource package exports `Cmd`, registers subcommands in `init()`
- `main.go` init() registers via `cmd.RegisterCommand(pkg.Cmd, "resources")`
- PATCH update pattern from `cmd/models/update.go`
- Tests: `httptest.NewServer` + `output.NewWithWriter` with `bytes.Buffer`

### Integration Points
- `cmd.RegisterCommand(subscribers.Cmd, "resources")` in main.go
- `cmd.RegisterCommand(subscriptions.Cmd, "resources")` in main.go

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-subscribers-subscriptions*
*Context gathered: 2026-03-12*
