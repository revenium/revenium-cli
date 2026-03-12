# Phase 3: First Resource (Sources) - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Full CRUD for Sources (list, get, create, update, delete) — the first resource implementation that proves the pattern all subsequent resources (Phases 4-9) will follow. Includes shared CRUD helpers, delete confirmation, and command registration patterns.

</domain>

<decisions>
## Implementation Decisions

### Table Columns & Display
- `sources list` shows essential columns only: ID, Name, Type, Status — compact and scannable
- `sources get <id>` shows same columns as list, rendered as a single-row table (Phase 2 decision)
- Status column uses Phase 2 color palette (green=active, red=inactive, yellow=pending) via existing `statusStyle()`
- Empty list displays "No sources found." — no empty table rendered
- Users use `--json` to see all fields when needed

### Create/Update Flags
- Required vs optional flags discovered from the OpenAPI spec during research/planning
- Long flags only for resource fields (`--name`, `--type`, `--description`) — short flags reserved for global options (-v, -q, -y)
- `sources update <id>` uses partial update semantics — only sends fields the user explicitly passes
- After successful create/update: render the result as a single-row table (or JSON with --json). No extra "Created successfully" message.

### Delete Confirmation
- Prompt shows ID: "Delete source abc-123? [y/N]" — default is No
- `--yes` / `-y` is a global persistent flag on root command — works across all resources
- `--json` mode implies `--yes` — scripts shouldn't be blocked by prompts
- After successful delete: "Deleted source abc-123." — one line confirmation
- Non-TTY (piped input) should also imply `--yes` or fail safely

### Reusable CRUD Pattern
- Package per resource: `cmd/sources/` with list.go, get.go, create.go, update.go, delete.go
- Shared resource helpers: common functions like `ConfirmDelete()`, `RenderResult()`, reducing boilerplate for Phases 4-9
- Auto-fetch all pages transparently if API returns paginated results
- 2-3 help examples per command, consistent with Phase 1 decision

### Claude's Discretion
- Shared helper package location and API (internal/resource or similar)
- How to detect which flags were explicitly set (Cobra `cmd.Flags().Changed()`)
- Confirmation prompt implementation (fmt.Scan, bufio, or Huh library)
- Pagination detection and auto-fetch implementation
- Exact command registration pattern in root.go

</decisions>

<specifics>
## Specific Ideas

- This is the "prove it once, copy it everywhere" phase — whatever pattern emerges here gets replicated across 6+ resource types
- The shared helpers should make adding a new resource in Phase 4+ feel like filling in a template, not writing from scratch
- Command group: sources goes under "Core Resources" group (already defined in root.go)

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/output/output.go`: Formatter with `Render()` dispatching to table or JSON based on mode
- `internal/output/table.go`: `TableDef` struct with Headers and StatusColumn, `RenderTable()`, `Truncate()`
- `internal/output/json.go`: `RenderJSON()` for raw API passthrough, `RenderJSONError()` for error output
- `internal/api/client.go`: `Client.Do(ctx, method, path, body, result)` — handles all HTTP, auth, error mapping
- `cmd/root.go`: `APIClient` and `Output` exported vars, command groups defined, PersistentPreRunE wired
- `internal/errors/errors.go`: `APIError` with status codes, `RenderError()` styled box

### Established Patterns
- Cobra persistent flags for global options (verbose, json, quiet already on root)
- `PersistentPreRunE` skips config for version/config commands — resource commands get full init
- `cmd/config/` package pattern: subcommands in their own package, parent command exported as `Cmd`
- Error mapping: HTTP 401→"Invalid API key", 404→"Resource not found", etc.

### Integration Points
- New `cmd/sources/` package registers a `sourcesCmd` with `rootCmd` in root.go init()
- Each verb command calls `cmd.APIClient.Do()` then `cmd.Output.Render()`
- `--yes` flag needs to be added as persistent flag on rootCmd (alongside verbose, json, quiet)
- Sources commands go in the "resources" group (`GroupID: "resources"`)

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-first-resource-sources*
*Context gathered: 2026-03-12*
