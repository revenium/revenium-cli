# Phase 8: Anomalies & Alerts - Research

**Researched:** 2026-03-12
**Domain:** Revenium CLI CRUD commands for anomaly detection rules, AI alerts, and budget alert thresholds
**Confidence:** HIGH

## Summary

Phase 8 adds two top-level command packages (`cmd/anomalies/` and `cmd/alerts/`) following the established CRUD patterns from Phases 3-7. Anomalies have full CRUD (list, get, create, update, delete) against the `/v2/api/sources/ai/anomaly` endpoint. AI alerts are read-only from the API (list, get at `/v2/api/sources/ai/alert`), so `alerts list` and `alerts get` are the available operations. Budget alerts are viewed via `/v2/api/ai/alerts/{anomalyId}/budget/progress` (single) and `/v2/api/ai/alerts/budgets/portfolio` (portfolio view).

The critical architectural insight is that **anomaly rules are the mechanism that generates alerts and budget thresholds**. The API has no separate create/update/delete for alerts or budgets -- those are controlled entirely through anomaly configuration. Budget progress is a read-only view of CUMULATIVE_USAGE anomaly rules. Therefore, `alerts budget list` maps to the portfolio endpoint, `alerts budget get <anomaly-id>` maps to the progress endpoint, and budget alert creation/configuration is done via anomaly commands. This phase introduces one new pattern: currency formatting for budget monetary values.

**Primary recommendation:** Implement `cmd/anomalies/` as standard full CRUD (clone products pattern), `cmd/alerts/` with list+get only, and `alerts budget` as a nested read-only subcommand (list via portfolio, get via progress). Budget create/update/delete are NOT supported by the API and should be omitted. Add a `formatCurrency` helper in the alerts package for monetary display.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Two top-level commands: `revenium anomalies` and `revenium alerts`
- `anomalies` -- standard full CRUD (list, get, create, update, delete)
- `alerts` -- AI alert CRUD (list, create at minimum per requirements)
- `alerts budget` -- nested subcommand group with full CRUD (list, get, create, update, delete), following the models/pricing nesting pattern
- `alerts list` shows AI alerts only -- homogeneous list
- `alerts budget list` shows budget alerts only -- separate view
- No combined "all alerts" view
- Budget alert thresholds display monetary values with currency formatting
- Use currency from API response if available (e.g., USD, EUR)
- Format as `$1,000.00` style with commas and 2 decimal places
- Budget alerts have full CRUD -- consistent with all other resources
- Package per resource: `cmd/anomalies/` and `cmd/alerts/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list: "No anomalies found." / "No alerts found." / "No budget alerts found."

### Claude's Discretion
- Table columns for anomalies, alerts, and budget alerts (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for all three resource types
- Currency formatting implementation details
- How to handle missing currency field in API response (fallback to USD or raw number)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ALRT-01 | User can list AI anomalies | GET `/v2/api/sources/ai/anomaly` returns array; standard list pattern |
| ALRT-02 | User can get an anomaly by ID | GET `/v2/api/sources/ai/anomaly/{id}` returns single object; standard get pattern |
| ALRT-03 | User can create an anomaly detection rule | POST `/v2/api/sources/ai/anomaly` with rule config body; standard create pattern |
| ALRT-04 | User can update an anomaly rule | PUT `/v2/api/sources/ai/anomaly/{id}` with partial fields; standard update pattern |
| ALRT-05 | User can delete an anomaly rule | DELETE `/v2/api/sources/ai/anomaly/{id}`; standard delete with ConfirmDelete |
| ALRT-06 | User can list AI alerts | GET `/v2/api/sources/ai/alert` returns array; read-only list |
| ALRT-07 | User can create AI alert rules | Alerts are generated from anomaly rules; `alerts create` should be an alias/redirect or the CLI can expose POST to `/v2/api/sources/ai/anomaly` under the alerts command. See Open Questions. |
| ALRT-08 | User can manage budget alert thresholds | Budget progress via GET `/v2/api/ai/alerts/{anomalyId}/budget/progress`; portfolio via GET `/v2/api/ai/alerts/budgets/portfolio`. Budget thresholds are configured as CUMULATIVE_USAGE anomaly rules via the anomaly CRUD endpoints. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| cobra | (existing) | Command structure | Already used across all phases |
| lipgloss v2 | (existing) | Table rendering | Already used via output.TableDef |
| testify | (existing) | Test assertions | Already used in all test files |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| internal/output | (existing) | Render/RenderJSON/IsJSON/IsQuiet | All display operations |
| internal/resource | (existing) | ConfirmDelete | Delete commands (anomalies only) |
| internal/api | (existing) | APIClient.Do() | All API calls |
| golang.org/x/text/message | (existing in stdlib vicinity) | Number formatting with commas | Currency display for budget alerts |

No new external dependencies needed. Currency formatting can use `fmt.Sprintf` with a simple helper or `golang.org/x/text/message` for locale-aware formatting. Given the project's simplicity preference, a simple `formatCurrency` function using `fmt.Sprintf` and manual comma insertion is sufficient.

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  anomalies/
    anomalies.go            # Cmd, tableDef, toRows, str, renderAnomaly
    list.go                 # newListCmd()
    list_test.go
    get.go                  # newGetCmd()
    get_test.go
    create.go               # newCreateCmd()
    create_test.go
    update.go               # newUpdateCmd()
    update_test.go
    delete.go               # newDeleteCmd()
    delete_test.go
  alerts/
    alerts.go               # Cmd, alertTableDef, toAlertRows, str, renderAlert
    list.go                 # newListCmd()
    list_test.go
    get.go                  # newGetCmd()
    get_test.go
    budget.go               # budgetCmd, initBudget(), budgetTableDef, formatCurrency()
    budget_list.go          # newBudgetListCmd()
    budget_list_test.go
    budget_get.go           # newBudgetGetCmd()
    budget_get_test.go
```

