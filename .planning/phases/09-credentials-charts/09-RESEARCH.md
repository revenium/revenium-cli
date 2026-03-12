# Phase 9: Credentials & Charts - Research

**Researched:** 2026-03-12
**Domain:** CRUD commands for provider credentials (with masking) and chart definitions
**Confidence:** HIGH

## Summary

Phase 9 adds two CRUD resources: provider credentials and chart definitions. Both follow the established patterns from Phases 3-8 (package-per-resource, `RegisterCommand()`, `map[string]interface{}`, styled tables, `--json`, `ConfirmDelete()`). The credentials resource has one unique aspect -- secret values must be masked in CLI display (e.g., `sk-****7f3a`). Charts are straightforward CRUD with no special patterns.

The API endpoints are confirmed via Revenium's API documentation: credentials at `/v2/api/credentials` and chart definitions at `/v2/api/reports/chart-definitions`. Both use standard REST verbs (GET/POST/PUT/DELETE). The credential response includes fields like `id`, `label`, `credentialType`, `provider`, and an associated `team` object. Chart definition response fields need discovery during implementation but follow the standard resource pattern (`id`, `resourceType`, `label`, `created`, `updated`).

**Primary recommendation:** Follow the products/tools CRUD template exactly for both resources. Add a `maskSecret()` helper in the credentials package for display masking. Discover exact API field names from live API responses during implementation, consistent with the project's `map[string]interface{}` approach.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Package per resource: `cmd/credentials/` and `cmd/charts/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list prints "No credentials found." / "No charts found."

