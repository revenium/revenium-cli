# Phase 5: Subscribers & Subscriptions - Research

**Researched:** 2026-03-12
**Domain:** Go CLI CRUD commands for Revenium subscriber and subscription resources
**Confidence:** HIGH

## Summary

Phase 5 adds two new resource command groups (`subscribers` and `subscriptions`) following the exact same CRUD patterns established in Phases 3 (Sources) and 4 (AI Models). The codebase already has a mature, repeatable pattern: package-per-resource under `cmd/`, exported `Cmd` variable, `init()` wiring subcommands, `RegisterCommand()` in `main.go`, `map[string]interface{}` response handling, `str()` helper, `tableDef`/`toRows`/`renderX` trio, and `httptest`-based unit tests.

The unique aspect of this phase is that subscriptions support both PUT (full update) and PATCH (partial update). The PATCH pattern already exists in `cmd/models/update.go`. The subscriber resource is straightforward standard CRUD. API endpoints follow the established `/v2/api/` prefix pattern: `/v2/api/subscribers` and `/v2/api/subscriptions`.

**Primary recommendation:** Clone the `cmd/sources/` package structure for both resources. For subscriptions, the update command should use PUT by default and accept a `--patch` flag to switch to PATCH, reusing the `Flags().Changed()` partial-body pattern from models.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions
- Package per resource: `cmd/subscribers/` and `cmd/subscriptions/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list -> "No subscribers found." / "No subscriptions found."

### Claude's Discretion
- Table columns for subscribers and subscriptions (discover from API)
- Which flags for create/update on each resource
- How to handle SUBR-05 (PATCH partial update) -- could be a `--patch` flag on update, or always use PATCH, or separate `patch` subcommand
- Whether subscriptions show related subscriber/source names or just IDs
- Any shared infrastructure between the two resources (if any)
- API endpoint paths for both resources

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SUBS-01 | User can list all subscribers | GET `/v2/api/subscribers`, table with ID/Name/Email/Status columns |
| SUBS-02 | User can get a subscriber by ID | GET `/v2/api/subscribers/{id}`, single-row render |
| SUBS-03 | User can create a new subscriber | POST `/v2/api/subscribers`, flags for email/firstName/lastName |
| SUBS-04 | User can update a subscriber | PUT `/v2/api/subscribers/{id}`, `Flags().Changed()` pattern |
| SUBS-05 | User can delete a subscriber with confirmation | DELETE `/v2/api/subscribers/{id}`, `ConfirmDelete()` helper |
| SUBR-01 | User can list all subscriptions | GET `/v2/api/subscriptions`, table with ID/Description/Status columns |
| SUBR-02 | User can get a subscription by ID | GET `/v2/api/subscriptions/{id}`, single-row render |
| SUBR-03 | User can create a new subscription | POST `/v2/api/subscriptions`, flags for description/subscriber/product |
| SUBR-04 | User can update a subscription | PUT `/v2/api/subscriptions/{id}`, full update |
| SUBR-05 | User can partially update a subscription (PATCH) | PATCH `/v2/api/subscriptions/{id}`, `--patch` flag on update command |
| SUBR-06 | User can delete a subscription with confirmation | DELETE `/v2/api/subscriptions/{id}`, `ConfirmDelete()` helper |

</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| cobra | (existing) | Command tree | Already in project, used by all commands |
| testify | (existing) | Test assertions | Already used for assert/require |
| httptest | stdlib | Test HTTP servers | Established test pattern in project |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| internal/output | (project) | Table/JSON rendering | All list/get/create/update commands |
| internal/resource | (project) | ConfirmDelete helper | Both delete commands |
| internal/api | (project) | HTTP client with Do() | All API calls |

No new dependencies needed. This phase uses only existing project infrastructure.

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  subscribers/
    subscribers.go      # Cmd, tableDef, toRows, str, renderSubscriber
    list.go             # newListCmd()
    get.go              # newGetCmd()
    create.go           # newCreateCmd()
    update.go           # newUpdateCmd()
    delete.go           # newDeleteCmd()
    list_test.go
    get_test.go
    create_test.go
    update_test.go
    delete_test.go
  subscriptions/
    subscriptions.go    # Cmd, tableDef, toRows, str, renderSubscription
    list.go             # newListCmd()
    get.go              # newGetCmd()
    create.go           # newCreateCmd()
    update.go           # newUpdateCmd() -- handles both PUT and PATCH
    delete.go           # newDeleteCmd()
    list_test.go
    get_test.go
    create_test.go
    update_test.go
    delete_test.go
```

