# Phase 7: Teams & Users - Research

**Researched:** 2026-03-12
**Domain:** Revenium CLI CRUD commands for Teams and Users resources, plus nested prompt-capture settings
**Confidence:** HIGH

## Summary

Phase 7 adds two independent resource command packages (`cmd/teams/` and `cmd/users/`) following the exact same patterns established in Phases 3-6. Teams have standard CRUD (list, get, create, update, delete) plus a nested `prompt-capture` subcommand for get/set operations. Users have standard CRUD. Both follow the `map[string]interface{}` response handling, styled table output, JSON mode, and `ConfirmDelete` patterns already proven across sources, models, products, subscribers, subscriptions, and tools.

The Revenium API endpoints are well-documented at `https://revenium.readme.io/reference`. Teams use `/v2/api/teams` and users use `/v2/api/users`. Prompt capture settings use `/v2/api/teams/{id}/settings/prompts` with GET and PUT methods. The API follows consistent patterns with the existing resources, so implementation is mechanical replication of the established CRUD template.

**Primary recommendation:** Clone the `cmd/products/` package structure for both teams and users CRUD, then add `prompt-capture` nested subcommand to teams following the `cmd/models/pricing.go` pattern (initPromptCapture called from teams init).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Package per resource: `cmd/teams/` and `cmd/users/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list: "No teams found." / "No users found."

### Claude's Discretion
- Table columns for teams and users (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for both resources
- Prompt capture subcommand nesting (follow Phase 4 `initPricing()` pattern)
- How prompt capture get/set displays and accepts settings
- Whether teams show member count or users show team name in list tables
- Any shared infrastructure between the two resources

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TEAM-01 | User can list all teams | GET `/v2/api/teams` returns array of team objects; use list pattern from products |
| TEAM-02 | User can get a team by ID | GET `/v2/api/teams/{id}` returns single team; use get pattern from products |
| TEAM-03 | User can create a new team | POST `/v2/api/teams` with name/description body; use create pattern from products |
| TEAM-04 | User can update a team | PUT `/v2/api/teams/{id}` with name/description/logo body; use update pattern from products |
| TEAM-05 | User can delete a team with confirmation | DELETE `/v2/api/teams/{id}` with ConfirmDelete; use delete pattern from products |
| TEAM-06 | User can view prompt capture settings for a team | GET `/v2/api/teams/{id}/settings/prompts`; nested subcommand pattern from models/pricing |
| TEAM-07 | User can update prompt capture settings for a team | PUT `/v2/api/teams/{id}/settings/prompts`; nested subcommand with flags for settings |
| USER-01 | User can list all users | GET `/v2/api/users` returns array of user objects; use list pattern |
| USER-02 | User can get a user by ID | GET `/v2/api/users/{id}` returns single user; use get pattern |
| USER-03 | User can create a new user | POST `/v2/api/users` with email/firstName/lastName/roles/teamIds; use create pattern |
| USER-04 | User can update a user | PUT `/v2/api/users/{id}` with updatable fields; use update pattern |
| USER-05 | User can delete a user with confirmation | DELETE `/v2/api/users/{id}` with ConfirmDelete; use delete pattern |
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
| internal/resource | (existing) | ConfirmDelete | Delete commands for both resources |
| internal/api | (existing) | APIClient.Do() | All API calls |

No new dependencies needed. This phase uses only existing infrastructure.

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  teams/
    teams.go              # Cmd, tableDef, toRows, str, renderTeam
    list.go               # newListCmd()
    list_test.go
    get.go                # newGetCmd()
    get_test.go
    create.go             # newCreateCmd()
    create_test.go
    update.go             # newUpdateCmd()
    update_test.go
    delete.go             # newDeleteCmd()
    delete_test.go
    prompt_capture.go     # promptCaptureCmd, initPromptCapture(), rendering helpers
    prompt_capture_get.go # newPromptCaptureGetCmd()
    prompt_capture_get_test.go
    prompt_capture_set.go # newPromptCaptureSetCmd()
    prompt_capture_set_test.go
  users/
    users.go              # Cmd, tableDef, toRows, str, renderUser
    list.go               # newListCmd()
    list_test.go
    get.go                # newGetCmd()
    get_test.go
    create.go             # newCreateCmd()
    create_test.go
    update.go             # newUpdateCmd()
    update_test.go
    delete.go             # newDeleteCmd()
    delete_test.go
```

