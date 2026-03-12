# Phase 10: Metrics - Research

**Researched:** 2026-03-12
**Domain:** Read-only metric queries across 9 metric types with time range filtering
**Confidence:** HIGH

## Summary

Phase 10 implements 9 read-only metric subcommands under `revenium metrics`. Unlike CRUD resources (Phases 3-9), metrics are query-only with shared time range filtering via `--from`/`--to` persistent flags on the parent command. The Revenium API exposes distinct endpoints per metric type under `/v2/api/sources/metrics/ai/...` for AI-related metrics, `/v2/api/sources/metrics/api` for API metrics, `/v2/api/traces` for traces, and `/v2/api/squads` for squad executions.

The established codebase patterns (exported `Cmd`, `RegisterCommand`, `map[string]interface{}`, `str()` helper, `output.TableDef`, `httptest.NewServer` tests) apply directly. The main new patterns are: (1) persistent flags on the parent metrics command for `--from`/`--to`, (2) a `formatNumber()` helper for comma-grouped numeric display, (3) query parameter injection into API paths, and (4) client-side trace grouping by traceId.

**Primary recommendation:** Structure as `cmd/metrics/` package with one file per subcommand plus `metrics.go` for the parent command, shared helpers (`str()`, `floatVal()`, `formatNumber()`), and persistent flag wiring. Use the models `initPricing()` pattern for wiring many subcommands from `init()`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- ISO 8601 date format only for --from/--to (e.g., 2024-01-15T00:00:00Z) -- no relative dates
- Smart defaults: omitting --from/--to defaults to last 24 hours
- Time range only -- no per-metric-type filters (like model ID or source ID)
- --from and --to as persistent flags on the metrics parent command -- all subcommands inherit them
- Standard table per metric type -- each subcommand defines its own tableDef with metric-appropriate columns
- Numeric values formatted with commas (e.g., 1,234,567) for readability in table output
- Traces grouped by traceId -- revenium metrics traces aggregates/groups results by traceId
- JSON output passes raw API data (no formatting applied)
- Package: cmd/metrics/
- RegisterCommand() in main.go for wiring
- map[string]interface{} for API responses, str() for extraction
- Long flags only, 2-3 help examples per command
- Empty result prints "No metrics found." / "No traces found." etc.
- --json passes raw API response

### Claude's Discretion
- Table columns for each metric type (discover from API)
- API endpoint paths for each metric type
- Number formatting implementation (comma helper function)
- How traceId grouping is rendered in table vs JSON
- Whether metrics response is paginated or single array
- Subcommand naming (e.g., tool-events vs tools)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| METR-01 | User can query AI metrics with --from and --to time range flags | API endpoint: GET /v2/api/sources/metrics/ai; persistent flags on parent command; query param injection for date range |
| METR-02 | User can query AI completion metrics | API endpoint: GET /v2/api/sources/metrics/ai/completions |
| METR-03 | User can query AI audio metrics | API endpoint: GET /v2/api/sources/metrics/ai/audio |
| METR-04 | User can query AI image metrics | API endpoint: GET /v2/api/sources/metrics/ai/images |
| METR-05 | User can query AI video metrics | API endpoint: GET /v2/api/sources/metrics/ai/video |
| METR-06 | User can query AI traces (aggregated by traceId) | API endpoint: GET /v2/api/traces; client-side grouping by traceId for table display |
| METR-07 | User can query squad metrics (multi-agent workflows) | API endpoint: GET /v2/api/squads for entity list, GET /v2/api/squad-executions for executions |
| METR-08 | User can query API metrics | API endpoint: GET /v2/api/sources/metrics/api |
| METR-09 | User can query tool event metrics | Endpoint discovery needed -- likely /v2/api/sources/metrics/tools or similar; may not have dedicated GET endpoint |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| cobra | (existing) | Command structure, persistent flags | Already used throughout project |
| lipgloss/v2 + table | (existing) | Styled table output | Already used via output.TableDef |
| encoding/json | stdlib | JSON parsing of API responses | Already used via map[string]interface{} pattern |
| time | stdlib | ISO 8601 parsing, 24h default calculation | Standard Go time handling |
| fmt | stdlib | Number formatting, string conversion | Already used for str(), formatCurrency() |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| stretchr/testify | (existing) | Test assertions | All test files |
| net/http/httptest | stdlib | Mock API server | All test files |