### Pattern 1: Standard Resource Package (clone from sources)
**What:** Each resource package follows identical structure -- exported `Cmd` var, `init()` adds subcommands, `tableDef`/`toRows`/`str`/`renderX` helpers.
**When to use:** Every new CRUD resource.
**Example:**
```go
// Source: cmd/sources/sources.go (existing project code)
package subscribers

var Cmd = &cobra.Command{
    Use:   "subscribers",
    Short: "Manage subscribers",
    Example: `  # List all subscribers
  revenium subscribers list

  # Get a specific subscriber
  revenium subscribers get abc-123`,
}

func init() {
    Cmd.AddCommand(newListCmd())
    Cmd.AddCommand(newGetCmd())
    Cmd.AddCommand(newCreateCmd())
    Cmd.AddCommand(newUpdateCmd())
    Cmd.AddCommand(newDeleteCmd())
}

var tableDef = output.TableDef{
    Headers:      []string{"ID", "Name", "Email", "Status"},
    StatusColumn: 3,
}
```

### Pattern 2: PUT + PATCH Update (subscription-specific)
**What:** Single `update` command uses PUT by default, `--patch` flag switches to PATCH method. When `--patch` is used, only `Flags().Changed()` fields go in the body.
**When to use:** Subscriptions update (SUBR-04 + SUBR-05).
**Example:**
```go
// Recommended approach for subscription update
func newUpdateCmd() *cobra.Command {
    var (
        patch       bool
        description string
        // ... other fields
    )

    c := &cobra.Command{
        Use:   "update <id>",
        Short: "Update a subscription",
        Args:  cobra.ExactArgs(1),
        Example: `  # Full update
  revenium subscriptions update sub-123 --description "Updated"

  # Partial update (PATCH)
  revenium subscriptions update sub-123 --patch --description "Only this field"`,
        RunE: func(c *cobra.Command, args []string) error {
            id := args[0]
            body := make(map[string]interface{})

            if c.Flags().Changed("description") {
                body["description"] = description
            }
            // ... other Changed() checks

            if len(body) == 0 {
                return fmt.Errorf("no fields specified to update")
            }

            method := "PUT"
            if patch {
                method = "PATCH"
            }

            var result map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), method, "/v2/api/subscriptions/"+id, body, &result); err != nil {
                return err
            }
            return renderSubscription(result)
        },
    }

    c.Flags().BoolVar(&patch, "patch", false, "Use partial update (PATCH) instead of full update (PUT)")
    // ... field flags
    return c
}
```

### Pattern 3: Registration in main.go
**What:** Add two `RegisterCommand` calls in `main.go init()`.
**Example:**
```go
// In main.go init():
import "github.com/revenium/revenium-cli/cmd/subscribers"
import "github.com/revenium/revenium-cli/cmd/subscriptions"

cmd.RegisterCommand(subscribers.Cmd, "resources")
cmd.RegisterCommand(subscriptions.Cmd, "resources")
```

### Anti-Patterns to Avoid
- **Nesting subscriptions under subscribers:** These are independent command groups per user decision. `revenium subscriptions list`, not `revenium subscribers subscriptions list`.
- **Separate `patch` subcommand:** Adds command-tree complexity. A `--patch` flag on `update` is simpler and consistent with how models handle PATCH.
- **Shared str() across packages:** Each package has its own `str()` function (private). This is the established convention -- do not extract to a shared package.

## API Endpoints

### Subscribers

| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| List | GET | `/v2/api/subscribers` | Returns array of subscriber objects |
| Get | GET | `/v2/api/subscribers/{id}` | Single subscriber |
| Create | POST | `/v2/api/subscribers` | Body: subscriber fields |
| Update | PUT | `/v2/api/subscribers/{id}` | Full update |
| Delete | DELETE | `/v2/api/subscribers/{id}` | No response body expected |

### Subscriptions

| Operation | Method | Path | Notes |
|-----------|--------|------|-------|
| List | GET | `/v2/api/subscriptions` | Returns array of subscription objects |
| Get | GET | `/v2/api/subscriptions/{id}` | Single subscription |
| Create | POST | `/v2/api/subscriptions` | Body: subscription fields |
| Update (full) | PUT | `/v2/api/subscriptions/{id}` | Full update |
| Update (partial) | PATCH | `/v2/api/subscriptions/{id}` | Partial update, only changed fields |
| Delete | DELETE | `/v2/api/subscriptions/{id}` | No response body expected |

**Confidence:** HIGH -- Endpoints confirmed from Revenium API reference docs and follow the `/v2/api/` pattern established in existing code.

### Subscriber Fields (from API docs)

| Field | Type | Writable | Table Column |
|-------|------|----------|-------------|
| `id` | string | read-only | Yes |
| `email` | string | yes | Yes |
| `firstName` | string | yes | Yes (as "Name" combined) |
| `lastName` | string | yes | Yes (as "Name" combined) |
| `phoneNumber` | string | yes | No |
| `teamIds` | []string | yes | No |
| `primaryUser` | boolean | yes | No |
| `roles` | []string | yes | No |
| `canViewPromptData` | boolean | yes | No |
| `subscriberId` | string | read-only | No |
| `created` | datetime | read-only | No |
| `updated` | datetime | read-only | No |
| `tenantId` | string | read-only | No |

**Recommended table columns for subscribers:** ID, Name (firstName + lastName combined), Email, Primary User (or just first three columns for simplicity). The `str()` approach can combine firstName+lastName into a display name.

**Confidence:** MEDIUM -- Fields from API reference page; exact writability needs runtime validation.

### Subscription Fields (from API docs)

The full subscription schema was not fully exposed in the API reference docs. Based on the documentation:

| Field | Type | Writable | Table Column |
|-------|------|----------|-------------|
| `id` | string | read-only | Yes |
| `label` | string | read-only | Yes (as "Name") |
| `description` | string | yes | Yes |
| `created` | datetime | read-only | No |
| `updated` | datetime | read-only | No |

**Note:** Subscription objects likely contain additional fields (subscriber reference, product/source reference, status, quota info, expiration) that are not fully documented in the public API reference. The `map[string]interface{}` approach handles this gracefully -- discover fields at runtime and pick sensible table columns.

**Recommended table columns for subscriptions:** ID, Label/Description, Status (if present). Show IDs for related entities rather than names (avoids extra API calls).

**Confidence:** LOW for full field list -- API reference was truncated. The `map[string]interface{}` approach makes this a non-blocker.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Delete confirmation | Custom prompt logic | `resource.ConfirmDelete()` | Handles --yes, JSON mode, non-TTY cases |
| Table rendering | Manual column formatting | `output.TableDef` + `Output.Render()` | Handles table/JSON dual mode |
| HTTP calls | Raw http.Client usage | `cmd.APIClient.Do()` | Auth, error handling, verbose mode |
| Flag change detection | Manual tracking | `c.Flags().Changed("flag")` | Cobra built-in, already used everywhere |

## Common Pitfalls

### Pitfall 1: Forgetting to Register --yes Flag in Tests
**What goes wrong:** Delete tests fail because `--yes` is a persistent flag on rootCmd, not on the delete command itself. In test isolation, the flag doesn't exist.
**Why it happens:** Tests create standalone commands without the root command tree.
**How to avoid:** In delete tests, register the `--yes` flag locally: `c.Flags().Bool("yes", false, "Skip confirmation prompts")` -- exactly as done in `cmd/sources/delete_test.go`.
**Warning signs:** Test panics on "unknown flag: --yes".

