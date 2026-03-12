# Phase 3: First Resource (Sources) - Research

**Researched:** 2026-03-12
**Domain:** Go CLI CRUD commands for REST API resources (Cobra + Lip Gloss)
**Confidence:** HIGH

## Summary

Phase 3 implements the first full CRUD resource (Sources) and establishes the reusable pattern for 6+ subsequent resource phases. The existing codebase provides a solid foundation: `api.Client.Do()` handles all HTTP, `output.Formatter.Render()` dispatches to table or JSON, and `cmd/config/` demonstrates the subcommand package pattern. The work breaks down into: (1) shared CRUD helpers (`internal/resource/`) for confirmation prompts, result rendering, and pagination, (2) the `cmd/sources/` package with list/get/create/update/delete commands, and (3) adding `--yes`/`-y` as a global persistent flag on rootCmd.

The Revenium API uses REST endpoints at `/v2/api/sources/{id}` for GET/PUT/DELETE and likely `/v2/api/sources` for GET (list) and POST (create). The SourceResource_Read schema includes fields like id, name, type, sourceType, description, status, plus nested objects (owner, team, environment). The SourceResource_Write schema is used for create/update payloads. The delete response includes a message field confirming deletion with the resource ID.

**Primary recommendation:** Build shared helpers in `internal/resource/` first, then implement `cmd/sources/` using those helpers. This investment pays off immediately in Phases 4-9.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- `sources list` shows essential columns only: ID, Name, Type, Status -- compact and scannable
- `sources get <id>` shows same columns as list, rendered as a single-row table (Phase 2 decision)
- Status column uses Phase 2 color palette (green=active, red=inactive, yellow=pending) via existing `statusStyle()`
- Empty list displays "No sources found." -- no empty table rendered
- Users use `--json` to see all fields when needed
- Required vs optional flags discovered from the OpenAPI spec during research/planning
- Long flags only for resource fields (`--name`, `--type`, `--description`) -- short flags reserved for global options (-v, -q, -y)
- `sources update <id>` uses partial update semantics -- only sends fields the user explicitly passes
- After successful create/update: render the result as a single-row table (or JSON with --json). No extra "Created successfully" message.
- Delete prompt shows ID: "Delete source abc-123? [y/N]" -- default is No
- `--yes` / `-y` is a global persistent flag on root command -- works across all resources
- `--json` mode implies `--yes` -- scripts shouldn't be blocked by prompts
- Non-TTY (piped input) should also imply `--yes` or fail safely
- After successful delete: "Deleted source abc-123." -- one line confirmation
- Package per resource: `cmd/sources/` with list.go, get.go, create.go, update.go, delete.go
- Shared resource helpers: common functions like `ConfirmDelete()`, `RenderResult()`
- Auto-fetch all pages transparently if API returns paginated results
- 2-3 help examples per command
- Sources go under "Core Resources" group (GroupID: "resources")

### Claude's Discretion
- Shared helper package location and API (internal/resource or similar)
- How to detect which flags were explicitly set (Cobra `cmd.Flags().Changed()`)
- Confirmation prompt implementation (fmt.Scan, bufio, or Huh library)
- Pagination detection and auto-fetch implementation
- Exact command registration pattern in root.go

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SRCS-01 | User can list all sources with styled table output | List command using GET /v2/api/sources, TableDef with ID/Name/Type/Status headers, statusStyle for Status column |
| SRCS-02 | User can get a source by ID with detailed view | Get command using GET /v2/api/sources/{id}, single-row table rendering via Formatter.Render() |
| SRCS-03 | User can create a new source | Create command using POST /v2/api/sources with SourceResource_Write payload, flag-based input |
| SRCS-04 | User can update an existing source | Update command using PUT /v2/api/sources/{id} with partial update semantics via cmd.Flags().Changed() |
| SRCS-05 | User can delete a source with confirmation prompt | Delete command using DELETE /v2/api/sources/{id}, ConfirmDelete helper with --yes/--json/non-TTY bypass |
</phase_requirements>

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| cobra | v1.10.2 | CLI framework and command tree | Already used; provides flag parsing, arg validation, help generation |
| lipgloss/v2 | v2.0.2 | Table rendering and styled output | Already used for TableDef, RenderTable, statusStyle |
| charmbracelet/x/term | v0.2.2 | TTY detection | Already used in output.Formatter for isTTY check |

