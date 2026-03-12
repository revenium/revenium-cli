# Phase 10: Metrics - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Read-only metric queries across 9 metric types: AI, completions, audio, image, video, traces, squads, API, and tool events. Each metric type is a subcommand under `revenium metrics`. All share time range filtering via `--from` and `--to` flags on the parent command. No CRUD — metrics are read-only queries.

</domain>

<decisions>
## Implementation Decisions

### Time Range Flag Design
- ISO 8601 date format only (e.g., `2024-01-15T00:00:00Z`) — no relative dates like "7d" or "last week"
- Smart defaults: omitting `--from`/`--to` defaults to last 24 hours
- Time range only — no per-metric-type filters (like model ID or source ID)
- `--from` and `--to` as persistent flags on the `metrics` parent command — all subcommands inherit them

### Metric Display Format
- Standard table per metric type — each subcommand defines its own tableDef with metric-appropriate columns
- Numeric values formatted with commas (e.g., `1,234,567`) for readability in table output
- Traces grouped by traceId — `revenium metrics traces` aggregates/groups results by traceId for meaningful display
- JSON output passes raw API data (no formatting applied)

### All Phase 2-9 Patterns Apply
- Package: `cmd/metrics/`
- `RegisterCommand()` in `main.go` for wiring
- `map[string]interface{}` for API responses, `str()` for extraction
- Long flags only, 2-3 help examples per command
- Empty result → "No metrics found." / "No traces found." etc.
- `--json` passes raw API response

### Claude's Discretion
- Table columns for each metric type (discover from API)
- API endpoint paths for each metric type
- Number formatting implementation (comma helper function)
- How traceId grouping is rendered in table vs JSON
- Whether metrics response is paginated or single array
- Subcommand naming (e.g., `tool-events` vs `tools`)

</decisions>

<specifics>
## Specific Ideas

- Metrics are the first read-only resource — no create/update/delete commands, just list/query
- A `formatNumber()` helper for comma-grouped numeric display (similar to `formatCurrency()` in alerts)
- Parent persistent flags (`--from`, `--to`) reduce boilerplate across 9 subcommands
- Smart 24-hour default means the CLI is useful without any flags: `revenium metrics ai`
- Traces are the most complex subcommand — grouping by traceId may require client-side aggregation

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/sources/` — Standard list pattern (closest for basic metric queries)
- `cmd/alerts/budget.go` — `formatCurrency()` as reference for `formatNumber()` helper
- `cmd/alerts/budget.go` — `floatVal()` for extracting numeric values from map[string]interface{}
- `cmd/root.go` — `RegisterCommand()`, global flags, persistent flag pattern
- `cmd/models/models.go` — `initPricing()` as pattern for wiring many subcommands from parent init()

### Established Patterns
- Resource package exports `Cmd`, registers subcommands in `init()`
- Display helpers (str(), formatCurrency(), maskSecret()) live in the resource package
- Tests: `httptest.NewServer` + `output.NewWithWriter` with `bytes.Buffer`
- Persistent flags on parent command inherited by all subcommands (e.g., `--from`, `--to`)

### Integration Points
- `cmd.RegisterCommand(metrics.Cmd, "resources")` in main.go

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 10-metrics*
*Context gathered: 2026-03-12*