### Pitfall 2: Subscription Update Missing Method Switch
**What goes wrong:** Both PUT and PATCH go to the same endpoint but with different semantics. Using the wrong method could overwrite fields with zero values (PUT) or fail to replace nested objects (PATCH).
**Why it happens:** Confusing partial body construction with HTTP method semantics.
**How to avoid:** PUT sends a complete body; PATCH sends only changed fields. The `--patch` flag switches the HTTP method. The `Flags().Changed()` check applies to both modes for building the body.
**Warning signs:** Fields getting reset to empty/default on update.

### Pitfall 3: Package Name Collision
**What goes wrong:** Using `subscription` (singular) as package name when the command is `subscriptions` (plural).
**Why it happens:** Go convention favors singular package names, but CLI convention uses plural.
**How to avoid:** Use plural package names (`subscribers`, `subscriptions`) matching the command names. This is the established convention in the project (`sources`, `models`).

### Pitfall 4: Empty Body on Update
**What goes wrong:** User runs `revenium subscribers update abc-123` without any flags.
**Why it happens:** No flags changed means empty body map.
**How to avoid:** Check `len(body) == 0` and return an error, exactly as sources/update.go does.

## Code Examples

### Subscriber List Command (direct clone of sources pattern)
```go
// Source: adapted from cmd/sources/list.go
func newListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all subscribers",
        Args:  cobra.NoArgs,
        Example: `  # List all subscribers
  revenium subscribers list

  # List subscribers as JSON
  revenium subscribers list --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var subscribers []map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/subscribers", nil, &subscribers); err != nil {
                return err
            }
            if len(subscribers) == 0 {
                if cmd.Output.IsJSON() {
                    return cmd.Output.RenderJSON([]interface{}{})
                }
                fmt.Fprintln(c.OutOrStdout(), "No subscribers found.")
                return nil
            }
            return cmd.Output.Render(tableDef, toRows(subscribers), subscribers)
        },
    }
}
```

### Subscriber Name Display (combining firstName + lastName)
```go
func toRows(subscribers []map[string]interface{}) [][]string {
    rows := make([][]string, len(subscribers))
    for i, s := range subscribers {
        name := strings.TrimSpace(str(s, "firstName") + " " + str(s, "lastName"))
        rows[i] = []string{
            str(s, "id"),
            name,
            str(s, "email"),
        }
    }
    return rows
}
```

### Subscriber Create Flags
```go
func newCreateCmd() *cobra.Command {
    var email, firstName, lastName string

    c := &cobra.Command{
        Use:   "create",
        Short: "Create a new subscriber",
        Example: `  # Create a subscriber
  revenium subscribers create --email user@example.com --first-name John --last-name Doe`,
        RunE: func(c *cobra.Command, args []string) error {
            body := map[string]interface{}{
                "email": email,
            }
            if c.Flags().Changed("first-name") {
                body["firstName"] = firstName
            }
            if c.Flags().Changed("last-name") {
                body["lastName"] = lastName
            }
            var result map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/subscribers", body, &result); err != nil {
                return err
            }
            return renderSubscriber(result)
        },
    }
    c.Flags().StringVar(&email, "email", "", "Subscriber email address")
    c.Flags().StringVar(&firstName, "first-name", "", "Subscriber first name")
    c.Flags().StringVar(&lastName, "last-name", "", "Subscriber last name")
    _ = c.MarkFlagRequired("email")
    return c
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Typed structs for API responses | `map[string]interface{}` | Phase 3 decision | No schema coupling; fields discovered at runtime |
| Interface-based DI for testing | Direct global var swap + httptest | Phase 1 decision | Simpler test setup |
| Huh library for prompts | bufio.Scanner for ConfirmDelete | Phase 3 decision | Minimal dependency for simple y/N prompt |

## Open Questions

1. **Subscriber create required fields**
   - What we know: `email` is almost certainly required. `firstName` and `lastName` are likely optional.
   - What's unclear: Exact required vs optional set for create. API docs were truncated.
   - Recommendation: Make `--email` required via Cobra. Make `--first-name` and `--last-name` optional. If the API rejects, the error message will guide the user.