### Supporting (already in project)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testify | v1.11.1 | Test assertions | All test files use assert/require |
| net/http/httptest | stdlib | HTTP test servers | API client integration tests |

### No New Dependencies Needed
| Problem | Solution | Why |
|---------|----------|-----|
| Confirmation prompts | `bufio.NewScanner(os.Stdin)` + `fmt.Fprintf(os.Stderr)` | Simple y/N prompt does not justify adding the Huh library. Two functions at most. |
| TTY detection for prompt bypass | `term.IsTerminal(os.Stdin.Fd())` | Already have charmbracelet/x/term in go.mod |
| Flag change detection | `cmd.Flags().Changed("name")` | Built into Cobra/pflag, no library needed |

## Architecture Patterns

### Recommended Project Structure
```
cmd/
├── sources/
│   ├── sources.go        # Parent command + subcommand registration
│   ├── list.go           # revenium sources list
│   ├── get.go            # revenium sources get <id>
│   ├── create.go         # revenium sources create --name ... --type ...
│   ├── update.go         # revenium sources update <id> --name ...
│   └── delete.go         # revenium sources delete <id>
├── root.go               # Add --yes persistent flag + sources command registration
└── ...
internal/
├── resource/
│   └── resource.go       # ConfirmDelete(), shared helpers
└── ...
```

### Pattern 1: Command Package Registration (follow cmd/config pattern)
**What:** Each resource package exports a `Cmd` variable (the parent cobra.Command) and registers subcommands in `init()`.
**When to use:** Every resource package.
**Example:**
```go
// cmd/sources/sources.go
package sources

import "github.com/spf13/cobra"

// Cmd is the parent sources command, exported for registration in root.go.
var Cmd = &cobra.Command{
    Use:   "sources",
    Short: "Manage sources",
    Example: `  # List all sources
  revenium sources list

  # Get a specific source
  revenium sources get abc-123`,
}

func init() {
    Cmd.AddCommand(newListCmd())
    Cmd.AddCommand(newGetCmd())
    Cmd.AddCommand(newCreateCmd())
    Cmd.AddCommand(newUpdateCmd())
    Cmd.AddCommand(newDeleteCmd())
}
```

```go
// In cmd/root.go init():
sourcesCmd := sources.Cmd
sourcesCmd.GroupID = "resources"
rootCmd.AddCommand(sourcesCmd)
```

### Pattern 2: Command RunE with API Client Access via Package Var
**What:** Commands access the shared `cmd.APIClient` and `cmd.Output` package variables (already established in Phase 1/2). No dependency injection needed.
**When to use:** Every verb command (list, get, create, update, delete).
**Example:**
```go
// cmd/sources/list.go
package sources

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/revenium/revenium-cli/cmd"
    "github.com/revenium/revenium-cli/internal/output"
)

func newListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all sources",
        Example: `  # List sources
  revenium sources list

  # List sources as JSON
  revenium sources list --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var sources []map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources", nil, &sources); err != nil {
                return err
            }
            if len(sources) == 0 {
                fmt.Fprintln(c.OutOrStdout(), "No sources found.")
                return nil
            }
            rows := make([][]string, len(sources))
            for i, s := range sources {
                rows[i] = []string{
                    fmt.Sprint(s["id"]),
                    fmt.Sprint(s["name"]),
                    fmt.Sprint(s["type"]),
                    fmt.Sprint(s["status"]),
                }
            }
            return cmd.Output.Render(sourcesTableDef, rows, sources)
        },
    }
}