No new dependencies required.

## Architecture Patterns

### Recommended Project Structure
```
cmd/metrics/
  metrics.go          # Parent Cmd, persistent flags (--from, --to), shared helpers
  ai.go               # revenium metrics ai
  completions.go      # revenium metrics completions
  audio.go            # revenium metrics audio
  image.go            # revenium metrics image
  video.go            # revenium metrics video
  traces.go           # revenium metrics traces (traceId grouping)
  squads.go           # revenium metrics squads
  api.go              # revenium metrics api
  tool_events.go      # revenium metrics tool-events
  metrics_test.go     # Shared helper tests (formatNumber, buildPath)
  ai_test.go          # AI metrics tests
  completions_test.go # Completion metrics tests
  audio_test.go       # Audio metrics tests
  image_test.go       # Image metrics tests
  video_test.go       # Video metrics tests
  traces_test.go      # Traces tests
  squads_test.go      # Squads tests
  api_test.go         # API metrics tests
  tool_events_test.go # Tool events tests
```

### Pattern 1: Persistent Flags on Parent Command
**What:** `--from` and `--to` flags defined as PersistentFlags on the metrics parent command, inherited by all subcommands.
**When to use:** All 9 metric subcommands need the same time range parameters.
**Example:**
```go
// metrics.go
var fromFlag string
var toFlag string

var Cmd = &cobra.Command{
    Use:   "metrics",
    Short: "Query metrics and analytics",
    Example: `  # Query AI metrics for last 24 hours
  revenium metrics ai

  # Query completions with time range
  revenium metrics completions --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
}

func init() {
    Cmd.PersistentFlags().StringVar(&fromFlag, "from", "", "Start date (ISO 8601, e.g. 2024-01-15T00:00:00Z)")
    Cmd.PersistentFlags().StringVar(&toFlag, "to", "", "End date (ISO 8601, e.g. 2024-01-15T23:59:59Z)")

    Cmd.AddCommand(newAICmd())
    Cmd.AddCommand(newCompletionsCmd())
    Cmd.AddCommand(newAudioCmd())
    Cmd.AddCommand(newImageCmd())
    Cmd.AddCommand(newVideoCmd())
    Cmd.AddCommand(newTracesCmd())
    Cmd.AddCommand(newSquadsCmd())
    Cmd.AddCommand(newAPICmd())
    Cmd.AddCommand(newToolEventsCmd())
}
```

### Pattern 2: Query Parameter Path Builder
**What:** Helper function that appends --from/--to as query parameters to the API path, applying 24h default when omitted.
**When to use:** Every subcommand's RunE function before calling APIClient.Do().
**Example:**
```go
// buildPath constructs the API path with time range query parameters.
// When --from or --to are omitted, defaults to last 24 hours.
func buildPath(base string) string {
    from := fromFlag
    to := toFlag

    if from == "" && to == "" {
        now := time.Now().UTC()
        to = now.Format(time.RFC3339)
        from = now.Add(-24 * time.Hour).Format(time.RFC3339)
    }

    sep := "?"
    if strings.Contains(base, "?") {
        sep = "&"
    }
    path := base
    if from != "" {
        path += sep + "startDate=" + url.QueryEscape(from)
        sep = "&"
    }
    if to != "" {
        path += sep + "endDate=" + url.QueryEscape(to)
    }
    return path
}
```
Note: The actual query parameter names (startDate/endDate vs from/to) should be verified against the API during implementation. The Revenium API documentation does not explicitly document these parameter names.

### Pattern 3: Standard Metric Subcommand
**What:** Each metric type follows the same list-only pattern -- GET endpoint, check empty, render table or JSON.
**When to use:** All 9 subcommands follow this pattern.
**Example:**
```go
func newCompletionsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "completions",
        Short: "Query AI completion metrics",
        Args:  cobra.NoArgs,
        Example: `  # Query completion metrics for last 24 hours
  revenium metrics completions

  # Query with time range
  revenium metrics completions --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
        RunE: func(c *cobra.Command, args []string) error {
            var metrics []map[string]interface{}
            path := buildPath("/v2/api/sources/metrics/ai/completions")
            if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &metrics); err != nil {
                return err
            }
            if len(metrics) == 0 {
                if cmd.Output.IsJSON() {
                    return cmd.Output.RenderJSON([]interface{}{})
                }
                fmt.Fprintln(c.OutOrStdout(), "No metrics found.")
                return nil
            }
            return cmd.Output.Render(completionsTableDef, toCompletionRows(metrics), metrics)
        },
    }
}
```