2. **Subscription create required fields**
   - What we know: Docs say "product must have at least one data source configured." Fields likely include subscriber reference, product/source reference, and description.
   - What's unclear: Exact field names and which are required.
   - Recommendation: Start with `--description` and discover other fields from API error responses or runtime testing. The `map[string]interface{}` approach makes this low-risk.

3. **Subscription table columns**
   - What we know: id, label, description, created, updated exist. Status field likely exists.
   - What's unclear: Full field list for subscriptions.
   - Recommendation: Start with ID, Label, and add more columns as discovered. Can be adjusted without breaking changes.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (assert/require) |
| Config file | None needed (Go standard) |
| Quick run command | `go test ./cmd/subscribers/... ./cmd/subscriptions/...` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SUBS-01 | List subscribers table + JSON + empty | unit | `go test ./cmd/subscribers/ -run TestList -x` | Wave 0 |
| SUBS-02 | Get subscriber by ID | unit | `go test ./cmd/subscribers/ -run TestGet -x` | Wave 0 |
| SUBS-03 | Create subscriber with required/optional fields | unit | `go test ./cmd/subscribers/ -run TestCreate -x` | Wave 0 |
| SUBS-04 | Update subscriber with Changed() fields | unit | `go test ./cmd/subscribers/ -run TestUpdate -x` | Wave 0 |
| SUBS-05 | Delete subscriber with confirmation | unit | `go test ./cmd/subscribers/ -run TestDelete -x` | Wave 0 |
| SUBR-01 | List subscriptions table + JSON + empty | unit | `go test ./cmd/subscriptions/ -run TestList -x` | Wave 0 |
| SUBR-02 | Get subscription by ID | unit | `go test ./cmd/subscriptions/ -run TestGet -x` | Wave 0 |
| SUBR-03 | Create subscription | unit | `go test ./cmd/subscriptions/ -run TestCreate -x` | Wave 0 |
| SUBR-04 | Update subscription (PUT) | unit | `go test ./cmd/subscriptions/ -run TestUpdate -x` | Wave 0 |
| SUBR-05 | Partial update subscription (PATCH via --patch) | unit | `go test ./cmd/subscriptions/ -run TestUpdatePatch -x` | Wave 0 |
| SUBR-06 | Delete subscription with confirmation | unit | `go test ./cmd/subscriptions/ -run TestDelete -x` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/subscribers/... ./cmd/subscriptions/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/subscribers/` directory -- all files are new
- [ ] `cmd/subscriptions/` directory -- all files are new
- No framework install needed (Go testing is standard library)

## Sources

### Primary (HIGH confidence)
- Existing project code: `cmd/sources/` -- full CRUD template (list, get, create, update, delete + tests)
- Existing project code: `cmd/models/update.go` -- PATCH update pattern
- Existing project code: `internal/resource/resource.go` -- ConfirmDelete helper
- Existing project code: `cmd/root.go` -- RegisterCommand, global flags
- Existing project code: `main.go` -- registration pattern

### Secondary (MEDIUM confidence)
- [Revenium API Reference](https://revenium.readme.io/reference) -- Subscriber and subscription endpoint paths confirmed as `/v2/api/subscribers` and `/v2/api/subscriptions`
- [Revenium Subscriber Docs](https://docs.revenium.io/customer-and-subscriber-management/subscribers) -- Subscriber field names (email, firstName, lastName, etc.)
- [Revenium Subscription Docs](https://docs.revenium.io/ai-and-api-monetization/manage/product-keys) -- Subscription field concepts

### Tertiary (LOW confidence)
- Subscription full field list -- API reference was truncated; exact writable fields need runtime discovery

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- No new dependencies, fully established patterns
- Architecture: HIGH -- Direct clone of existing sources/models patterns
- API Endpoints: HIGH -- Confirmed from Revenium API reference
- Subscriber fields: MEDIUM -- Confirmed from API docs but writability needs validation
- Subscription fields: LOW -- Truncated in API docs, need runtime discovery
- Pitfalls: HIGH -- Based on direct observation of existing code patterns

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable patterns, unlikely to change)