### Pattern 1: Standard CRUD Package (Teams)
**What:** Exact replication of the products package pattern
**When to use:** Teams CRUD (list, get, create, update, delete)
**Example:**
```go
// cmd/teams/teams.go
package teams

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/revenium/revenium-cli/cmd"
    "github.com/revenium/revenium-cli/internal/output"
)

var Cmd = &cobra.Command{
    Use:   "teams",
    Short: "Manage teams",
    Example: `  # List all teams
  revenium teams list

  # Get a specific team
  revenium teams get team-123`,
}

func init() {
    Cmd.AddCommand(newListCmd())
    Cmd.AddCommand(newGetCmd())
    Cmd.AddCommand(newCreateCmd())
    Cmd.AddCommand(newUpdateCmd())
    Cmd.AddCommand(newDeleteCmd())
    Cmd.AddCommand(promptCaptureCmd)
    initPromptCapture()
}

var tableDef = output.TableDef{
    Headers:      []string{"ID", "Name"},
    StatusColumn: -1,
}
```

### Pattern 2: Nested Prompt Capture Subcommand
**What:** Follows the `models/pricing.go` init pattern but with get/set instead of CRUD
**When to use:** `revenium teams prompt-capture get <team-id>` and `set <team-id>`
**Example:**
```go
// cmd/teams/prompt_capture.go
package teams

import "github.com/spf13/cobra"

var promptCaptureCmd = &cobra.Command{
    Use:   "prompt-capture",
    Short: "Manage prompt capture settings for a team",
    Example: `  # View prompt capture settings
  revenium teams prompt-capture get team-123

  # Update prompt capture settings
  revenium teams prompt-capture set team-123 --enabled true`,
}

func initPromptCapture() {
    promptCaptureCmd.AddCommand(newPromptCaptureGetCmd())
    promptCaptureCmd.AddCommand(newPromptCaptureSetCmd())
}
```

### Pattern 3: Prompt Capture Get (Single-Object Display)
**What:** GET endpoint returns a settings object, not an array. Render as key-value table.
**When to use:** `revenium teams prompt-capture get <team-id>`
**Example:**
```go
// cmd/teams/prompt_capture_get.go
func newPromptCaptureGetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "get <team-id>",
        Short: "View prompt capture settings for a team",
        Args:  cobra.ExactArgs(1),
        RunE: func(c *cobra.Command, args []string) error {
            teamID := args[0]
            path := fmt.Sprintf("/v2/api/teams/%s/settings/prompts", teamID)
            var settings map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &settings); err != nil {
                return err
            }
            return renderPromptSettings(settings)
        },
    }
}
```

### Pattern 4: Prompt Capture Set (PUT with Flags)
**What:** PUT endpoint accepts settings fields as flags
**When to use:** `revenium teams prompt-capture set <team-id> --flag value`
**Example:**
```go
// cmd/teams/prompt_capture_set.go
func newPromptCaptureSetCmd() *cobra.Command {
    // Flags defined based on API fields discovered at runtime
    // Use Flags().Changed() pattern for partial updates
    // PUT to /v2/api/teams/{id}/settings/prompts
    // Render updated settings after success
}
```

### Anti-Patterns to Avoid
- **Shared str() function across packages:** Each package defines its own `str()` helper (unexported). Do NOT try to extract to a shared package -- this is intentional per existing codebase convention.
- **Struct types for API responses:** Always use `map[string]interface{}` -- the project deliberately avoids schema coupling.
- **Extra success messages after create/update:** Render the result only, no "Created successfully" prefix.

## API Endpoints

### Teams
| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| List | GET | `/v2/api/teams` | Returns array of team objects |
| Get | GET | `/v2/api/teams/{id}` | Returns single team object |
| Create | POST | `/v2/api/teams` | Body: name, description (at minimum) |
| Update | PUT | `/v2/api/teams/{id}` | Body: name, description, logo, settings |
| Delete | DELETE | `/v2/api/teams/{id}` | No body |
| Get Prompt Settings | GET | `/v2/api/teams/{id}/settings/prompts` | Returns settings object |
| Set Prompt Settings | PUT | `/v2/api/teams/{id}/settings/prompts` | Body: settings fields (systemMaxPromptLength is read-only) |

### Users
| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| List | GET | `/v2/api/users` | Returns array of user objects |
| Get | GET | `/v2/api/users/{id}` | Returns single user object |
| Create | POST | `/v2/api/users` | Body: email, firstName, lastName, roles, teamIds (required); phoneNumber, canViewPromptData (optional) |
| Update | PUT | `/v2/api/users/{id}` | Body: email, firstName, lastName, roles, teamIds (required per API); other fields optional |
| Delete | DELETE | `/v2/api/users/{id}` | No body |