### Pattern 4: Trace Grouping (Client-Side)
**What:** The traces subcommand groups results by traceId for meaningful display. The API returns individual trace entries; the CLI aggregates them.
**When to use:** Only for `revenium metrics traces`.
**Example:**
```go
func groupByTraceId(metrics []map[string]interface{}) []map[string]interface{} {
    groups := make(map[string]map[string]interface{})
    order := []string{}

    for _, m := range metrics {
        tid := str(m, "traceId")
        if _, exists := groups[tid]; !exists {
            groups[tid] = map[string]interface{}{
                "traceId": tid,
                "count":   0,
                // Aggregate fields as needed
            }
            order = append(order, tid)
        }
        g := groups[tid]
        // Increment count, sum costs, etc.
        count, _ := g["count"].(int)
        g["count"] = count + 1
    }

    result := make([]map[string]interface{}, len(order))
    for i, tid := range order {
        result[i] = groups[tid]
    }
    return result
}
```

### Anti-Patterns to Avoid
- **Separate flag definitions per subcommand:** Use PersistentFlags on parent -- do NOT duplicate --from/--to on each subcommand.
- **Custom HTTP client calls:** Always use `cmd.APIClient.Do()` -- never bypass the shared client.
- **Formatting in JSON mode:** JSON output must pass raw API data. Formatting (commas, grouping) is for table mode only.
- **Hardcoded time.Now() in tests:** Extract time generation so tests can use fixed timestamps.

## API Endpoints (Discovered)

| Subcommand | API Endpoint | Confidence |
|------------|-------------|------------|
| `metrics ai` | GET /v2/api/sources/metrics/ai | HIGH |
| `metrics completions` | GET /v2/api/sources/metrics/ai/completions | HIGH |
| `metrics audio` | GET /v2/api/sources/metrics/ai/audio | HIGH |
| `metrics image` | GET /v2/api/sources/metrics/ai/images | HIGH |
| `metrics video` | GET /v2/api/sources/metrics/ai/video | HIGH |
| `metrics traces` | GET /v2/api/traces | HIGH |
| `metrics squads` | GET /v2/api/squads | MEDIUM |
| `metrics api` | GET /v2/api/sources/metrics/api | HIGH |
| `metrics tool-events` | Unknown -- no dedicated GET endpoint found | LOW |

**Note on tool-events:** The API documentation shows a POST endpoint for metering tool events (`POST /v2/api/metering/tool`) but no dedicated GET endpoint for querying tool event metrics. Possible paths to investigate during implementation:
- `/v2/api/sources/metrics/tools` (by convention with other metric endpoints)
- `/v2/api/sources/metrics/tool-events`
- Tool events may be included in the general AI metrics endpoint

**Note on squads:** The API has both `/v2/api/squads/entities` (squad entity list) and `/v2/api/squads` or `/v2/api/squad-executions` (execution metrics). The `metrics squads` subcommand should likely use the execution-focused endpoint since this is a metrics context.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Number formatting with commas | Custom string manipulation | Adapt `formatCurrency()` from `cmd/alerts/budget.go` | Already proven pattern with negative number handling |
| ISO 8601 parsing | Custom date parser | `time.Parse(time.RFC3339, value)` | Standard Go, handles timezone offsets |
| Query string building | Manual string concat | `net/url` package `url.Values` | Handles escaping edge cases |
| Table rendering | Custom table formatter | `output.Render(tableDef, rows, data)` | Already built and tested |

**Key insight:** The entire output infrastructure exists. Every subcommand just needs a `tableDef`, a `toXxxRows()` conversion function, and a `RunE` that calls `buildPath()` + `APIClient.Do()` + `Output.Render()`.

## Common Pitfalls