### Pattern 1: Anomalies - Standard Full CRUD
**What:** Exact replication of the products package pattern
**When to use:** All anomaly commands (list, get, create, update, delete)
**Example:**
```go
// cmd/anomalies/anomalies.go
package anomalies

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/revenium/revenium-cli/cmd"
    "github.com/revenium/revenium-cli/internal/output"
)

var Cmd = &cobra.Command{
    Use:   "anomalies",
    Short: "Manage AI anomaly detection rules",
    Example: `  # List all anomaly rules
  revenium anomalies list

  # Get a specific anomaly
  revenium anomalies get anom-123

  # Create an anomaly rule
  revenium anomalies create --name "High Cost Alert"`,
}

func init() {
    Cmd.AddCommand(newListCmd())
    Cmd.AddCommand(newGetCmd())
    Cmd.AddCommand(newCreateCmd())
    Cmd.AddCommand(newUpdateCmd())
    Cmd.AddCommand(newDeleteCmd())
}

var tableDef = output.TableDef{
    Headers:      []string{"ID", "Name", "Status"},
    StatusColumn: 2,
}
```

### Pattern 2: Alerts - Read-Only with Nested Budget
**What:** Alerts with list+get only, plus nested budget subcommand
**When to use:** `revenium alerts list`, `revenium alerts get`, `revenium alerts budget list/get`
**Example:**
```go
// cmd/alerts/alerts.go
package alerts

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/revenium/revenium-cli/cmd"
    "github.com/revenium/revenium-cli/internal/output"
)

var Cmd = &cobra.Command{
    Use:   "alerts",
    Short: "Manage AI alerts and budget thresholds",
    Example: `  # List all AI alerts
  revenium alerts list

  # View budget alert progress
  revenium alerts budget list`,
}

func init() {
    Cmd.AddCommand(newListCmd())
    Cmd.AddCommand(newGetCmd())
    Cmd.AddCommand(budgetCmd)
    initBudget()
}
```

