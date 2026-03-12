# Phase 6: Products & Tools - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

CRUD for Products (catalog entries) and Tools (tool registrations). Two independent resource packages following the established CRUD pattern. No nested resources, no special update semantics.

</domain>

<decisions>
## Implementation Decisions

### All Phase 3-5 Patterns Apply
- Package per resource: `cmd/products/` and `cmd/tools/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list → "No products found." / "No tools found."

### Claude's Discretion
- Table columns for products and tools (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for both resources
- Whether products and tools share any infrastructure beyond existing patterns
- Any resource-specific display considerations

</decisions>

<specifics>
## Specific Ideas

- Products and tools are independent resources with no known relationship requiring coordination
- Standard CRUD replication — no special patterns needed beyond what Phases 3-5 established

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/sources/` — Full CRUD template (list, get, create, update, delete)
- `cmd/subscribers/` — Standard CRUD without nested resources (closest pattern)
- `internal/resource/resource.go` — `ConfirmDelete()` shared helper
- `cmd/root.go` — `RegisterCommand()`, global flags (--yes, --json, --quiet, --verbose)

### Established Patterns
- Resource package exports `Cmd`, registers subcommands in `init()`
- `main.go` init() registers via `cmd.RegisterCommand(pkg.Cmd, "resources")`
- Tests: `httptest.NewServer` + `output.NewWithWriter` with `bytes.Buffer`

### Integration Points
- `cmd.RegisterCommand(products.Cmd, "resources")` in main.go
- `cmd.RegisterCommand(tools.Cmd, "resources")` in main.go

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-products-tools*
*Context gathered: 2026-03-12*
