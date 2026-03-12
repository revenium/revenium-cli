# Phase 2: Output Layer - Context

**Gathered:** 2026-03-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Styled table rendering with Lip Gloss v2, `--json` flag for machine-readable output, TTY detection for pipe-safe output, `NO_COLOR` support, `--quiet` and `--verbose` flag integration. This creates the output infrastructure that all resource commands (Phase 3+) will use.

</domain>

<decisions>
## Implementation Decisions

### Table Appearance
- Rounded borders (Lip Gloss `lipgloss.RoundedBorder()`) — matches error box style from Phase 1
- Bold/colored headers, colored status values (green=active, red=inactive, yellow=pending), plain data cells
- Long values truncated at ~40 chars with ellipsis (…) — keeps tables compact
- Single resource `get` commands use same table format as `list` (single-row table, not key-value pairs)

### JSON Output Shape
- `--json` passes through raw API response — no wrapping, no transformation
- Pretty-printed (indented) by default
- List commands output a JSON array `[{...}, {...}]` — simple, works with `jq '.[]'`
- When `--json` is set and an error occurs, errors are also JSON: `{"error": "Invalid API key", "status": 401}` on stderr
- Exit codes remain non-zero on errors even in JSON mode

### TTY & Quiet/Verbose
- Claude's Discretion — TTY detection, NO_COLOR handling, and --quiet interaction are implementation details

### Claude's Discretion
- Output package internal API (function signatures, how resource commands register columns)
- TTY detection implementation (lipgloss.HasDarkBackground, isatty, etc.)
- NO_COLOR and TERM=dumb handling
- How --quiet interacts with --json (--quiet suppresses styled output, --json still outputs)
- How --verbose integrates with the output layer (already exists on root command from Phase 1)
- Table column width calculation and terminal width detection

</decisions>

<specifics>
## Specific Ideas

- Tables should feel consistent with the Lip Gloss error boxes from Phase 1 — rounded borders, similar visual language
- JSON mode should be completely clean — no styled output leaking into JSON, even partial styling
- The output layer should be designed so adding a new resource command in Phase 3+ is minimal boilerplate

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/errors/errors.go`: `RenderError()` uses Lip Gloss with `lipgloss.RoundedBorder()` — table style should match
- `internal/api/client.go`: `Client.Do()` unmarshals API response into `result interface{}` — output layer receives this data
- `cmd/root.go`: Already has `--verbose` persistent flag and command groups

### Established Patterns
- Lip Gloss v2 already imported and used for styling (`charm.land/lipgloss/v2`)
- Cobra persistent flags for global options (verbose already set up)
- Error handling via `internal/errors.APIError` with status codes

### Integration Points
- Output layer will be called from Cobra command `RunE` functions after `api.Client.Do()` returns
- `--json` and `--quiet` flags need to be added as persistent flags on root command (alongside `--verbose`)
- JSON error output needs to integrate with existing error handling in `main.go`

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-output-layer*
*Context gathered: 2026-03-12*
