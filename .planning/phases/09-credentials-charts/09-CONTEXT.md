# Phase 9: Credentials & Charts - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

CRUD for provider credentials (with masked secret display) and chart definitions. Credentials have a unique masking requirement — secret values display as masked (e.g., `sk-****7f3a`) in both table and detail views. Credential delete may also support deactivation. Charts are standard CRUD with no special patterns.

</domain>

<decisions>
## Implementation Decisions

### All Phase 3-8 Patterns Apply
- Package per resource: `cmd/credentials/` and `cmd/charts/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list → "No credentials found." / "No charts found."

### Claude's Discretion
- Table columns for credentials and charts (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for both resources
- Credential masking format (e.g., `sk-****7f3a` — show last 4 chars with prefix)
- Whether masking happens client-side or if the API returns pre-masked values
- Whether to offer an `--unmask` flag or always mask
- How delete vs deactivate works for credentials (single `delete` command with `--deactivate` flag, or separate commands)
- Any shared infrastructure between the two resources

</decisions>

<specifics>
## Specific Ideas

- Credential masking is the first security-sensitive display in the CLI — the masking helper should be clean and reusable
- Charts are straightforward CRUD with no special patterns
- If the API returns secrets in plain text, the CLI must mask them before display (never show full secrets in table output)

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/products/` — Standard CRUD template (closest pattern for charts)
- `cmd/alerts/budget.go` — `formatCurrency()` as an example of a display helper in a resource package
- `internal/resource/resource.go` — `ConfirmDelete()` shared helper
- `cmd/root.go` — `RegisterCommand()`, global flags

### Established Patterns
- Resource package exports `Cmd`, registers subcommands in `init()`
- Display helpers (str(), formatCurrency()) live in the resource package
- Tests: `httptest.NewServer` + `output.NewWithWriter` with `bytes.Buffer`

### Integration Points
- `cmd.RegisterCommand(credentials.Cmd, "resources")` in main.go
- `cmd.RegisterCommand(charts.Cmd, "resources")` in main.go

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-credentials-charts*
*Context gathered: 2026-03-12*