### Pitfall 1: Query Parameter Names Unknown
**What goes wrong:** Using wrong parameter names (from/to vs startDate/endDate) causes API to ignore filters or return errors.
**Why it happens:** The Revenium API documentation does not explicitly document query parameter names for date range filtering on metric endpoints.
**How to avoid:** Test against the actual API during implementation. Try `startDate`/`endDate` first (common in Java APIs). Fall back to `from`/`to`. Use `--verbose` to inspect the actual request URL.
**Warning signs:** Getting all metrics regardless of date range specified.

### Pitfall 2: Paginated vs Array Response
**What goes wrong:** Expecting `[]map[string]interface{}` but getting a paginated wrapper object `{"content": [...], "page": {...}}`.
**Why it happens:** API docs mention "paginated list" for most metric endpoints.
**How to avoid:** Handle both shapes -- try array first, if decode fails try paginated wrapper. Or decode into `interface{}` first and type-switch.
**Warning signs:** Empty table despite data existing in the API.

### Pitfall 3: Traces Need Client-Side Aggregation
**What goes wrong:** Displaying raw trace entries produces hundreds of rows with duplicate traceIds, which is not meaningful.
**Why it happens:** The API returns individual entries; the CLI must group by traceId.
**How to avoid:** Always group traces by traceId before rendering. Show aggregate columns: count, total cost, total duration.
**Warning signs:** Table has many rows with the same traceId.

### Pitfall 4: formatNumber vs formatCurrency Confusion
**What goes wrong:** Using `formatCurrency()` for non-monetary values adds dollar signs.
**Why it happens:** Copy-paste from budget.go without adapting.
**How to avoid:** Create a separate `formatNumber()` that only adds comma grouping, no currency symbol and no decimal places for integers.

### Pitfall 5: Time Default Mismatch Between Table and JSON
**What goes wrong:** Table mode shows "last 24 hours" data but JSON mode shows all data because defaults weren't applied.
**Why it happens:** Applying defaults conditionally or in wrong place.
**How to avoid:** Apply 24h default in `buildPath()` before the API call, so both JSON and table modes get the same data.

## Code Examples

### formatNumber Helper
```go
// formatNumber formats an integer with comma grouping (e.g., 1234567 -> "1,234,567").
func formatNumber(n float64) string {
    intPart := fmt.Sprintf("%.0f", n)
    negative := ""
    if strings.HasPrefix(intPart, "-") {
        negative = "-"
        intPart = intPart[1:]
    }
    if len(intPart) <= 3 {
        return negative + intPart
    }
    var result []byte
    for i, c := range intPart {
        if i > 0 && (len(intPart)-i)%3 == 0 {
            result = append(result, ',')
        }
        result = append(result, byte(c))
    }
    return negative + string(result)
}
```

### Test Pattern for Metric Subcommand
```go
func TestCompletionMetrics(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/v2/api/sources/metrics/ai/completions", r.URL.Path)
        // Verify query params
        assert.NotEmpty(t, r.URL.Query().Get("startDate"))
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, `[{"id": "m-1", "model": "gpt-4", "totalTokens": 1500, "totalCost": 0.05}]`)
    }))
    defer srv.Close()

    var buf bytes.Buffer
    cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
    cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

    // Reset flags for test
    fromFlag = "2024-01-01T00:00:00Z"
    toFlag = "2024-01-31T23:59:59Z"

    c := newCompletionsCmd()
    c.SetOut(&buf)
    err := c.Execute()

    require.NoError(t, err)
    assert.Contains(t, buf.String(), "gpt-4")
}
```

### Registration in main.go
```go
// In main.go init():
cmd.RegisterCommand(metrics.Cmd, "monitoring")
```
Note: Metrics fits the "monitoring" group alongside anomalies and alerts.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single str() per package | Each package defines its own str() | Phase 3+ | Each metrics package needs its own str() copy |
| PUT for updates | Mix of PUT/PATCH | Phase 4+ | Not applicable -- metrics are read-only |
| Inline flag vars | Package-level flag vars | Phase 1+ | Use package-level fromFlag/toFlag |

## Open Questions

1. **Query parameter names for date range**
   - What we know: The API supports date range filtering on metric endpoints
   - What's unclear: Whether params are named `startDate`/`endDate`, `from`/`to`, or something else
   - Recommendation: Try `startDate`/`endDate` first (consistent with Java-based API conventions). Use `--verbose` to verify. Can be adjusted during implementation without architectural changes.