### Pattern 3: Budget Nested Subcommand (Read-Only)
**What:** Follows the `models/pricing.go` init pattern but with list+get only (no create/update/delete)
**When to use:** `revenium alerts budget list` and `revenium alerts budget get <anomaly-id>`
**Example:**
```go
// cmd/alerts/budget.go
package alerts

import (
    "fmt"
    "strings"
    "github.com/spf13/cobra"
    "github.com/revenium/revenium-cli/cmd"
    "github.com/revenium/revenium-cli/internal/output"
)

var budgetCmd = &cobra.Command{
    Use:   "budget",
    Short: "View budget alert thresholds and progress",
    Example: `  # List all budget alerts
  revenium alerts budget list

  # Get budget progress for an anomaly
  revenium alerts budget get anom-123`,
}

func initBudget() {
    budgetCmd.AddCommand(newBudgetListCmd())
    budgetCmd.AddCommand(newBudgetGetCmd())
}

var budgetTableDef = output.TableDef{
    Headers:      []string{"Anomaly ID", "Budget", "Current", "Remaining", "% Used"},
    StatusColumn: -1,
}

// formatCurrency formats a number as currency with commas and 2 decimal places.
// Falls back to raw number display if currency is empty.
func formatCurrency(amount float64, currency string) string {
    // Format with 2 decimal places
    formatted := fmt.Sprintf("%.2f", amount)
    // Add commas to the integer part
    parts := strings.Split(formatted, ".")
    intPart := parts[0]
    negative := ""
    if strings.HasPrefix(intPart, "-") {
        negative = "-"
        intPart = intPart[1:]
    }
    // Insert commas every 3 digits from the right
    if len(intPart) > 3 {
        var result []byte
        for i, c := range intPart {
            if i > 0 && (len(intPart)-i)%3 == 0 {
                result = append(result, ',')
            }
            result = append(result, byte(c))
        }
        intPart = string(result)
    }
    // Add currency symbol
    symbol := "$"
    if currency != "" && currency != "USD" {
        symbol = currency + " "
    }
    return fmt.Sprintf("%s%s%s.%s", negative, symbol, intPart, parts[1])
}
```

