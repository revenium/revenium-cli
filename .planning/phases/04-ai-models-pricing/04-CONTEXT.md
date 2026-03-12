# Phase 4: AI Models & Pricing - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

CRUD for AI Models (list, get, update via PATCH, delete) and nested pricing dimension management (list, create, update, delete under a specific model). This is the first resource with nested child resources — `revenium models pricing list <model-id>`. No model creation (models are discovered by the platform, not user-created).

</domain>

<decisions>
## Implementation Decisions

### All Phase 3 Patterns Apply
- Package per resource: `cmd/models/` with verb files
- `RegisterCommand()` in `main.go` for wiring
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list → "No models found." / "No pricing dimensions found."

### Claude's Discretion
- Table columns for models list (ID, Name, Type/Provider, Status — or whatever fields the API returns that are most useful)
- Table columns for pricing dimensions list
- How model-id is passed to pricing subcommands (positional arg: `models pricing list <model-id>`)
- Whether AIMD-03 (PATCH) uses a different HTTP method than regular update
- Subcommand nesting structure for `models pricing`
- Which flags are available for pricing dimension create/update
- How to structure the `cmd/models/` package to handle both model and pricing commands

</decisions>

<specifics>
## Specific Ideas

- This is the first nested resource — the pricing subcommand pattern should be clean enough to reuse if other resources need nesting later
- Models are likely platform-managed (auto-discovered from AI providers), so there may not be a `models create` command — just list, get, update pricing, delete
- `models pricing` commands always require a model-id to scope the operation

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/sources/sources.go`: Full CRUD pattern template — `tableDef`, `toRows()`, `str()`, `renderSource()`
- `cmd/sources/delete.go`: Delete with `ConfirmDelete()` helper pattern
- `cmd/sources/update.go`: Partial update via `Flags().Changed()` pattern
- `internal/resource/resource.go`: `ConfirmDelete()` shared helper
- `cmd/root.go`: `RegisterCommand()` for wiring, `--yes`/`-y` global flag
- `main.go`: Resource registration via `cmd.RegisterCommand(models.Cmd, "resources")`

### Established Patterns
- Resource package exports `Cmd` var, registers subcommands in `init()`
- Commands access `cmd.APIClient` and `cmd.Output` package vars
- Tests use `httptest.NewServer` + `output.NewWithWriter` with `bytes.Buffer`
- Cobra `cobra.ExactArgs(1)` for ID arguments

### Integration Points
- New `cmd/models/` package with `Cmd` var
- Registration: `cmd.RegisterCommand(models.Cmd, "resources")` in main.go init()
- Pricing subcommand group under models: `revenium models pricing ...`

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-ai-models-pricing*
*Context gathered: 2026-03-12*
