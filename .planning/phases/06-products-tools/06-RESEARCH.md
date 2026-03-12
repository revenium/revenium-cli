# Phase 6: Products & Tools - Research

**Researched:** 2026-03-12
**Domain:** CRUD command packages for Revenium Products and Tools resources
**Confidence:** HIGH

## Summary

Phase 6 implements two independent CRUD resource packages (`cmd/products/` and `cmd/tools/`) following the identical pattern established in Phases 3-5. The subscribers package (Phase 5) is the closest template since it has standard CRUD without nested resources or special update semantics.

Both resources use the `/v2/api/products` and `/v2/api/tools` API endpoints with standard REST verbs (GET list, GET by ID, POST create, PUT update, DELETE). The tool resource has richer fields (toolType enum, toolProvider, enabled flag) while the product resource fields need to be discovered at implementation time since the API docs did not fully render the product schema. The implementation approach using `map[string]interface{}` means exact field discovery can happen during development without blocking planning.

**Primary recommendation:** Clone the subscribers package structure verbatim for both resources, adjusting only the package name, API path, table columns, flag definitions, and field mappings. No new patterns or infrastructure needed.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Package per resource: `cmd/products/` and `cmd/tools/`
- `RegisterCommand()` in `main.go` for wiring both
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list -> "No products found." / "No tools found."

### Claude's Discretion
- Table columns for products and tools (discover from API)
- Which flags for create/update on each resource
- API endpoint paths for both resources
- Whether products and tools share any infrastructure beyond existing patterns
- Any resource-specific display considerations

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PROD-01 | User can list all products | Standard list pattern from subscribers; API: GET /v2/api/products |
| PROD-02 | User can get a product by ID | Standard get pattern; API: GET /v2/api/products/{id} |
| PROD-03 | User can create a new product | Standard create with flags; API: POST /v2/api/products |
| PROD-04 | User can update a product | Standard update with Flags().Changed(); API: PUT /v2/api/products/{id} |
| PROD-05 | User can delete a product with confirmation prompt | Standard delete with ConfirmDelete(); API: DELETE /v2/api/products/{id} |
| TOOL-01 | User can list all tools | Standard list pattern; API: GET /v2/api/tools |
| TOOL-02 | User can get a tool by ID | Standard get pattern; API: GET /v2/api/tools/{id} |
| TOOL-03 | User can create a new tool | Create with toolId, name, toolType required; API: POST /v2/api/tools |
| TOOL-04 | User can update a tool | Standard update with Flags().Changed(); API: PUT /v2/api/tools/{id} |
| TOOL-05 | User can delete a tool with confirmation prompt | Standard delete with ConfirmDelete(); API: DELETE /v2/api/tools/{id} |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| cobra | (existing) | Command structure | Already used across all resource packages |
| lipgloss/v2 + table | (existing) | Styled table output | Already configured in output package |
| testify | (existing) | Test assertions | Already used in all test files |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| net/http/httptest | stdlib | Mock API server in tests | Every test file |
| internal/output | project | TableDef, Render, RenderJSON | All display commands |
| internal/resource | project | ConfirmDelete() | Delete commands |
| internal/api | project | NewClient for test setup | All test files |

No new dependencies needed. Everything required already exists in the project.

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  products/
    products.go       # Cmd, tableDef, toRows, str, renderProduct
    list.go           # newListCmd()
    get.go            # newGetCmd()
    create.go         # newCreateCmd()
    update.go         # newUpdateCmd()
    delete.go         # newDeleteCmd()
    list_test.go      # TestListProducts, TestListProductsEmpty, TestListProductsJSON, TestListProductsEmptyJSON
    get_test.go       # TestGetProduct, TestGetProductJSON
    create_test.go    # TestCreateProduct, TestCreateProduct* variants
    update_test.go    # TestUpdateProduct, TestUpdateProductNoFields
    delete_test.go    # TestDeleteProductWithYes, TestDeleteProductQuiet, TestDeleteProductJSONMode
  tools/
    tools.go          # Cmd, tableDef, toRows, str, renderTool
    list.go           # newListCmd()
    get.go            # newGetCmd()
    create.go         # newCreateCmd()
    update.go         # newUpdateCmd()
    delete.go         # newDeleteCmd()
    list_test.go
    get_test.go
    create_test.go
    update_test.go
    delete_test.go