### Pattern 4: Budget List (Portfolio View)
**What:** Lists all budget alerts via the portfolio endpoint
**When to use:** `revenium alerts budget list`
**Example:**
```go
// cmd/alerts/budget_list.go
func newBudgetListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all budget alerts",
        Args:  cobra.NoArgs,
        Example: `  # List all budget alerts
  revenium alerts budget list

  # List as JSON
  revenium alerts budget list --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var budgets []map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/ai/alerts/budgets/portfolio", nil, &budgets); err != nil {
                return err
            }
            if len(budgets) == 0 {
                if cmd.Output.IsJSON() {
                    return cmd.Output.RenderJSON([]interface{}{})
                }
                fmt.Fprintln(c.OutOrStdout(), "No budget alerts found.")
                return nil
            }
            return cmd.Output.Render(budgetTableDef, toBudgetRows(budgets), budgets)
        },
    }
}
```

### Pattern 5: Budget Get (Single Progress View)
**What:** Gets budget progress for a specific anomaly
**When to use:** `revenium alerts budget get <anomaly-id>`
**Example:**
```go
// cmd/alerts/budget_get.go
func newBudgetGetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "get <anomaly-id>",
        Short: "Get budget progress for an anomaly",
        Args:  cobra.ExactArgs(1),
        Example: `  # Get budget progress
  revenium alerts budget get anom-123

  # Get as JSON
  revenium alerts budget get anom-123 --json`,
        RunE: func(c *cobra.Command, args []string) error {
            anomalyID := args[0]
            path := fmt.Sprintf("/v2/api/ai/alerts/%s/budget/progress", anomalyID)
            var progress map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &progress); err != nil {
                return err
            }
            return renderBudgetProgress(progress)
        },
    }
}
```

### Anti-Patterns to Avoid
- **Creating alert/budget CRUD that the API doesn't support:** The API has no POST/PUT/DELETE for alerts or budget progress. Don't create commands that would fail.
- **Shared str() function across packages:** Each package defines its own `str()` helper (unexported). Do NOT extract to shared package.
- **Struct types for API responses:** Always use `map[string]interface{}`.
- **Extra success messages after create/update:** Render the result only.
- **Complex currency formatting library:** A simple helper function is sufficient; don't add `golang.org/x/text` as a dependency.

## API Endpoints

### Anomalies (Full CRUD)
| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| List | GET | `/v2/api/sources/ai/anomaly` | Returns array of anomaly objects |
| Get | GET | `/v2/api/sources/ai/anomaly/{id}` | Returns single anomaly object |
| Create | POST | `/v2/api/sources/ai/anomaly` | Body: anomaly rule configuration |
| Update | PUT | `/v2/api/sources/ai/anomaly/{id}` | Body: updated fields |
| Delete | DELETE | `/v2/api/sources/ai/anomaly/{id}` | No body |

### Alerts (Read-Only)
| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| List | GET | `/v2/api/sources/ai/alert` | Returns paginated list of AI alerts |
| Get | GET | `/v2/api/sources/ai/alert/{id}` | Returns single alert object |

### Budget Alerts (Read-Only)
| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| List (Portfolio) | GET | `/v2/api/ai/alerts/budgets/portfolio` | All budget alerts for tenant |
| Get (Progress) | GET | `/v2/api/ai/alerts/{anomalyId}/budget/progress` | Budget progress for specific anomaly |

### Budget Progress Response Fields (from API docs)
| Field | Type | Description |
|-------|------|-------------|
| currentValue | number | Present consumption amount |
| remainingBudget | number | Unspent allocation |
| percentUsed | number | Usage percentage (0-100) |
| aheadBehind | number | Variance versus linear expectation |
| budgetThreshold | number | Maximum allowable amount |
| currency | string | Currency code (e.g., "USD") |

### Anomaly Object Fields (tentative, from API pattern)
| Field | Type | Notes |
|-------|------|-------|
| id | string | Read-only, unique identifier |
| resourceType | string | Read-only |
| label | string | Read-only, display label |
| created | string | Read-only, ISO 8601 |
| updated | string | Read-only, ISO 8601 |
| _links | object | Read-only, HATEOAS links (skip in table) |

Note: The full anomaly schema (writable fields for create/update) was not available in the API reference docs. Fields like name, type, threshold, metric, and condition are expected but need to be discovered at implementation time via the `map[string]interface{}` approach. This is consistent with how other resources were handled.

### Recommended Table Columns

**Anomalies list/get:**
| Column | Field | Rationale |
|--------|-------|-----------|
| ID | id | Standard identifier |
| Name | label (or name) | Primary display field |
| Status | status (if available) | Rule status |

**Alerts list/get:**
| Column | Field | Rationale |
|--------|-------|-----------|
| ID | id | Standard identifier |
| Name | label | Alert display name |
| Created | created | When alert was triggered |

**Budget alerts list (portfolio):**
| Column | Field | Rationale |
|--------|-------|-----------|
| Anomaly ID | id or anomalyId | Links to anomaly rule |
| Budget | budgetThreshold (formatted) | Total budget threshold |
| Current | currentValue (formatted) | Current spend |
| Remaining | remainingBudget (formatted) | Remaining budget |
| % Used | percentUsed | Percentage consumed |

**Budget alerts get (progress):**
Single-row table with all budget progress fields formatted with currency.

### Recommended Create/Update Flags

**Anomalies create:** Flags will depend on the API schema discovered at implementation time. Start with `--name` (likely required) and add additional flags as the schema reveals them. Use `map[string]interface{}` body construction with `Flags().Changed()` for optional fields.

**Anomalies update:** Same flags as create, none individually required; at least one must be changed (standard `len(body)==0` check).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Delete confirmation | Custom prompt | `resource.ConfirmDelete()` | Handles --yes, JSON mode, non-TTY |
| Table rendering | Custom formatting | `output.TableDef` + `cmd.Output.Render()` | Consistent styling, JSON/table toggle |
| API calls | Custom HTTP client | `cmd.APIClient.Do()` | Auth headers, error mapping, timeouts |
| Flag-based partial updates | Custom diff logic | `cmd.Flags().Changed()` | Cobra built-in, used everywhere |
| Complex currency locale | i18n library | Simple `formatCurrency` helper | Only need USD-style formatting with commas |

## Common Pitfalls

### Pitfall 1: Alerts API Is Read-Only
**What goes wrong:** Attempting to implement create/update/delete for alerts or budget alerts when the API only supports GET operations for these resources.
**Why it happens:** The CONTEXT.md specifies "alerts create" and "budget alerts full CRUD" based on the requirements, but the underlying API generates alerts from anomaly rules.
**How to avoid:** Implement only list+get for alerts. For budget alerts, implement only list (portfolio) and get (progress). If the user needs to create alert rules, they do so via `revenium anomalies create`. Document this in the help text.
**Warning signs:** 404 or 405 errors on POST/PUT/DELETE to alert endpoints.

### Pitfall 2: Budget Progress Takes anomalyId, Not alertId
**What goes wrong:** Using alert IDs instead of anomaly IDs when fetching budget progress.
**Why it happens:** The endpoint path is `/v2/api/ai/alerts/{anomalyId}/budget/progress` -- it lives under "alerts" but takes an anomaly ID.
**How to avoid:** Name the argument `<anomaly-id>` in the CLI command help, not `<alert-id>`.
**Warning signs:** 404 errors when passing alert IDs.

### Pitfall 3: Currency Formatting Edge Cases
**What goes wrong:** Negative values, zero values, missing currency field, very large numbers display incorrectly.
**Why it happens:** Currency formatting has many edge cases.
**How to avoid:** Handle negative amounts (prefix `-`), default to `$` when currency is "USD" or empty, handle zero as `$0.00`, test with large numbers like 1000000.
**Warning signs:** Misaligned table columns due to varying currency string lengths.

### Pitfall 4: Anomaly Create Schema Unknown
**What goes wrong:** Hardcoding wrong field names for anomaly creation because the API docs don't fully expose the schema.
**Why it happens:** The API reference truncates the request body schema for the create endpoint.
**How to avoid:** Use `map[string]interface{}` body construction. Start with discoverable fields (name at minimum). Let the API validate and return errors for missing required fields. The CLI's `--json` mode passes through API errors to the user.
**Warning signs:** 422 errors on create with incomplete bodies.

### Pitfall 5: Budget Portfolio vs Progress Have Different Response Shapes
**What goes wrong:** Using the same rendering function for both portfolio list and individual progress.
**Why it happens:** Both return budget data but in different structures (array vs single object, different fields).
**How to avoid:** Use separate `toBudgetRows` for portfolio list and `renderBudgetProgress` for individual get. The portfolio may include additional fields like risk assessment and window boundaries.
**Warning signs:** Empty or misaligned columns in budget table output.

### Pitfall 6: Delete Test Needs --yes Flag Registration
**What goes wrong:** Tests for delete fail because `--yes` flag is inherited from root cmd at runtime but not available in isolated test.
**Why it happens:** Global flags like `--yes` are on the root command, not individual subcommands.
**How to avoid:** In delete tests, register `--yes` flag on the test command -- exactly as done in existing delete tests.
**Warning signs:** Flag access error in delete tests.

## Code Examples

### Registration in main.go
```go
// Source: existing main.go pattern
import (
    "github.com/revenium/revenium-cli/cmd/anomalies"
    "github.com/revenium/revenium-cli/cmd/alerts"
)