var sourcesTableDef = output.TableDef{
    Headers:      []string{"ID", "Name", "Type", "Status"},
    StatusColumn: 3,
}
```

### Pattern 3: Partial Update with cmd.Flags().Changed()
**What:** For update commands, only include fields in the request body that the user explicitly passed as flags. Use Cobra's `Changed()` method on the flag set.
**When to use:** All update commands.
**Example:**
```go
// cmd/sources/update.go
func newUpdateCmd() *cobra.Command {
    var name, typ, description string
    c := &cobra.Command{
        Use:   "update <id>",
        Short: "Update a source",
        Args:  cobra.ExactArgs(1),
        RunE: func(c *cobra.Command, args []string) error {
            id := args[0]
            body := make(map[string]interface{})
            if c.Flags().Changed("name") {
                body["name"] = name
            }
            if c.Flags().Changed("type") {
                body["type"] = typ
            }
            if c.Flags().Changed("description") {
                body["description"] = description
            }
            if len(body) == 0 {
                return fmt.Errorf("no fields specified to update")
            }
            var result map[string]interface{}
            err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/sources/"+id, body, &result)
            if err != nil {
                return err
            }
            return renderSource(result)
        },
    }
    c.Flags().StringVar(&name, "name", "", "Source name")
    c.Flags().StringVar(&typ, "type", "", "Source type")
    c.Flags().StringVar(&description, "description", "", "Source description")
    return c
}
```

### Pattern 4: Delete Confirmation with Shared Helper
**What:** A reusable `ConfirmDelete()` function in `internal/resource/` that handles the --yes flag, --json bypass, non-TTY bypass, and interactive prompt.
**When to use:** Every delete command across all resource phases.
**Example:**
```go
// internal/resource/resource.go
package resource

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/charmbracelet/x/term"
)