```

### Pattern 1: Resource Package Structure (Clone from subscribers)

**What:** Each resource package follows identical structure -- parent command, tableDef, toRows, str, renderX helpers, and individual command files.

**When to use:** Every CRUD resource in this CLI.

**Example (products.go):**
```go
package products

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/revenium/revenium-cli/cmd"
    "github.com/revenium/revenium-cli/internal/output"
)

var Cmd = &cobra.Command{
    Use:   "products",
    Short: "Manage products",
    Example: `  # List all products
  revenium products list

  # Get a specific product
  revenium products get abc-123`,
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

func toRows(products []map[string]interface{}) [][]string {
    rows := make([][]string, len(products))
    for i, p := range products {
        rows[i] = []string{
            str(p, "id"),
            str(p, "name"),
            str(p, "status"),
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

func renderProduct(product map[string]interface{}) error {
    rows := [][]string{{
        str(product, "id"),
        str(product, "name"),
        str(product, "status"),
    }}
    return cmd.Output.Render(tableDef, rows, product)
}
```

### Pattern 2: Registration in main.go

**What:** Add two import lines and two RegisterCommand calls.

**Example:**
```go
import (
    // ... existing imports
    "github.com/revenium/revenium-cli/cmd/products"
    "github.com/revenium/revenium-cli/cmd/tools"
)

func init() {
    // ... existing registrations
    cmd.RegisterCommand(products.Cmd, "resources")
    cmd.RegisterCommand(tools.Cmd, "resources")
}
```

### Pattern 3: Test Pattern (from subscribers)

**What:** Each test creates httptest.NewServer, sets cmd.APIClient and cmd.Output, constructs the command, sets args, and executes.

**Example:**
```go
func TestListProducts(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/v2/api/products", r.URL.Path)
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, `[{"id": "prod-1", "name": "My Product", "status": "active"}]`)
    }))
    defer srv.Close()

    var buf bytes.Buffer
    cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
    cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

    c := newListCmd()
    c.SetOut(&buf)
    err := c.Execute()

    require.NoError(t, err)
    assert.Contains(t, buf.String(), "prod-1")
    assert.Contains(t, buf.String(), "My Product")
}
```

### Anti-Patterns to Avoid
- **Shared str() across packages:** Each package defines its own `str()` helper. Do NOT try to extract a common utility -- it would couple packages unnecessarily for a 5-line function.
- **Struct-based API responses:** Use `map[string]interface{}` per project convention. Do not define Go structs for API responses.
- **Success messages after create/update:** Render the result directly. No "Product created successfully" messages.

## API Endpoints and Fields

### Products API

**Confidence: MEDIUM** -- Endpoint paths confirmed from API docs; field names partially inferred.

| Operation | Method | Path |
|-----------|--------|------|
| List | GET | /v2/api/products |
| Get | GET | /v2/api/products/{id} |
| Create | POST | /v2/api/products |
| Update | PUT | /v2/api/products/{id} |
| Delete | DELETE | /v2/api/products/{id} |

**Likely product fields** (to be confirmed during implementation):
- `id` (string, read-only) -- unique identifier
- `name` (string) -- product name
- `description` (string) -- product description
- `status` (string) -- product status (likely: active/inactive)
- `created` (string) -- creation timestamp
- `updated` (string) -- last update timestamp
- `resourceType` (string, read-only) -- always "product"

**Recommended table columns:** ID, Name, Status (with StatusColumn coloring)
**Recommended create flags:** --name (required), --description (optional)
**Recommended update flags:** --name, --description (all optional, at least one required)

**Note:** Product schema was not fully available in API docs. The `map[string]interface{}` approach means fields can be adjusted at implementation time without changing architecture.

### Tools API

**Confidence: HIGH** -- Full schema retrieved from API documentation.

| Operation | Method | Path |
|-----------|--------|------|
| List | GET | /v2/api/tools |
| Get | GET | /v2/api/tools/{id} |
| Create | POST | /v2/api/tools |
| Update | PUT | /v2/api/tools/{id} |
| Delete | DELETE | /v2/api/tools/{id} |

**Tool fields (confirmed):**
- `id` (string, read-only) -- unique identifier
- `toolId` (string) -- unique identifier within organization
- `name` (string) -- display name
- `description` (string) -- detailed description
- `toolType` (string, enum) -- MCP_SERVER, MULTIMODAL, TOOL_CALL, CUSTOM
- `toolProvider` (string) -- provider/vendor
- `enabled` (boolean) -- whether active for metering (default: true)
- `configSource` (string, read-only) -- DATABASE, YAML, MERGED
- `created` (string, read-only) -- creation timestamp
- `updated` (string, read-only) -- last update timestamp

**Recommended table columns:** ID, Name, Type, Provider, Enabled
**Recommended create flags:** --name (required), --tool-id (required), --tool-type (required, enum), --description (optional), --tool-provider (optional), --enabled (optional bool)
**Recommended update flags:** --name, --tool-id, --tool-type, --description, --tool-provider, --enabled (all optional, at least one required)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Delete confirmation | Custom prompt | `resource.ConfirmDelete()` | Already handles --yes, JSON mode, non-TTY |
| Table rendering | Manual formatting | `output.TableDef` + `Render()` | Handles styled tables, JSON, quiet mode |
| API calls | Custom HTTP | `cmd.APIClient.Do()` | Handles auth, timeouts, error mapping |
| Flag-based partial update | Manual body construction | `Flags().Changed()` pattern | Only sends changed fields |

**Key insight:** This phase requires zero new infrastructure. Everything is pure replication of the subscribers pattern.

## Common Pitfalls

### Pitfall 1: Forgetting to register in main.go
**What goes wrong:** Commands exist but don't appear in CLI help
**Why it happens:** New package created but import + RegisterCommand missed
**How to avoid:** Registration in main.go is a required step, not optional cleanup
**Warning signs:** `revenium products` returns "unknown command"

### Pitfall 2: StatusColumn index mismatch
**What goes wrong:** Wrong column gets status coloring
**Why it happens:** StatusColumn index doesn't match column position in Headers array
**How to avoid:** StatusColumn is 0-indexed. If Status is the 3rd column (index 2), set StatusColumn: 2. If no status column exists, set StatusColumn: -1
**Warning signs:** Random column appears colored; status column appears unstyled

### Pitfall 3: Missing --yes flag in delete tests
**What goes wrong:** Delete test hangs waiting for interactive confirmation
**Why it happens:** The --yes flag is inherited from rootCmd at runtime but not available in isolated test commands
**How to avoid:** In delete tests, register `c.Flags().Bool("yes", false, "Skip confirmation prompts")` on the test command before execution (see existing delete_test.go files)
**Warning signs:** Test hangs indefinitely

### Pitfall 4: Boolean flags for tools enabled field
**What goes wrong:** `--enabled` flag always sends a value even when not specified
**Why it happens:** Bool flags default to false, so Flags().Changed() must be used
**How to avoid:** Use `Flags().Changed("enabled")` to check if user explicitly set the flag before including in body
**Warning signs:** Update always sets enabled=false when flag not provided

### Pitfall 5: Product fields unknown at research time
**What goes wrong:** Implementer guesses wrong fields for product schema
**Why it happens:** API docs did not render product schema completely
**How to avoid:** Use map[string]interface{} and discover fields from actual API response. Start with likely fields (name, description, status) and adjust based on what the API returns. The CLI pattern is resilient to this.
**Warning signs:** Create returns 400 -- check API error message for required fields

## Code Examples

### List Command (verified pattern from cmd/subscribers/list.go)
```go
func newListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all products",
        Args:  cobra.NoArgs,
        Example: `  # List all products
  revenium products list

  # List products as JSON
  revenium products list --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var products []map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/products", nil, &products); err != nil {
                return err
            }
            if len(products) == 0 {
                if cmd.Output.IsJSON() {
                    return cmd.Output.RenderJSON([]interface{}{})
                }
                fmt.Fprintln(c.OutOrStdout(), "No products found.")
                return nil
            }
            return cmd.Output.Render(tableDef, toRows(products), products)
        },
    }
}
```

