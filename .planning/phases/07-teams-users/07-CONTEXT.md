# Phase 7: Teams & Users - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

CRUD for Teams (list, get, create, update, delete) plus nested prompt capture settings (get/set per team), and CRUD for Users (list, get, create, update, delete). Teams have a nested sub-feature similar to Phase 4's pricing dimensions pattern. Two independent resource packages plus the prompt-capture subcommand.

</domain>

<decisions>
## Implementation Decisions

### All Phase 3-6 Patterns Apply
- Package per resource: `cmd/teams/` and `cmd/users/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list → "No teams found." / "No users found."

### Claude's Discretion
- Table columns for teams and users (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for both resources
- Prompt capture subcommand nesting (follow Phase 4 `initPricing()` pattern)
- How prompt capture get/set displays and accepts settings
- Whether teams show member count or users show team name in list tables
- Any shared infrastructure between the two resources

</decisions>

<specifics>
## Specific Ideas

- Prompt capture settings follow the Phase 4 nested subcommand pattern: `revenium teams prompt-capture get <team-id>` / `revenium teams prompt-capture set <team-id>`
- Teams and users are logically related but should be independent command groups

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/sources/` — Full CRUD template (list, get, create, update, delete)
- `cmd/models/` — Nested subcommand pattern (`initPricing()` for pricing dimensions)
- `cmd/models/pricing.go` — Template for nested get/set subcommands under a parent resource
- `internal/resource/resource.go` — `ConfirmDelete()` shared helper
- `cmd/root.go` — `RegisterCommand()`, global flags (--yes, --json, --quiet, --verbose)

### Established Patterns
- Resource package exports `Cmd`, registers subcommands in `init()`
- Nested subcommands via `initNestedCmd()` pattern called from parent init()
- `main.go` init() registers via `cmd.RegisterCommand(pkg.Cmd, "resources")`
- Tests: `httptest.NewServer` + `output.NewWithWriter` with `bytes.Buffer`

### Integration Points
- `cmd.RegisterCommand(teams.Cmd, "resources")` in main.go
- `cmd.RegisterCommand(users.Cmd, "resources")` in main.go

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-teams-users*
*Context gathered: 2026-03-12*