### Claude's Discretion
- Table columns for credentials and charts (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for both resources
- Credential masking format (e.g., `sk-****7f3a` -- show last 4 chars with prefix)
- Whether masking happens client-side or if the API returns pre-masked values
- Whether to offer an `--unmask` flag or always mask
- How delete vs deactivate works for credentials (single `delete` command with `--deactivate` flag, or separate commands)
- Any shared infrastructure between the two resources

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CRED-01 | User can list provider credentials (masked display) | API: `GET /v2/api/credentials` confirmed. Masking helper covers display. |
| CRED-02 | User can get a provider credential by ID (masked) | API: `GET /v2/api/credentials/{id}` confirmed. Same masking applies. |
| CRED-03 | User can create a provider credential | API: `POST /v2/api/credentials` confirmed. Flags from API schema. |
| CRED-04 | User can update a provider credential | API: `PUT /v2/api/credentials/{id}` confirmed. Partial update via `Flags().Changed()`. |
| CRED-05 | User can delete/deactivate a provider credential | API: `DELETE /v2/api/credentials/{id}` confirmed. No separate deactivation endpoint found; use `--deactivate` flag if API supports status field, otherwise delete-only. |
| CHRT-01 | User can list chart definitions | API: `GET /v2/api/reports/chart-definitions` confirmed. |
| CHRT-02 | User can get a chart definition by ID | API: `GET /v2/api/reports/chart-definitions/{id}` confirmed. |
| CHRT-03 | User can create a chart definition | API: `POST /v2/api/reports/chart-definitions` confirmed. |
| CHRT-04 | User can update a chart definition | API: `PUT /v2/api/reports/chart-definitions/{id}` confirmed. |
| CHRT-05 | User can delete a chart definition | API: `DELETE /v2/api/reports/chart-definitions/{id}` confirmed. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| cobra | (existing) | Command framework | Already in use across all phases |
| lipgloss v2 | (existing) | Styled table output | Already in use for all resource tables |
| testify | (existing) | Test assertions | Already in use across all test files |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| net/http/httptest | stdlib | Mock API server in tests | Every test file |
| strings | stdlib | String manipulation for masking | maskSecret() helper |

No new dependencies needed. This phase uses only existing libraries.

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  credentials/
    credentials.go     # Cmd, tableDef, toRows, str, maskSecret, renderCredential
    list.go
    list_test.go
    get.go
    get_test.go
    create.go
    create_test.go
    update.go
    update_test.go
    delete.go
    delete_test.go
  charts/
    charts.go          # Cmd, tableDef, toRows, str, renderChart
    list.go
    list_test.go
    get.go
    get_test.go
    create.go
    create_test.go
    update.go
    update_test.go
    delete.go
    delete_test.go
```

### Pattern 1: Standard CRUD Resource (clone from products)
**What:** Each resource has a parent command, tableDef, toRows(), str(), renderX(), and five subcommands (list/get/create/update/delete).
**When to use:** Every CRUD resource in this CLI.
**Example:** (from `cmd/products/products.go`)
```go
var Cmd = &cobra.Command{
    Use:   "credentials",
    Short: "Manage provider credentials",
}

func init() {
    Cmd.AddCommand(newListCmd())
    Cmd.AddCommand(newGetCmd())
    Cmd.AddCommand(newCreateCmd())
    Cmd.AddCommand(newUpdateCmd())
    Cmd.AddCommand(newDeleteCmd())
}

var tableDef = output.TableDef{
    Headers:      []string{"ID", "Label", "Provider", "Type", "Secret"},
    StatusColumn: -1,
}
```

### Pattern 2: Secret Masking Helper
**What:** A function that masks sensitive values for display, showing only the last 4 characters with a prefix.
**When to use:** Credential list and get display.
**Example:**
```go
// maskSecret masks a secret value for display, showing prefix and last 4 chars.
// Examples: "sk-abc123xyz7f3a" -> "sk-****7f3a", "short" -> "****hort", "" -> ""
func maskSecret(value string) string {
    if value == "" {
        return ""
    }
    if len(value) <= 4 {
        return "****" + value
    }
    // Find prefix (characters before first hyphen, if any)
    if idx := strings.Index(value, "-"); idx > 0 && idx < len(value)-4 {
        return value[:idx+1] + "****" + value[len(value)-4:]
    }
    return "****" + value[len(value)-4:]
}
```

### Pattern 3: Registration in main.go
**What:** Two new RegisterCommand calls in main.go init().
**Example:**
```go
cmd.RegisterCommand(credentials.Cmd, "resources")
cmd.RegisterCommand(charts.Cmd, "resources")
```

### Anti-Patterns to Avoid
- **Logging or printing full secret values:** Never render unmasked secrets to stdout/stderr. Masking must happen before any output call.
- **Sharing str() across packages:** Each resource package defines its own `str()` helper. Do not create a shared utility -- this is the established pattern.
- **Using PATCH for updates:** Both resources use PUT for updates, consistent with products/tools pattern. Only models use PATCH.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Delete confirmation | Custom prompt logic | `resource.ConfirmDelete()` | Already handles --yes, JSON mode, non-TTY |
| Table rendering | Custom formatting | `output.TableDef` + `cmd.Output.Render()` | Handles styling, width, JSON mode |
| API calls | Custom HTTP logic | `cmd.APIClient.Do()` | Handles auth, errors, timeouts |

**Key insight:** 8 phases of CRUD resources have been built. The patterns are fully established. This phase is pure replication with one small addition (masking).

## Common Pitfalls

### Pitfall 1: Masking in JSON mode
**What goes wrong:** Masking secrets in JSON output when the user needs the raw API response.
**Why it happens:** Applying masking universally to all output modes.
**How to avoid:** Only mask in table display (toRows/renderCredential). In JSON mode (`--json`), pass the raw API response through unchanged. The API may already return masked values; if it returns plain text secrets, mask only for table output, not JSON. This mirrors how `--json` always shows raw API data in this CLI.
**Warning signs:** Test that JSON output does not contain masked values unless the API itself masks them.

### Pitfall 2: Chart definition endpoint path differs from other resources
**What goes wrong:** Using `/v2/api/chart-definitions` instead of the correct `/v2/api/reports/chart-definitions`.
**Why it happens:** Assuming all resources follow the same URL pattern.
**How to avoid:** Use the confirmed path: `/v2/api/reports/chart-definitions`.
**Warning signs:** 404 errors from the API.

### Pitfall 3: Credential fields may vary by provider
**What goes wrong:** Hardcoding field names that only apply to one provider type.
**Why it happens:** Testing with only one credential type.
**How to avoid:** Use `map[string]interface{}` (already the pattern) and extract fields defensively with `str()`. The table should show common fields; JSON mode shows everything.
**Warning signs:** Missing columns or empty values for certain credential types.

### Pitfall 4: Delete test needs --yes flag registration
**What goes wrong:** Delete tests fail because `--yes` flag is not available.
**Why it happens:** The `--yes` flag is a persistent flag on rootCmd, inherited at runtime but not in isolated test execution.
**How to avoid:** Register `--yes` flag in the test: `c.Flags().Bool("yes", false, "Skip confirmation prompts")` -- see `cmd/products/delete_test.go` for the exact pattern.
**Warning signs:** "unknown flag: --yes" in test output.

## Code Examples

### Credentials list command
```go
// Source: based on cmd/products/list.go pattern
func newListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all provider credentials",
        Args:  cobra.NoArgs,
        Example: `  # List all credentials
  revenium credentials list

  # List credentials as JSON
  revenium credentials list --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var creds []map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/credentials", nil, &creds); err != nil {
                return err
            }
            if len(creds) == 0 {
                if cmd.Output.IsJSON() {
                    return cmd.Output.RenderJSON([]interface{}{})
                }
                fmt.Fprintln(c.OutOrStdout(), "No credentials found.")
                return nil
            }
            return cmd.Output.Render(tableDef, toRows(creds), creds)
        },
    }
}
```

### Masking in toRows
```go
func toRows(creds []map[string]interface{}) [][]string {
    rows := make([][]string, len(creds))
    for i, c := range creds {
        rows[i] = []string{
            str(c, "id"),
            str(c, "label"),
            str(c, "provider"),
            str(c, "credentialType"),
            maskSecret(str(c, "apiKey")), // masked in table output
        }
    }
    return rows
}
```

### Charts list command
```go
// Source: based on cmd/products/list.go pattern
func newListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all chart definitions",
        Args:  cobra.NoArgs,
        Example: `  # List all chart definitions
  revenium charts list

  # List chart definitions as JSON
  revenium charts list --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var charts []map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/reports/chart-definitions", nil, &charts); err != nil {
                return err
            }
            if len(charts) == 0 {
                if cmd.Output.IsJSON() {
                    return cmd.Output.RenderJSON([]interface{}{})
                }
                fmt.Fprintln(c.OutOrStdout(), "No charts found.")
                return nil
            }
            return cmd.Output.Render(tableDef, toRows(charts), charts)
        },
    }
}
```

## API Endpoint Reference

### Credentials
| Operation | Method | Path |
|-----------|--------|------|
| List | GET | `/v2/api/credentials` |
| Get | GET | `/v2/api/credentials/{id}` |
| Create | POST | `/v2/api/credentials` |
| Update | PUT | `/v2/api/credentials/{id}` |
| Delete | DELETE | `/v2/api/credentials/{id}` |

**Known response fields:** `id`, `resourceType`, `label`, `credentialType`, `provider`, `team` (nested object with `id`, `label`), `created`, `updated`, `_links`
**Likely secret field:** `apiKey` or similar -- discover from live API response during implementation
**Suggested table columns:** ID, Label, Provider, Type, Secret (masked)

### Chart Definitions
| Operation | Method | Path |
|-----------|--------|------|
| List | GET | `/v2/api/reports/chart-definitions` |
| Get | GET | `/v2/api/reports/chart-definitions/{id}` |
| Create | POST | `/v2/api/reports/chart-definitions` |
| Update | PUT | `/v2/api/reports/chart-definitions/{id}` |
| Delete | DELETE | `/v2/api/reports/chart-definitions/{id}` |

**Known response fields:** `id`, `resourceType`, `label`, `created`, `updated`, `_links`
**Likely additional fields:** chart type, configuration, filters -- discover from live API
**Suggested table columns:** ID, Name/Label, Type, Created

## Discretion Recommendations

### Masking approach
**Recommendation:** Client-side masking in table display only. Do NOT mask in `--json` output. The API documentation shows credential fields like `label`, `provider`, `credentialType` but the secret field name needs discovery. Implement `maskSecret()` in the credentials package. Format: show prefix up to first hyphen + `****` + last 4 chars (e.g., `sk-****7f3a`). For values without hyphens, show `****` + last 4 chars.

### No --unmask flag
**Recommendation:** Do not offer `--unmask`. Users who need raw values can use `--json`. This keeps the CLI secure by default and avoids the complexity of an extra flag.

### Delete vs deactivate for credentials
**Recommendation:** Use a single `delete` command with a `--deactivate` flag. When `--deactivate` is passed, send a PUT request setting an `active: false` (or `status: "INACTIVE"`) field instead of a DELETE. If the API does not support a status/active field on credentials (which the docs suggest -- no deactivation endpoint was found), implement delete-only and skip `--deactivate`. Discover during implementation.

### Shared infrastructure
**Recommendation:** None needed. Each package is self-contained per the established pattern. The `maskSecret()` function lives in `cmd/credentials/` since only credentials need it. If a future phase needs masking, it can be moved to `internal/` at that time.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | None (Go test conventions) |
| Quick run command | `go test ./cmd/credentials/... ./cmd/charts/...` |
| Full suite command | `go test ./...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CRED-01 | List credentials with masked display | unit | `go test ./cmd/credentials/... -run TestList -x` | Wave 0 |
| CRED-02 | Get credential by ID with masked display | unit | `go test ./cmd/credentials/... -run TestGet -x` | Wave 0 |
| CRED-03 | Create credential | unit | `go test ./cmd/credentials/... -run TestCreate -x` | Wave 0 |
| CRED-04 | Update credential | unit | `go test ./cmd/credentials/... -run TestUpdate -x` | Wave 0 |
| CRED-05 | Delete credential | unit | `go test ./cmd/credentials/... -run TestDelete -x` | Wave 0 |
| CHRT-01 | List chart definitions | unit | `go test ./cmd/charts/... -run TestList -x` | Wave 0 |
| CHRT-02 | Get chart definition by ID | unit | `go test ./cmd/charts/... -run TestGet -x` | Wave 0 |
| CHRT-03 | Create chart definition | unit | `go test ./cmd/charts/... -run TestCreate -x` | Wave 0 |
| CHRT-04 | Update chart definition | unit | `go test ./cmd/charts/... -run TestUpdate -x` | Wave 0 |
| CHRT-05 | Delete chart definition | unit | `go test ./cmd/charts/... -run TestDelete -x` | Wave 0 |