### Delete Command (verified pattern from cmd/subscribers/delete.go)
```go
func newDeleteCmd() *cobra.Command {
    c := &cobra.Command{
        Use:   "delete <id>",
        Short: "Delete a product",
        Args:  cobra.ExactArgs(1),
        Example: `  # Delete a product (with confirmation)
  revenium products delete abc-123

  # Delete without confirmation
  revenium products delete abc-123 --yes`,
        RunE: func(c *cobra.Command, args []string) error {
            id := args[0]
            yes, _ := c.Flags().GetBool("yes")

            ok, err := resource.ConfirmDelete("product", id, yes, cmd.Output.IsJSON())
            if err != nil {
                return err
            }
            if !ok {
                return nil
            }

            if err := cmd.APIClient.Do(c.Context(), "DELETE", "/v2/api/products/"+id, nil, nil); err != nil {
                return err
            }

            if !cmd.Output.IsQuiet() {
                fmt.Fprintf(c.OutOrStdout(), "Deleted product %s.\n", id)
            }
            return nil
        },
    }
    return c
}
```

## State of the Art

No changes since Phase 5. The CRUD pattern is stable and proven across sources, models, subscribers, and subscriptions.

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A | Replicate subscribers pattern | Phase 3 established, refined through Phase 5 | Direct clone with field changes only |