2. **Paginated vs array response shape**
   - What we know: API docs mention "paginated list" for most metric endpoints
   - What's unclear: Whether response is `[...]` array or `{"content": [...], "page": {...}}` wrapper
   - Recommendation: Implement for array response first (consistent with all other endpoints in codebase). If paginated wrapper found, extract `.content` before processing. This is a small adjustment.

3. **Tool events GET endpoint**
   - What we know: POST /v2/api/metering/tool exists for submitting tool events. No GET endpoint documented.
   - What's unclear: Whether a GET endpoint exists at /v2/api/sources/metrics/tools or similar
   - Recommendation: Try `/v2/api/sources/metrics/tools` by convention. If not found, try `/v2/api/sources/metrics/tool-events`. Document as LOW confidence and verify during implementation.

4. **Squads endpoint choice**
   - What we know: Both `/v2/api/squads/entities` (entity list) and `/v2/api/squads` (executions with metrics) exist
   - What's unclear: Which is more appropriate for CLI metrics display
   - Recommendation: Use `/v2/api/squads` (execution metrics) since this is a metrics context. The entities endpoint is more of a management view.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (existing) |
| Config file | None needed -- standard Go test conventions |
| Quick run command | `go test ./cmd/metrics/ -v -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| METR-01 | AI metrics with --from/--to | unit | `go test ./cmd/metrics/ -run TestAIMetrics -v` | Wave 0 |
| METR-02 | Completion metrics query | unit | `go test ./cmd/metrics/ -run TestCompletionMetrics -v` | Wave 0 |
| METR-03 | Audio metrics query | unit | `go test ./cmd/metrics/ -run TestAudioMetrics -v` | Wave 0 |
| METR-04 | Image metrics query | unit | `go test ./cmd/metrics/ -run TestImageMetrics -v` | Wave 0 |
| METR-05 | Video metrics query | unit | `go test ./cmd/metrics/ -run TestVideoMetrics -v` | Wave 0 |
| METR-06 | Traces grouped by traceId | unit | `go test ./cmd/metrics/ -run TestTraceMetrics -v` | Wave 0 |
| METR-07 | Squad metrics query | unit | `go test ./cmd/metrics/ -run TestSquadMetrics -v` | Wave 0 |
| METR-08 | API metrics query | unit | `go test ./cmd/metrics/ -run TestAPIMetrics -v` | Wave 0 |
| METR-09 | Tool event metrics query | unit | `go test ./cmd/metrics/ -run TestToolEventMetrics -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/metrics/ -v -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before /gsd:verify-work

### Wave 0 Gaps
- [ ] `cmd/metrics/` directory -- needs creation
- [ ] All test files listed above -- created alongside implementation files
- [ ] `formatNumber()` helper tests in `metrics_test.go`
- [ ] `buildPath()` helper tests in `metrics_test.go`

*(Existing test infrastructure -- Go test framework, testify, httptest -- covers all needs. No new tooling required.)*

## Sources

### Primary (HIGH confidence)
- Revenium API Reference (revenium.readme.io/reference) -- endpoint paths for all metric types
- Revenium API llms.txt (revenium.readme.io/llms.txt) -- comprehensive endpoint listing
- Existing codebase: cmd/alerts/budget.go -- formatCurrency(), floatVal() patterns
- Existing codebase: cmd/models/models.go -- initPricing() multi-subcommand pattern
- Existing codebase: cmd/sources/list.go -- standard list command pattern
- Existing codebase: cmd/root.go -- PersistentFlags pattern, RegisterCommand

### Secondary (MEDIUM confidence)
- Squads endpoint path (/v2/api/squads vs /v2/api/squads/entities) -- multiple endpoints exist, execution-focused one likely correct
- Query parameter names for date range -- API convention suggests startDate/endDate but not confirmed

### Tertiary (LOW confidence)
- Tool events GET endpoint -- no documented GET endpoint found; path must be discovered during implementation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all patterns established in phases 2-9
- Architecture: HIGH -- direct extension of existing package/command patterns
- API endpoints: HIGH for 7 of 9 types, MEDIUM for squads, LOW for tool-events
- Pitfalls: MEDIUM -- query parameter names and pagination shape need runtime verification

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable codebase patterns, API endpoints unlikely to change)