### Team Object Fields (from API docs)
| Field | Type | Notes |
|-------|------|-------|
| id | string | Read-only, hashid |
| resourceType | string | Read-only, "team" |
| label | string | Read-only, team's name |
| created | string | Read-only, ISO 8601 |
| updated | string | Read-only, ISO 8601 |
| logo | string | Read-only in GET, writable in update |
| name | string | Writable (create/update) |
| description | string | Writable (create/update) |

### User Object Fields (from API docs)
| Field | Type | Notes |
|-------|------|-------|
| id | string | Read-only |
| resourceType | string | Read-only, "user" |
| label | string | Read-only, auto-set to email |
| created | string | Read-only |
| updated | string | Read-only |
| email | string | Required for create/update |
| firstName | string | Required for create/update |
| lastName | string | Required for create/update |
| phoneNumber | string | Optional |
| roles | array(string) | Required for create/update (e.g., ROLE_API_CONSUMER) |
| teamIds | array(string) | Required for create/update |
| teams | array(object) | Read-only in response |
| tenantId | string | Read-only |
| primaryUser | boolean | Optional |
| subscriberId | string | Read-only |
| canViewPromptData | boolean | Optional |
| homepagePreference | string | Optional |

### Recommended Table Columns

**Teams list/get table:**
| Column | Field | Rationale |
|--------|-------|-----------|
| ID | id | Standard identifier |
| Name | label (or name) | Primary display field; `label` is read-only and set to team name |

Note: Teams have minimal visible fields from the API. The list table is intentionally simple.

**Users list/get table:**
| Column | Field | Rationale |
|--------|-------|-----------|
| ID | id | Standard identifier |
| Email | email | Primary identifier for users |
| Name | firstName + " " + lastName | Human-readable name (compose like subscribers) |
| Roles | roles (joined) | Important for understanding user access |

### Recommended Create/Update Flags

**Teams create:** `--name` (required)
**Teams update:** `--name`, `--description` (at least one required via len(body)==0 check)

**Users create:** `--email` (required), `--first-name` (required), `--last-name` (required), `--roles` (required, string slice), `--team-ids` (required, string slice), `--phone-number` (optional), `--can-view-prompt-data` (optional bool)
**Users update:** Same flags as create, but none individually required; at least one must be changed

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Delete confirmation | Custom prompt | `resource.ConfirmDelete()` | Handles --yes, JSON mode, non-TTY |
| Table rendering | Custom formatting | `output.TableDef` + `cmd.Output.Render()` | Consistent styling, JSON/table toggle |
| API calls | Custom HTTP client | `cmd.APIClient.Do()` | Auth headers, error mapping, timeouts |
| Flag-based partial updates | Custom diff logic | `cmd.Flags().Changed()` | Cobra built-in, used everywhere |

## Common Pitfalls

### Pitfall 1: Prompt Capture Settings Field Discovery
**What goes wrong:** The prompt settings object fields are not fully documented in the API reference. The `systemMaxPromptLength` field is confirmed read-only.
**Why it happens:** Settings endpoints often have dynamic or tenant-specific fields.
**How to avoid:** Use `map[string]interface{}` for the settings response. For the `set` command, accept flags for known fields and let the API validate. Render all returned fields in the get output.
**Warning signs:** Hardcoded field lists that miss new settings added by the platform.

### Pitfall 2: Users Require Multiple Required Fields
**What goes wrong:** Users create/update requires email, firstName, lastName, roles, AND teamIds. Missing any causes a 422.
**Why it happens:** Unlike simpler resources (products need only name), users have a richer schema.
**How to avoid:** Mark email, first-name, last-name, roles, and team-ids as required via `MarkFlagRequired()` on the create command. For update, the API expects all required fields -- either send all changed fields or document that the user must provide all required fields.
**Warning signs:** 422 errors on create when required fields are omitted.

### Pitfall 3: Roles and TeamIds Are String Slices
**What goes wrong:** Using `StringVar` instead of `StringSliceVar` for roles and team-ids flags.
**Why it happens:** Most existing resources use simple string flags.
**How to avoid:** Use `StringSliceVar` for `--roles` and `--team-ids` flags. Cobra supports `--roles ROLE_API_CONSUMER,ROLE_ADMIN` or repeated `--roles ROLE_API_CONSUMER --roles ROLE_ADMIN`.
**Warning signs:** API receives string instead of array, causing type errors.