### Additional masking tests
| Behavior | Test Type | Automated Command |
|----------|-----------|-------------------|
| maskSecret with prefix (sk-xxx) | unit | `go test ./cmd/credentials/... -run TestMaskSecret -x` |
| maskSecret with short value | unit | `go test ./cmd/credentials/... -run TestMaskSecret -x` |
| maskSecret with empty value | unit | `go test ./cmd/credentials/... -run TestMaskSecret -x` |
| JSON output shows unmasked values | unit | `go test ./cmd/credentials/... -run TestListJSON -x` |

### Sampling Rate
- **Per task commit:** `go test ./cmd/credentials/... ./cmd/charts/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/credentials/` directory and all files -- new package
- [ ] `cmd/charts/` directory and all files -- new package
- [ ] Test files for each command in both packages

## Sources

### Primary (HIGH confidence)
- [Revenium API Reference - Credentials](https://revenium.readme.io/reference/list_credentials) - Confirmed endpoint paths and response fields
- [Revenium API Reference - Chart Definitions](https://revenium.readme.io/reference/list_chart_definitions) - Confirmed endpoint paths
- Existing codebase `cmd/products/` - Template for CRUD pattern (verified via Read tool)
- Existing codebase `cmd/tools/` - Template for fields with provider/type columns (verified via Read tool)

### Secondary (MEDIUM confidence)
- [Revenium API Reference - Get Credential](https://revenium.readme.io/reference/get_credential) - Response schema with credentialType, provider, team fields
- [Revenium API Reference - Create Credential](https://revenium.readme.io/reference/create_credential) - POST endpoint confirmed, request body schema not fully visible
- [Revenium API Reference - Delete Credential](https://revenium.readme.io/reference/delete_credential) - No deactivation option documented

### Tertiary (LOW confidence)
- Credential secret field name (`apiKey` or similar) -- needs discovery from live API
- Chart definition additional fields beyond standard resource metadata -- needs discovery
- Whether API returns pre-masked or plain text secret values -- needs discovery

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies, same patterns as 8 prior phases
- Architecture: HIGH - Direct clone of products/tools pattern with one addition (masking)
- API endpoints: HIGH for paths, MEDIUM for field names (some need live discovery)
- Pitfalls: HIGH - Based on observed patterns in existing codebase
- Masking approach: MEDIUM - Secret field name needs discovery; masking logic is straightforward

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable -- patterns established, API unlikely to change)