// ConfirmDelete prompts the user to confirm deletion of a resource.
// Returns true if the user confirms, or if skipConfirm is true,
// jsonMode is true, or stdin is not a TTY.
func ConfirmDelete(resourceType, id string, skipConfirm, jsonMode bool) (bool, error) {
    if skipConfirm || jsonMode {
        return true, nil
    }
    if !term.IsTerminal(os.Stdin.Fd()) {
        return true, nil
    }
    fmt.Fprintf(os.Stderr, "Delete %s %s? [y/N] ", resourceType, id)
    scanner := bufio.NewScanner(os.Stdin)
    if !scanner.Scan() {
        return false, nil
    }
    answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
    return answer == "y" || answer == "yes", nil
}
```

### Anti-Patterns to Avoid
- **Typed resource structs too early:** Use `map[string]interface{}` for API responses initially. The API returns many fields and the exact schema is partially documented. Typed structs can be introduced later when the response shapes are fully validated. JSON passthrough with `--json` works perfectly with maps.
- **HTTP logic in command files:** All API calls go through `cmd.APIClient.Do()`. Commands never construct URLs or set headers.
- **Rendering logic in commands:** Commands call `cmd.Output.Render()` for table/JSON dispatch. Table definition (`TableDef`) is a package-level var.
- **Hardcoding the confirmation prompt in every delete command:** Use the shared `ConfirmDelete()` helper.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Flag change detection | Custom flag tracking | `cmd.Flags().Changed("flag")` | Built into pflag/Cobra; tracks which flags were explicitly set vs defaults |
| TTY detection | Manual stat/ioctl calls | `term.IsTerminal(fd)` | Already in go.mod via charmbracelet/x/term |
| Table rendering | Custom column formatting | `output.Formatter.RenderTable()` | Already built in Phase 2 with border styles, status colors, width handling |
| JSON output | Custom marshal/write | `output.Formatter.RenderJSON()` | Already built in Phase 2 with pretty-printing |
| HTTP + auth | Custom request building | `api.Client.Do()` | Already built in Phase 1 with auth, error mapping, verbose logging |
| Error display | Custom error formatting | Return error from RunE; main.go handles rendering | Already built in Phase 1/2 with styled boxes and JSON error format |

**Key insight:** Phase 1 and 2 already built the infrastructure. Phase 3 commands should be thin wrappers: parse flags, call API, render result. If a command file exceeds ~60 lines, something is being hand-rolled.

## Common Pitfalls

### Pitfall 1: Circular Import Between cmd/sources and cmd
**What goes wrong:** `cmd/sources/` imports `cmd` for `cmd.APIClient` and `cmd.Output`. If `cmd/root.go` also imports `cmd/sources`, you get a circular import.
**Why it happens:** Go does not allow circular package dependencies.
**How to avoid:** `cmd/root.go` imports `cmd/sources` (to register the command). `cmd/sources/*.go` imports `cmd` (to access APIClient/Output). This is NOT circular because `cmd/sources` is a sub-package of `cmd/` in the directory tree, but they are separate Go packages. The import path is `github.com/revenium/revenium-cli/cmd` (for the root package vars) and registration happens in `cmd/root.go`'s `init()`. This mirrors the existing `cmd/config` pattern exactly.
**Warning signs:** Compiler error "import cycle not allowed."

### Pitfall 2: Forgetting to Handle Empty List Response
**What goes wrong:** Rendering an empty table (headers with no rows) looks broken.
**Why it happens:** API returns empty array, code passes it straight to RenderTable.
**How to avoid:** Check `len(items) == 0` before rendering. Print "No sources found." and return nil. This is a locked decision from CONTEXT.md.
**Warning signs:** Empty table borders with no data rows.

### Pitfall 3: Update Sending All Fields (Overwriting Unset Values)
**What goes wrong:** User runs `sources update abc --name "New Name"` but the PUT body includes all fields with zero values, overwriting existing data on the server.
**Why it happens:** Using a typed struct with all fields means unset fields send as zero values ("", 0, false).
**How to avoid:** Build the request body as `map[string]interface{}` and only include fields where `cmd.Flags().Changed()` returns true. This is a locked decision.
**Warning signs:** Update command clobbering fields the user didn't intend to change.

### Pitfall 4: Not Adding --yes Flag as Persistent on Root
**What goes wrong:** `--yes` only works on the specific delete command where it's defined, not globally.
**Why it happens:** Flag added to the delete command instead of rootCmd.
**How to avoid:** Add `--yes`/`-y` as `rootCmd.PersistentFlags().BoolVarP()` in root.go init(), alongside verbose/json/quiet. Access via `cmd.YesMode` package var or look it up from the command's inherited flags.
**Warning signs:** `--yes` flag not recognized on other resource delete commands.

### Pitfall 5: Blocking on Prompt in Non-Interactive Contexts
**What goes wrong:** Delete command hangs in CI/CD pipelines or scripts waiting for stdin input.
**Why it happens:** Prompt reads from stdin without checking if it's a terminal.
**How to avoid:** `ConfirmDelete()` must check `term.IsTerminal(os.Stdin.Fd())` and auto-confirm (or auto-skip) in non-TTY mode. Also check `--json` flag. Both are locked decisions.
**Warning signs:** CI pipeline hangs on delete commands.

### Pitfall 6: API Response Shape Mismatch
**What goes wrong:** Code expects an array from list endpoint but gets a wrapped object `{"content": [...], "page": {...}}`.
**Why it happens:** API pagination wraps results in an envelope.
**How to avoid:** Use `map[string]interface{}` initially and inspect the actual response shape. If the list endpoint returns a paginated envelope, extract the array from the content/items field. Log the raw response with `--verbose` to debug.
**Warning signs:** JSON decode error or empty table despite having data.

## Code Examples

### List Command (verified pattern from existing codebase)
```go
// cmd/sources/list.go
func newListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all sources",
        Args:  cobra.NoArgs,
        Example: `  # List all sources
  revenium sources list

  # List sources as JSON
  revenium sources list --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var sources []map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources", nil, &sources); err != nil {
                return err
            }
            if len(sources) == 0 {
                if !cmd.Output.IsJSON() {
                    fmt.Fprintln(c.OutOrStdout(), "No sources found.")
                } else {
                    cmd.Output.RenderJSON([]interface{}{})
                }
                return nil
            }
            rows := toRows(sources)
            return cmd.Output.Render(tableDef, rows, sources)
        },
    }
}
```

### Delete Command with Confirmation
```go
// cmd/sources/delete.go
func newDeleteCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "delete <id>",
        Short: "Delete a source",
        Args:  cobra.ExactArgs(1),
        Example: `  # Delete a source (with confirmation)
  revenium sources delete abc-123

  # Delete without confirmation
  revenium sources delete abc-123 --yes`,
        RunE: func(c *cobra.Command, args []string) error {
            id := args[0]
            yes, _ := c.Flags().GetBool("yes")
            ok, err := resource.ConfirmDelete("source", id, yes, cmd.Output.IsJSON())
            if err != nil {
                return err
            }
            if !ok {
                return nil
            }
            if err := cmd.APIClient.Do(c.Context(), "DELETE", "/v2/api/sources/"+id, nil, nil); err != nil {
                return err
            }
            if !cmd.Output.IsQuiet() {
                fmt.Fprintf(c.OutOrStdout(), "Deleted source %s.\n", id)
            }
            return nil
        },
    }
}
```

### Accessing the Global --yes Flag
```go
// In cmd/root.go init(), add alongside existing persistent flags:
var yesMode bool
rootCmd.PersistentFlags().BoolVarP(&yesMode, "yes", "y", false, "Skip confirmation prompts")

// Export for subcommands:
// Option A: Package variable (like jsonMode)
var YesMode bool  // set in init

// Option B: Look up from command's inherited flags in RunE:
yes, _ := c.Flags().GetBool("yes")
```

### Row Extraction Helper (reusable per resource)
```go
// cmd/sources/sources.go
var tableDef = output.TableDef{
    Headers:      []string{"ID", "Name", "Type", "Status"},
    StatusColumn: 3,
}

func toRows(sources []map[string]interface{}) [][]string {
    rows := make([][]string, len(sources))
    for i, s := range sources {
        rows[i] = []string{
            str(s, "id"),
            str(s, "name"),
            str(s, "type"),
            str(s, "status"),
        }
    }
    return rows
}

func str(m map[string]interface{}, key string) string {
    if v, ok := m[key]; ok && v != nil {
        return fmt.Sprint(v)
    }
    return ""
}
```

## API Endpoints

Based on OpenAPI spec analysis (partially documented, some inferences):

| Operation | Method | Path | Request Body | Response |
|-----------|--------|------|-------------|----------|
| List | GET | `/v2/api/sources` | none | Array of SourceResource_Read (possibly paginated envelope) |
| Get | GET | `/v2/api/sources/{id}` | none | Single SourceResource_Read |
| Create | POST | `/v2/api/sources` | SourceResource_Write JSON | SourceResource_Read (201) |
| Update | PUT | `/v2/api/sources/{id}` | SourceResource_Write JSON | SourceResource_Read (201) |
| Delete | DELETE | `/v2/api/sources/{id}` | none | DeleteResponse with message and id |

### Known SourceResource_Read Fields
- `id` (string, hashid format)
- `name` (string)
- `description` (string)
- `type` (string, e.g., "AI")
- `sourceType` (string, e.g., "UNKNOWN")
- `version` (string)
- `label` (string)
- `resourceType` (string, always "source")
- `created` (timestamp)
- `updated` (timestamp)
- `syncedWithApiGateway` (boolean)
- `autoDiscoveryEnabled` (boolean)
- `externalId` (string)
- `externalUsagePlanId` (string)
- `tags` (array of strings)
- `sourceClassifications` (array)
- `owner` (object with user reference)
- `team` (object with team reference)
- `environment` (object)
- `products` (array)
- `contracts` (array)
- `meteringId` (string)
- `metadata` (object)
- `assetUsageIdentifier` (string)

### Create/Update Flag Recommendations

Based on the API schema, the most useful flags for create/update:

| Flag | Type | Create | Update | Notes |
|------|------|--------|--------|-------|
| `--name` | string | Required | Optional | Source display name |
| `--type` | string | Required | Optional | e.g., "AI", "API" |
| `--description` | string | Optional | Optional | Source description |

**Note:** The exact required fields for create need validation against the actual API. The OpenAPI spec's SourceResource_Write schema was not fully detailed in the accessible documentation. Start with `--name` and `--type` as required for create, and add more flags as needed based on API error responses. The `--json` raw output will show all available fields.

### Pagination
**Confidence: LOW** -- The OpenAPI spec did not clearly show pagination parameters for the list endpoint. The API may return all results in a single response, or may use an envelope like `{"content": [...], "totalElements": N, "page": {...}}` (common Spring Boot pattern given the Java-style field names). Implementation should:
1. First try decoding as a plain array
2. If that fails or if response has pagination fields, handle the envelope
3. Use `--verbose` output to inspect actual response shape during development

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| lipgloss v1 table API | lipgloss v2 `charm.land/lipgloss/v2/table` | 2025 | Already using v2; fluent API with `.Headers()`, `.Rows()`, `.StyleFunc()` |
| Cobra global vars | Cobra package vars with PersistentPreRunE | established | Project already uses this pattern; continue it |

## Open Questions

1. **List endpoint response shape (pagination)**
   - What we know: GET/PUT/DELETE on `/v2/api/sources/{id}` confirmed. List endpoint likely at `/v2/api/sources`.
   - What's unclear: Whether list returns a bare array or a paginated envelope. Whether there are query params like `page`, `size`.
   - Recommendation: Implement assuming bare array first. Add pagination handling if the actual response reveals an envelope. Use `--verbose` to inspect. Auto-pagination can be added incrementally.

2. **Required fields for source creation**
   - What we know: name, type, description are the primary user-facing fields.
   - What's unclear: Which fields the API requires vs. accepts. The SourceResource_Write schema was not fully documented in the accessible spec.
   - Recommendation: Start with `--name` (required) and `--type` (required). If the API returns 400 with missing field errors, add those fields. The error mapping in `api.Client` already handles non-2xx responses.

3. **Source "status" field availability**
   - What we know: The table should show a Status column. SourceResource_Read has many fields but "status" was not explicitly confirmed in the partial schema.
   - What's unclear: Whether the field is called `status`, `state`, or derived from another field.
   - Recommendation: Try `status` first. If not present, check `sourceType` or another field. The `str()` helper gracefully returns "" for missing fields.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.11.1 |
| Config file | none (Go convention: `go test ./...`) |
| Quick run command | `go test ./cmd/sources/... ./internal/resource/... -v -count=1` |
| Full suite command | `make test` (`go test ./... -v -count=1`) |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SRCS-01 | List sources renders table | unit | `go test ./cmd/sources/... -run TestList -v -count=1` | No - Wave 0 |
| SRCS-01 | List sources empty shows message | unit | `go test ./cmd/sources/... -run TestListEmpty -v -count=1` | No - Wave 0 |
| SRCS-02 | Get source by ID renders single row | unit | `go test ./cmd/sources/... -run TestGet -v -count=1` | No - Wave 0 |
| SRCS-03 | Create source sends POST, renders result | unit | `go test ./cmd/sources/... -run TestCreate -v -count=1` | No - Wave 0 |
| SRCS-04 | Update source sends only changed fields | unit | `go test ./cmd/sources/... -run TestUpdate -v -count=1` | No - Wave 0 |
| SRCS-05 | Delete source prompts and deletes | unit | `go test ./cmd/sources/... -run TestDelete -v -count=1` | No - Wave 0 |
| SRCS-05 | Delete with --yes skips prompt | unit | `go test ./cmd/sources/... -run TestDeleteYes -v -count=1` | No - Wave 0 |
| SRCS-05 | ConfirmDelete helper logic | unit | `go test ./internal/resource/... -run TestConfirm -v -count=1` | No - Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/sources/... ./internal/resource/... -v -count=1`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/sources/list_test.go` -- covers SRCS-01 (list with data + empty)
- [ ] `cmd/sources/get_test.go` -- covers SRCS-02
- [ ] `cmd/sources/create_test.go` -- covers SRCS-03
- [ ] `cmd/sources/update_test.go` -- covers SRCS-04 (partial update semantics)
- [ ] `cmd/sources/delete_test.go` -- covers SRCS-05 (with/without confirmation)
- [ ] `internal/resource/resource_test.go` -- covers ConfirmDelete helper

### Test Pattern (from existing codebase)
Tests use `httptest.NewServer` to mock API responses and `testify/assert` + `testify/require` for assertions. The existing `client_test.go` demonstrates this pattern clearly. Source command tests should:
1. Create an `httptest.NewServer` returning mock JSON
2. Set `cmd.APIClient` to a client pointing at the test server
3. Set `cmd.Output` to a `NewWithWriter` with a `bytes.Buffer`
4. Execute the command and assert on captured output

## Sources

### Primary (HIGH confidence)
- Existing codebase: `cmd/root.go`, `cmd/config/`, `internal/api/client.go`, `internal/output/` -- established patterns
- Revenium OpenAPI spec at `https://api.dev.hcapp.io/profitstream/api-docs/v2` -- endpoint paths and response shapes (partial)

### Secondary (MEDIUM confidence)
- Architecture research (`.planning/research/ARCHITECTURE.md`) -- project structure and command patterns
- Feature research (`.planning/research/FEATURES.md`) -- CRUD pattern recommendations

### Tertiary (LOW confidence)
- Source list endpoint response shape -- inferred, not confirmed from spec
- SourceResource_Write required fields -- partially documented
- Pagination parameters -- not found in spec

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - using only existing dependencies, no new libraries needed
- Architecture: HIGH - follows established patterns from Phase 1/2 codebase
- API endpoints: MEDIUM - GET/PUT/DELETE confirmed, POST and list inferred from REST conventions
- Pitfalls: HIGH - based on direct code analysis and Go/Cobra expertise

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable -- Go ecosystem and project dependencies are settled)