## Open Questions

1. **Product schema fields**
   - What we know: API endpoint is /v2/api/products, standard CRUD. Likely has name, description, status based on other resources.
   - What's unclear: Exact field names and which are required for create
   - Recommendation: Start with name (required) + description (optional). If API returns 400, read error message to discover required fields. The map[string]interface{} approach handles this gracefully.

2. **Product status column**
   - What we know: Sources have a status column with colored styling. Products likely also have status.
   - What's unclear: Whether products have a status field and what values it takes
   - Recommendation: Include StatusColumn in tableDef if API response includes "status". Set to -1 if it does not.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | None needed (go test built-in) |
| Quick run command | `go test ./cmd/products/... ./cmd/tools/...` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PROD-01 | List products (table, empty, JSON, empty JSON) | unit | `go test ./cmd/products/ -run TestList -x` | Wave 0 |
| PROD-02 | Get product by ID (table, JSON) | unit | `go test ./cmd/products/ -run TestGet -x` | Wave 0 |
| PROD-03 | Create product (full fields, minimal) | unit | `go test ./cmd/products/ -run TestCreate -x` | Wave 0 |
| PROD-04 | Update product (partial, no fields error) | unit | `go test ./cmd/products/ -run TestUpdate -x` | Wave 0 |
| PROD-05 | Delete product (--yes, quiet, JSON mode) | unit | `go test ./cmd/products/ -run TestDelete -x` | Wave 0 |
| TOOL-01 | List tools (table, empty, JSON, empty JSON) | unit | `go test ./cmd/tools/ -run TestList -x` | Wave 0 |
| TOOL-02 | Get tool by ID (table, JSON) | unit | `go test ./cmd/tools/ -run TestGet -x` | Wave 0 |
| TOOL-03 | Create tool (full fields, minimal) | unit | `go test ./cmd/tools/ -run TestCreate -x` | Wave 0 |
| TOOL-04 | Update tool (partial, no fields error) | unit | `go test ./cmd/tools/ -run TestUpdate -x` | Wave 0 |
| TOOL-05 | Delete tool (--yes, quiet, JSON mode) | unit | `go test ./cmd/tools/ -run TestDelete -x` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/products/... ./cmd/tools/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before /gsd:verify-work

### Wave 0 Gaps
- [ ] `cmd/products/` directory -- entire package (all .go and _test.go files)
- [ ] `cmd/tools/` directory -- entire package (all .go and _test.go files)

*(All tests are new -- created alongside implementation per established pattern)*

## Sources

### Primary (HIGH confidence)
- cmd/subscribers/ -- Full CRUD package used as template (read all files)
- cmd/sources/ -- Alternative CRUD reference (read sources.go, create.go)
- internal/output/table.go -- TableDef struct and rendering
- internal/resource/resource.go -- ConfirmDelete helper
- main.go -- RegisterCommand pattern

### Secondary (MEDIUM confidence)
- [Revenium API Reference](https://revenium.readme.io/reference/getting-started-with-your-api) -- Endpoint paths for products and tools
- [Revenium API Reference - Create Tool](https://revenium.readme.io/reference/create_tool) -- Tool schema with field names and types
- [Revenium API Reference - List Tools](https://revenium.readme.io/reference/list_tools) -- Tool response schema

### Tertiary (LOW confidence)
- Product field names (name, description, status) -- inferred from other resources, not confirmed from API docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, pure replication
- Architecture: HIGH -- identical to 4 prior resource packages
- Pitfalls: HIGH -- documented from actual patterns in codebase
- Tool fields: HIGH -- confirmed from API documentation
- Product fields: LOW -- API docs did not render product schema; using inference

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable -- pattern is internal, not dependent on external changes)