### Pitfall 4: Team Name vs Label
**What goes wrong:** Confusing `name` (writable) and `label` (read-only, auto-set to name).
**Why it happens:** API returns `label` as the display name, but `name` is the field you send in create/update.
**How to avoid:** Use `label` for display in tables (it's always populated in GET responses), use `name` in POST/PUT request bodies. If `label` is empty in some responses, fall back to `name`.
**Warning signs:** Empty name column in tables because you used `name` instead of `label`.

### Pitfall 5: Delete Test Needs --yes Flag Registration
**What goes wrong:** Tests for delete fail because `--yes` flag is inherited from root cmd at runtime but not available in isolated test.
**Why it happens:** Global flags like `--yes` are on the root command, not individual subcommands.
**How to avoid:** In delete tests, register `--yes` flag on the test command: `c.Flags().Bool("yes", false, "Skip confirmation prompts")` -- exactly as done in existing delete tests.
**Warning signs:** Flag access error in delete tests.

## Code Examples

### Registration in main.go
```go
// Source: existing main.go pattern
import (
    "github.com/revenium/revenium-cli/cmd/teams"
    "github.com/revenium/revenium-cli/cmd/users"
)

func init() {
    // ... existing registrations ...
    cmd.RegisterCommand(teams.Cmd, "resources")
    cmd.RegisterCommand(users.Cmd, "resources")
}
```

### Users Create with Required Slice Flags
```go
// Source: pattern from existing create commands + API docs
func newCreateCmd() *cobra.Command {
    var (
        email, firstName, lastName, phoneNumber string
        roles, teamIDs                          []string
        canViewPromptData                       bool
    )

    c := &cobra.Command{
        Use:   "create",
        Short: "Create a new user",
        RunE: func(c *cobra.Command, args []string) error {
            body := map[string]interface{}{
                "email":     email,
                "firstName": firstName,
                "lastName":  lastName,
                "roles":     roles,
                "teamIds":   teamIDs,
            }
            if c.Flags().Changed("phone-number") {
                body["phoneNumber"] = phoneNumber
            }
            if c.Flags().Changed("can-view-prompt-data") {
                body["canViewPromptData"] = canViewPromptData
            }

            var result map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/users", body, &result); err != nil {
                return err
            }
            return renderUser(result)
        },
    }

    c.Flags().StringVar(&email, "email", "", "User email address")
    c.Flags().StringVar(&firstName, "first-name", "", "User first name")
    c.Flags().StringVar(&lastName, "last-name", "", "User last name")
    c.Flags().StringSliceVar(&roles, "roles", nil, "User roles (e.g., ROLE_API_CONSUMER)")
    c.Flags().StringSliceVar(&teamIDs, "team-ids", nil, "Team IDs the user belongs to")
    c.Flags().StringVar(&phoneNumber, "phone-number", "", "Phone number")
    c.Flags().BoolVar(&canViewPromptData, "can-view-prompt-data", false, "Whether user can view prompt data")
    _ = c.MarkFlagRequired("email")
    _ = c.MarkFlagRequired("first-name")
    _ = c.MarkFlagRequired("last-name")
    _ = c.MarkFlagRequired("roles")
    _ = c.MarkFlagRequired("team-ids")

    return c
}
```

### Prompt Capture Get Rendering
```go
// Source: pattern adaptation from existing render functions
// Prompt settings is a single object, not an array -- render as key-value table
var promptCaptureTableDef = output.TableDef{
    Headers:      []string{"Setting", "Value"},
    StatusColumn: -1,
}

func renderPromptSettings(settings map[string]interface{}) error {
    var rows [][]string
    for key, val := range settings {
        if key == "_links" {
            continue // Skip HATEOAS links in table display
        }
        rows = append(rows, []string{key, fmt.Sprint(val)})
    }
    // Sort rows by key for consistent output
    sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
    return cmd.Output.Render(promptCaptureTableDef, rows, settings)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Struct-based API responses | `map[string]interface{}` | Phase 3 decision | Avoids schema coupling, used throughout |
| Shared str() helper | Per-package str() | Phase 3 decision | Avoids import dependencies |
| Huh library for prompts | bufio.Scanner | Phase 3 decision | Simpler, fewer dependencies |

No changes in approach needed for Phase 7 -- all patterns are stable.

## Open Questions

1. **Prompt capture settings fields**
   - What we know: `systemMaxPromptLength` (read-only integer) exists. API returns a settings object.
   - What's unclear: Full set of writable fields in the prompt settings object.
   - Recommendation: Implement `get` to display all returned fields. For `set`, start with discoverable boolean/string/integer flags and let the API guide what's accepted. The `map[string]interface{}` approach handles unknown fields gracefully.

2. **Team create required fields**
   - What we know: API accepts `name` and `description`. The API reference for team create body was truncated.
   - What's unclear: Whether `name` is strictly required or if the API has other required fields.
   - Recommendation: Make `--name` required for create (consistent with products pattern). Discovery during implementation will confirm.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (assert/require) |
| Config file | None needed (Go test built-in) |
| Quick run command | `go test ./cmd/teams/... ./cmd/users/... -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEAM-01 | List teams table + JSON + empty | unit | `go test ./cmd/teams/ -run TestList -count=1` | Wave 0 |
| TEAM-02 | Get team by ID | unit | `go test ./cmd/teams/ -run TestGet -count=1` | Wave 0 |
| TEAM-03 | Create team with flags | unit | `go test ./cmd/teams/ -run TestCreate -count=1` | Wave 0 |
| TEAM-04 | Update team partial fields | unit | `go test ./cmd/teams/ -run TestUpdate -count=1` | Wave 0 |
| TEAM-05 | Delete team with confirm | unit | `go test ./cmd/teams/ -run TestDelete -count=1` | Wave 0 |
| TEAM-06 | View prompt capture settings | unit | `go test ./cmd/teams/ -run TestPromptCaptureGet -count=1` | Wave 0 |
| TEAM-07 | Update prompt capture settings | unit | `go test ./cmd/teams/ -run TestPromptCaptureSet -count=1` | Wave 0 |
| USER-01 | List users table + JSON + empty | unit | `go test ./cmd/users/ -run TestList -count=1` | Wave 0 |
| USER-02 | Get user by ID | unit | `go test ./cmd/users/ -run TestGet -count=1` | Wave 0 |
| USER-03 | Create user with required flags | unit | `go test ./cmd/users/ -run TestCreate -count=1` | Wave 0 |
| USER-04 | Update user partial fields | unit | `go test ./cmd/users/ -run TestUpdate -count=1` | Wave 0 |
| USER-05 | Delete user with confirm | unit | `go test ./cmd/users/ -run TestDelete -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/teams/... ./cmd/users/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before verify

### Wave 0 Gaps
- [ ] `cmd/teams/` directory -- all files (teams.go, list.go, get.go, create.go, update.go, delete.go, prompt_capture.go, prompt_capture_get.go, prompt_capture_set.go + tests)
- [ ] `cmd/users/` directory -- all files (users.go, list.go, get.go, create.go, update.go, delete.go + tests)
- [ ] main.go registration for both teams and users

## Sources

### Primary (HIGH confidence)
- Existing codebase: `cmd/products/`, `cmd/tools/`, `cmd/models/` -- CRUD and nested subcommand patterns
- Existing codebase: `cmd/models/pricing.go` -- `initPricing()` pattern for nested commands
- Existing codebase: `internal/resource/resource.go` -- `ConfirmDelete` helper
- Existing codebase: `main.go` -- `RegisterCommand` pattern

### Secondary (MEDIUM confidence)
- [Revenium API Reference](https://revenium.readme.io/reference/getting-started-with-your-api) -- Teams and Users endpoint paths, methods, and field schemas
- [Revenium API: Create User](https://revenium.readme.io/reference/create_user) -- User required fields (email, firstName, lastName, roles, teamIds)
- [Revenium API: Update Team](https://revenium.readme.io/reference/update_team) -- Team update fields (name, description, logo, settings)
- [Revenium API: Prompt Settings](https://revenium.readme.io/reference/get_team_prompt_settings) -- GET/PUT `/v2/api/teams/{id}/settings/prompts`

### Tertiary (LOW confidence)
- Team create body fields -- API docs were truncated; assumed `name` required based on pattern consistency
- Prompt settings writable fields -- only `systemMaxPromptLength` (read-only) confirmed; other fields need runtime discovery

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- exact replication of existing patterns, no new libraries
- Architecture: HIGH -- follows established package structure and naming conventions
- API endpoints: MEDIUM -- confirmed from API reference docs, some field details truncated
- Pitfalls: HIGH -- based on concrete codebase analysis and API documentation

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable patterns, unlikely to change)