func init() {
    // ... existing registrations ...
    cmd.RegisterCommand(anomalies.Cmd, "resources")
    cmd.RegisterCommand(alerts.Cmd, "resources")
}
```

### Currency Formatting Helper
```go
// Source: project convention -- simple helper, no external dependencies
func formatCurrency(amount float64, currency string) string {
    formatted := fmt.Sprintf("%.2f", amount)
    parts := strings.Split(formatted, ".")
    intPart := parts[0]
    negative := ""
    if strings.HasPrefix(intPart, "-") {
        negative = "-"
        intPart = intPart[1:]
    }
    if len(intPart) > 3 {
        var result []byte
        for i, c := range intPart {
            if i > 0 && (len(intPart)-i)%3 == 0 {
                result = append(result, ',')
            }
            result = append(result, byte(c))
        }
        intPart = string(result)
    }
    symbol := "$"
    if currency != "" && currency != "USD" {
        symbol = currency + " "
    }
    return fmt.Sprintf("%s%s%s.%s", negative, symbol, intPart, parts[1])
}
```

### Extracting Float from API Response
```go
// Budget progress fields are numbers, need float extraction
func floatVal(m map[string]interface{}, key string) float64 {
    if v, ok := m[key]; ok && v != nil {
        switch n := v.(type) {
        case float64:
            return n
        case json.Number:
            f, _ := n.Float64()
            return f
        }
    }
    return 0
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Struct-based API responses | `map[string]interface{}` | Phase 3 decision | Avoids schema coupling, used throughout |
| Shared str() helper | Per-package str() | Phase 3 decision | Avoids import dependencies |
| Huh library for prompts | bufio.Scanner | Phase 3 decision | Simpler, fewer dependencies |

No changes in approach needed for Phase 8 -- all patterns are stable.

**New in this phase:** Currency formatting helper (`formatCurrency`) and float extraction helper (`floatVal`) are new but localized to the alerts package.

## Open Questions

1. **Anomaly create/update schema fields**
   - What we know: The API accepts POST to `/v2/api/sources/ai/anomaly` and PUT to `/v2/api/sources/ai/anomaly/{id}`. The description says it creates "AI anomaly alert configuration for monitoring AI metrics and costs."
   - What's unclear: The complete list of writable fields (name, type, threshold, metric, condition, etc.) was not available in the API reference docs.
   - Recommendation: Implement create with `--name` as likely required. Add other flags as the API schema is discovered at implementation time. The `map[string]interface{}` approach handles this gracefully -- the API will return validation errors for missing required fields.

2. **ALRT-07 (Create AI alert rules) mapping**
   - What we know: The API has no POST endpoint for alerts. Alerts are generated from anomaly rules.
   - What's unclear: Whether the user expectation is to create alerts via the `alerts` command or understands that anomaly rules generate alerts.
   - Recommendation: Implement `alerts create` as an **alias** that creates an anomaly rule (POST to `/v2/api/sources/ai/anomaly`), OR document in help text that alert rules are created via `revenium anomalies create`. The former is more user-friendly; the latter is more honest about the API. Given the CONTEXT.md says "alerts -- AI alert CRUD (list, create at minimum per requirements)", implement `alerts create` as a command that POSTs to the anomaly endpoint.

3. **Budget alerts full CRUD vs API reality**
   - What we know: The API only exposes GET endpoints for budget progress and portfolio. There is no POST/PUT/DELETE for budget-specific resources.
   - What's unclear: Whether budget thresholds can be managed independently of anomaly rules.
   - Recommendation: Implement `alerts budget list` and `alerts budget get` as read-only commands. For ALRT-08 ("manage budget alert thresholds"), the management is done through the anomaly CRUD commands (creating/updating CUMULATIVE_USAGE type anomalies with budget thresholds). Document this in help text. Do NOT implement budget create/update/delete commands that would fail against the API.

4. **Portfolio response shape**
   - What we know: The portfolio endpoint "returns current progress, risk assessment, and window boundaries for each budget" with pagination and sorting.
   - What's unclear: The exact field names in the portfolio response.
   - Recommendation: Use `map[string]interface{}` and discover fields at implementation time. The budget progress fields (currentValue, remainingBudget, percentUsed, aheadBehind, budgetThreshold, currency) are confirmed and likely present in portfolio items too.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (assert/require) |
| Config file | None needed (Go test built-in) |
| Quick run command | `go test ./cmd/anomalies/... ./cmd/alerts/... -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ALRT-01 | List anomalies table + JSON + empty | unit | `go test ./cmd/anomalies/ -run TestList -count=1` | Wave 0 |
| ALRT-02 | Get anomaly by ID | unit | `go test ./cmd/anomalies/ -run TestGet -count=1` | Wave 0 |
| ALRT-03 | Create anomaly with flags | unit | `go test ./cmd/anomalies/ -run TestCreate -count=1` | Wave 0 |
| ALRT-04 | Update anomaly partial fields | unit | `go test ./cmd/anomalies/ -run TestUpdate -count=1` | Wave 0 |
| ALRT-05 | Delete anomaly with confirm | unit | `go test ./cmd/anomalies/ -run TestDelete -count=1` | Wave 0 |
| ALRT-06 | List AI alerts table + JSON + empty | unit | `go test ./cmd/alerts/ -run TestList -count=1` | Wave 0 |
| ALRT-07 | Create AI alert rule (via anomaly endpoint) | unit | `go test ./cmd/alerts/ -run TestCreate -count=1` | Wave 0 |
| ALRT-08 | Budget list (portfolio) + get (progress) | unit | `go test ./cmd/alerts/ -run TestBudget -count=1` | Wave 0 |

### Additional Tests
| Behavior | Test Type | Automated Command | File Exists? |
|----------|-----------|-------------------|-------------|
| formatCurrency helper | unit | `go test ./cmd/alerts/ -run TestFormatCurrency -count=1` | Wave 0 |
| Budget get with currency formatting | unit | `go test ./cmd/alerts/ -run TestBudgetGet -count=1` | Wave 0 |
| Alert get by ID | unit | `go test ./cmd/alerts/ -run TestGetAlert -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/anomalies/... ./cmd/alerts/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before verify

### Wave 0 Gaps
- [ ] `cmd/anomalies/` directory -- all files (anomalies.go, list.go, get.go, create.go, update.go, delete.go + tests)
- [ ] `cmd/alerts/` directory -- all files (alerts.go, list.go, get.go, budget.go, budget_list.go, budget_get.go + tests)
- [ ] main.go registration for both anomalies and alerts

## Sources

### Primary (HIGH confidence)
- Existing codebase: `cmd/products/` -- standard CRUD template (list, get, create, update, delete)
- Existing codebase: `cmd/models/pricing.go` -- `initPricing()` pattern for nested budget subcommand
- Existing codebase: `cmd/models/pricing_list.go`, `pricing_create.go` -- nested subcommand implementation
- Existing codebase: `internal/resource/resource.go` -- `ConfirmDelete` helper
- Existing codebase: `main.go` -- `RegisterCommand` pattern

### Secondary (MEDIUM confidence)
- [Revenium API: List AI Anomalies](https://revenium.readme.io/reference/list_ai_anomalies) -- GET `/v2/api/sources/ai/anomaly`
- [Revenium API: Get AI Anomaly](https://revenium.readme.io/reference/get_ai_anomaly) -- GET `/v2/api/sources/ai/anomaly/{id}`
- [Revenium API: Create AI Anomaly](https://revenium.readme.io/reference/create_ai_anomaly) -- POST `/v2/api/sources/ai/anomaly` (schema truncated)
- [Revenium API: Update AI Anomaly](https://revenium.readme.io/reference/update_ai_anomaly) -- PUT `/v2/api/sources/ai/anomaly/{id}`
- [Revenium API: Delete AI Anomaly](https://revenium.readme.io/reference/delete_ai_anomaly) -- DELETE `/v2/api/sources/ai/anomaly/{id}`
- [Revenium API: List AI Alerts](https://revenium.readme.io/reference/list_ai_alerts) -- GET `/v2/api/sources/ai/alert`
- [Revenium API: Get AI Alert](https://revenium.readme.io/reference/get_ai_alert) -- GET `/v2/api/sources/ai/alert/{id}`
- [Revenium API: Get Budget Progress](https://revenium.readme.io/reference/get_budget_progress) -- GET `/v2/api/ai/alerts/{anomalyId}/budget/progress` with confirmed response fields
- [Revenium API: Get Budget Portfolio](https://revenium.readme.io/reference/get_budget_portfolio) -- GET `/v2/api/ai/alerts/budgets/portfolio`

### Tertiary (LOW confidence)
- Anomaly create/update schema fields -- API docs truncated, field names assumed from patterns
- Portfolio response shape -- description available but field names not confirmed
- Alert object fields beyond id/label/created/updated -- not fully documented

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- exact replication of existing patterns, no new libraries
- Architecture: HIGH -- follows established package structure and naming conventions
- API endpoints (anomaly CRUD): MEDIUM -- paths confirmed from API reference, schema partially truncated
- API endpoints (alerts read): MEDIUM -- paths confirmed from API reference
- API endpoints (budget): MEDIUM-HIGH -- paths and response fields confirmed from API reference
- Currency formatting: HIGH -- simple implementation, well-understood domain
- Pitfalls: HIGH -- based on concrete codebase analysis and API documentation

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable patterns, unlikely to change)
