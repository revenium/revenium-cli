# Phase 4: AI Models & Pricing - Research

**Researched:** 2026-03-12
**Domain:** Go CLI CRUD commands for AI models and nested pricing dimensions (Cobra + Lip Gloss)
**Confidence:** HIGH

## Summary

Phase 4 implements AI model management and nested pricing dimension CRUD, following the patterns established in Phase 3 (Sources). The key novelty is the first nested resource: pricing dimensions live under a specific model (`revenium models pricing list <model-id>`). The Revenium API serves AI models under `/v2/api/sources/ai/models` with the notable constraint that models use PATCH (not PUT) for updates, and there is no model creation endpoint (models are auto-discovered by the platform).

The API structure for pricing dimensions is at `/v2/api/sources/ai/models/{modelId}/pricing/dimensions` with POST (create single), PUT (update by ID), and DELETE operations. There is also a batch PUT at the collection level. The pricing dimension schema is referenced but not fully documented in the OpenAPI spec -- field names will need to be discovered from actual API responses during implementation.

The `cmd/sources/` package provides an exact template for the `cmd/models/` package. The model commands replicate list/get/update/delete (no create), while pricing subcommands add create/list/update/delete nested under the models command. The subcommand nesting pattern (`models pricing ...`) is a new structure that needs clean implementation for reuse in future nested resources.

**Primary recommendation:** Build `cmd/models/` following the sources pattern exactly, with a `pricing` subcommand group for nested pricing dimension operations. Use PATCH for model updates. Discover pricing dimension fields from the API at runtime using `map[string]interface{}`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Package per resource: `cmd/models/` with verb files
- `RegisterCommand()` in `main.go` for wiring
- Essential columns in list table, single-row for get
- `map[string]interface{}` for API responses, `str()` for extraction
- Partial update via `Flags().Changed()`, delete via `ConfirmDelete()`
- Long flags only, 2-3 help examples per command
- Render result after create/update (no extra success message)
- Empty list: "No models found." / "No pricing dimensions found."
- All Phase 3 patterns apply

### Claude's Discretion
- Table columns for models list (ID, Name, Type/Provider, Status -- or whatever fields the API returns that are most useful)
- Table columns for pricing dimensions list
- How model-id is passed to pricing subcommands (positional arg: `models pricing list <model-id>`)
- Whether AIMD-03 (PATCH) uses a different HTTP method than regular update
- Subcommand nesting structure for `models pricing`
- Which flags are available for pricing dimension create/update
- How to structure the `cmd/models/` package to handle both model and pricing commands

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AIMD-01 | User can list all AI models | GET /v2/api/sources/ai/models, table with ID/Name/Provider/Mode columns, statusStyle not needed (no status field) |
| AIMD-02 | User can get an AI model by ID | GET /v2/api/sources/ai/models/{id}, single-row table rendering via renderModel() |
| AIMD-03 | User can update AI model pricing (PATCH) | PATCH /v2/api/sources/ai/models/{id}?teamId=X, flags for inputCostPerToken/outputCostPerToken/etc. |
| AIMD-04 | User can delete an AI model | DELETE /v2/api/sources/ai/models/{id}, reuse ConfirmDelete("model", id, ...) |
| AIMD-05 | User can list pricing dimensions for a model | GET /v2/api/sources/ai/models/{modelId}/pricing/dimensions (needs discovery -- may not exist as GET), positional model-id arg |
| AIMD-06 | User can create a pricing dimension for a model | POST /v2/api/sources/ai/models/{modelId}/pricing/dimensions, flags TBD from schema discovery |
| AIMD-07 | User can update a pricing dimension | PUT /v2/api/sources/ai/models/{modelId}/pricing/dimensions/{dimensionId} |
| AIMD-08 | User can delete a pricing dimension | DELETE /v2/api/sources/ai/models/{modelId}/pricing/dimensions/{dimensionId}, ConfirmDelete("pricing dimension", id, ...) |
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
This phase adds zero new libraries. Everything is built on Phase 1-3 infrastructure.

## API Endpoints

Based on OpenAPI spec analysis at `https://api.dev.hcapp.io/profitstream/api-docs/v2`:

### AI Model Endpoints (Confidence: HIGH)

| Operation | Method | Path | Request Body | Response | Notes |
|-----------|--------|------|-------------|----------|-------|
| List | GET | `/v2/api/sources/ai/models` | none | Array of AIModelResource_Read | May accept teamId query param |
| Get | GET | `/v2/api/sources/ai/models/{id}` | none | Single AIModelResource_Read | Inferred from pattern (not explicitly in spec) |
| Update (PATCH) | PATCH | `/v2/api/sources/ai/models/{id}` | AIModelPatchResource JSON | AIModelResource_Read | **teamId query param required** |
| Delete | DELETE | `/v2/api/sources/ai/models/{id}` | none | DeleteResponse | Standard delete pattern |

**CRITICAL: Models use PATCH, not PUT.** The spec explicitly states: "PUT is not supported for AI model updates. Use PATCH for pricing updates on tenant-specific models." The `cmd.APIClient.Do()` method already supports any HTTP method string.

### AIModelResource_Read Fields (Confidence: HIGH)

| Field | Type | Table Column? | Notes |
|-------|------|---------------|-------|
| `id` | string | Yes | Primary identifier |
| `name` | string | Yes | Model name (e.g., "gpt-4o") |
| `provider` | string | Yes | AI provider (e.g., "OpenAI") |
| `mode` | string | Yes | Model mode/type |
| `inputCostPerToken` | number | No (detail view) | Per-token input pricing |
| `outputCostPerToken` | number | No (detail view) | Per-token output pricing |
| `cacheCreationCostPerInputToken` | number | No | Cache creation cost |
| `cacheReadCostPerInputToken` | number | No | Cache read cost |
| `resourceType` | string | No | Always "ai_model_metadata" |
| `label` | string | No | Display label |
| `created` | timestamp | No | Creation time |
| `updated` | timestamp | No | Last update time |
| `team` | object | No | Owning organization |
| `supportFunctionCalling` | boolean | No | Capability flag |
| `supportsParallelFunctionCalling` | boolean | No | Capability flag |
| `supportsResponseSchema` | boolean | No | Capability flag |
| `supportsVision` | boolean | No | Capability flag |
| `supportsPromptCaching` | boolean | No | Capability flag |
| `supportsSystemMessages` | boolean | No | Capability flag |
| `supportsToolChoice` | boolean | No | Capability flag |
| `supportsWebSearch` | boolean | No | Capability flag |
| `_links` | object | No | HATEOAS links |

### AIModelPatchResource Fields (for PATCH update)

| Field | Type | Flag Name | Notes |
|-------|------|-----------|-------|
| `inputCostPerToken` | number | `--input-cost-per-token` | Cost per input token |
| `outputCostPerToken` | number | `--output-cost-per-token` | Cost per output token |
| `cacheCreationCostPerInputToken` | number | `--cache-creation-cost-per-input-token` | Cache creation cost |
| `cacheReadCostPerInputToken` | number | `--cache-read-cost-per-input-token` | Cache read cost |

### Pricing Dimension Endpoints (Confidence: MEDIUM)

| Operation | Method | Path | Request Body | Response | Notes |
|-----------|--------|------|-------------|----------|-------|
| List | GET | `/v2/api/sources/ai/models/{modelId}/pricing/dimensions` | none | Array of PricingDimensionResource_Read | **Not explicitly in spec -- needs validation** |
| Create | POST | `/v2/api/sources/ai/models/{modelId}/pricing/dimensions` | PricingDimensionResource_Read | PricingDimensionResource_Read | Confirmed in spec |
| Update | PUT | `/v2/api/sources/ai/models/{modelId}/pricing/dimensions/{dimensionId}` | PricingDimensionResource_Read | PricingDimensionResource_Read | Confirmed in spec |
| Delete | DELETE | `/v2/api/sources/ai/models/{modelId}/pricing/dimensions/{dimensionId}` | none | DeleteResponse | Confirmed in spec |
| Batch Save | PUT | `/v2/api/sources/ai/models/{modelId}/pricing/dimensions` | Array of PricingDimensionResource_Read | Array | Collection-level PUT replaces all |

### PricingDimensionResource_Read Fields (Confidence: LOW)

The PricingDimensionResource_Read schema is referenced via `$ref` in the OpenAPI spec but its fields are not enumerated. **Field names must be discovered from actual API responses.**

Likely fields based on domain knowledge (needs validation):
- `id` - dimension identifier
- `name` or `label` - dimension display name
- `type` or `dimensionType` - what the dimension measures
- `unitPrice` or `price` - price per unit
- `unit` or `unitOfMeasure` - unit type
- Other pricing-related fields

**Recommendation:** Use `--verbose` during development to inspect actual response shapes. The `map[string]interface{}` pattern handles unknown schemas gracefully.

### teamId Query Parameter (Confidence: HIGH)

The `teamId` query parameter is **required for PATCH** operations on models. It identifies the tenant/organization for tenant-specific pricing. Implementation must either:
1. Accept `--team-id` as a flag on model update commands
2. Or derive it from the model's team field after a GET

**Recommendation:** Add `--team-id` as a required flag on `models update`. This is simpler and makes the command explicit.

## Architecture Patterns

### Recommended Project Structure
```
cmd/
├── models/
│   ├── models.go           # Parent command + subcommand registration + shared helpers
│   ├── list.go             # revenium models list
│   ├── get.go              # revenium models get <id>
│   ├── update.go           # revenium models update <id> --input-cost-per-token 0.01
│   ├── delete.go           # revenium models delete <id>
│   ├── pricing.go          # Parent pricing subcommand + pricing helpers
│   ├── pricing_list.go     # revenium models pricing list <model-id>
│   ├── pricing_create.go   # revenium models pricing create <model-id> --flags
│   ├── pricing_update.go   # revenium models pricing update <model-id> <dimension-id> --flags
│   └── pricing_delete.go   # revenium models pricing delete <model-id> <dimension-id>
├── sources/
│   └── ...                 # Existing sources package
└── root.go
```

### Pattern 1: Nested Subcommand Group (NEW for Phase 4)
**What:** The `models` command has a `pricing` subcommand group that itself has verb subcommands. This is the first nested resource pattern.
**When to use:** Any resource with child resources (models -> pricing dimensions).
**Example:**
```go
// cmd/models/models.go
package models

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
    Use:   "models",
    Short: "Manage AI models and pricing",
    Example: `  # List all AI models
  revenium models list

  # Get a specific model
  revenium models get abc-123

  # List pricing dimensions for a model
  revenium models pricing list abc-123`,
}

func init() {
    Cmd.AddCommand(newListCmd())
    Cmd.AddCommand(newGetCmd())
    Cmd.AddCommand(newUpdateCmd())
    Cmd.AddCommand(newDeleteCmd())
    Cmd.AddCommand(pricingCmd) // nested subcommand group
}
```

```go
// cmd/models/pricing.go
package models

import "github.com/spf13/cobra"

var pricingCmd = &cobra.Command{
    Use:   "pricing",
    Short: "Manage pricing dimensions for AI models",
    Example: `  # List pricing dimensions for a model
  revenium models pricing list abc-123

  # Create a pricing dimension
  revenium models pricing create abc-123 --name "Input Tokens" --price 0.01`,
}

func init() {
    pricingCmd.AddCommand(newPricingListCmd())
    pricingCmd.AddCommand(newPricingCreateCmd())
    pricingCmd.AddCommand(newPricingUpdateCmd())
    pricingCmd.AddCommand(newPricingDeleteCmd())
}
```

**IMPORTANT:** The `pricingCmd` init runs within the `models` package, so there are no circular import issues. All pricing commands are in the same package as model commands.

### Pattern 2: Model-ID as Positional Arg for Pricing Commands
**What:** Pricing subcommands take model-id as the first positional argument, and dimension-id as the second where needed.
**When to use:** All pricing subcommands.
**Example:**
```go
// cmd/models/pricing_list.go
func newPricingListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list <model-id>",
        Short: "List pricing dimensions for an AI model",
        Args:  cobra.ExactArgs(1),
        Example: `  # List pricing dimensions
  revenium models pricing list abc-123

  # List as JSON
  revenium models pricing list abc-123 --json`,
        RunE: func(c *cobra.Command, args []string) error {
            modelID := args[0]
            var dims []map[string]interface{}
            path := "/v2/api/sources/ai/models/" + modelID + "/pricing/dimensions"
            if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &dims); err != nil {
                return err
            }
            if len(dims) == 0 {
                if cmd.Output.IsJSON() {
                    return cmd.Output.RenderJSON([]interface{}{})
                }
                fmt.Fprintln(c.OutOrStdout(), "No pricing dimensions found.")
                return nil
            }
            return cmd.Output.Render(pricingTableDef, toPricingRows(dims), dims)
        },
    }
}
```

### Pattern 3: Two Positional Args for Pricing Update/Delete
**What:** Pricing update and delete need both model-id and dimension-id.
**When to use:** Operations on a specific pricing dimension.
**Example:**
```go
// cmd/models/pricing_delete.go
func newPricingDeleteCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "delete <model-id> <dimension-id>",
        Short: "Delete a pricing dimension",
        Args:  cobra.ExactArgs(2),
        Example: `  # Delete a pricing dimension
  revenium models pricing delete model-123 dim-456

  # Delete without confirmation
  revenium models pricing delete model-123 dim-456 --yes`,
        RunE: func(c *cobra.Command, args []string) error {
            modelID, dimID := args[0], args[1]
            yes, _ := c.Flags().GetBool("yes")
            ok, err := resource.ConfirmDelete("pricing dimension", dimID, yes, cmd.Output.IsJSON())
            if err != nil {
                return err
            }
            if !ok {
                return nil
            }
            path := "/v2/api/sources/ai/models/" + modelID + "/pricing/dimensions/" + dimID
            if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, nil); err != nil {
                return err
            }
            if !cmd.Output.IsQuiet() {
                fmt.Fprintf(c.OutOrStdout(), "Deleted pricing dimension %s.\n", dimID)
            }
            return nil
        },
    }
}
```

### Pattern 4: PATCH Instead of PUT for Model Update
**What:** Models use PATCH for updates, not PUT. The API explicitly disallows PUT for model updates.
**When to use:** `models update` command only.
**Example:**
```go
// cmd/models/update.go
func newUpdateCmd() *cobra.Command {
    var inputCost, outputCost, cacheCreateCost, cacheReadCost float64
    var teamID string

    c := &cobra.Command{
        Use:   "update <id>",
        Short: "Update AI model pricing",
        Args:  cobra.ExactArgs(1),
        Example: `  # Update input cost per token
  revenium models update abc-123 --team-id team-1 --input-cost-per-token 0.003

  # Update multiple pricing fields
  revenium models update abc-123 --team-id team-1 --input-cost-per-token 0.003 --output-cost-per-token 0.015`,
        RunE: func(c *cobra.Command, args []string) error {
            id := args[0]
            body := make(map[string]interface{})

            if c.Flags().Changed("input-cost-per-token") {
                body["inputCostPerToken"] = inputCost
            }
            if c.Flags().Changed("output-cost-per-token") {
                body["outputCostPerToken"] = outputCost
            }
            if c.Flags().Changed("cache-creation-cost-per-input-token") {
                body["cacheCreationCostPerInputToken"] = cacheCreateCost
            }
            if c.Flags().Changed("cache-read-cost-per-input-token") {
                body["cacheReadCostPerInputToken"] = cacheReadCost
            }

            if len(body) == 0 {
                return fmt.Errorf("no fields specified to update")
            }

            path := "/v2/api/sources/ai/models/" + id + "?teamId=" + teamID
            var result map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "PATCH", path, body, &result); err != nil {
                return err
            }
            return renderModel(result)
        },
    }

    c.Flags().StringVar(&teamID, "team-id", "", "Team/organization ID (required)")
    c.Flags().Float64Var(&inputCost, "input-cost-per-token", 0, "Cost per input token")
    c.Flags().Float64Var(&outputCost, "output-cost-per-token", 0, "Cost per output token")
    c.Flags().Float64Var(&cacheCreateCost, "cache-creation-cost-per-input-token", 0, "Cache creation cost per input token")
    c.Flags().Float64Var(&cacheReadCost, "cache-read-cost-per-input-token", 0, "Cache read cost per input token")
    _ = c.MarkFlagRequired("team-id")

    return c
}
```

### Pattern 5: Recommended Table Columns

**Models list table:**
```go
var modelTableDef = output.TableDef{
    Headers: []string{"ID", "Name", "Provider", "Mode"},
    // No StatusColumn -- models don't have a status field
}

func toModelRows(models []map[string]interface{}) [][]string {
    rows := make([][]string, len(models))
    for i, m := range models {
        rows[i] = []string{
            str(m, "id"),
            str(m, "name"),
            str(m, "provider"),
            str(m, "mode"),
        }
    }
    return rows
}
```

**Pricing dimensions list table (tentative -- fields need discovery):**
```go
var pricingTableDef = output.TableDef{
    Headers: []string{"ID", "Name", "Type", "Price"},
}
```

### Anti-Patterns to Avoid
- **Using PUT for model updates:** The API explicitly rejects PUT. Use PATCH.
- **Creating a `models create` command:** Models are auto-discovered by the platform. No create endpoint exists.
- **Hardcoding pricing dimension field names before discovery:** Use `map[string]interface{}` and discover fields from actual API responses. Table columns for pricing can be adjusted after seeing real data.
- **Embedding teamId in the path segment:** The `teamId` is a query parameter, not a path parameter. Use `?teamId=X` appended to the path.
- **Separate packages for models and pricing:** Keep both in `cmd/models/` package. Separate packages would complicate the nesting without benefit.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Delete confirmation | Custom prompt per command | `resource.ConfirmDelete()` | Already built in Phase 3, handles --yes/--json/non-TTY |
| Table rendering | Custom column formatting | `output.Formatter.Render()` | Already built in Phase 2 |
| JSON output | Custom marshal/write | `output.Formatter.RenderJSON()` | Already built in Phase 2 |
| HTTP + auth | Custom request building | `api.Client.Do()` | Already built in Phase 1, supports any HTTP method including PATCH |
| Flag change detection | Custom tracking | `cmd.Flags().Changed()` | Built into Cobra/pflag |
| Command registration | Manual rootCmd import | `cmd.RegisterCommand()` | Avoids circular imports, established in Phase 3 |
| str() helper | New extraction function | Copy from sources pattern | Same `str(m, key)` helper, can be per-package or shared |

**Key insight:** Phase 4 adds zero new infrastructure. Every command is a thin wrapper: parse flags/args, build path, call API, render result. The only new concept is subcommand nesting, which is native Cobra.

## Common Pitfalls

### Pitfall 1: Using PUT Instead of PATCH for Model Update
**What goes wrong:** API returns an error or ignores the request.
**Why it happens:** All other resources use PUT for updates. Models are the exception.
**How to avoid:** Use `cmd.APIClient.Do(c.Context(), "PATCH", path, body, &result)`. The Do() method accepts any HTTP method string.
**Warning signs:** 405 Method Not Allowed or similar error from update command.

### Pitfall 2: Missing teamId on PATCH Requests
**What goes wrong:** API returns 400 or 403 because teamId is required for tenant-specific pricing.
**Why it happens:** teamId is a query parameter, easy to forget.
**How to avoid:** Make `--team-id` a required flag on the update command. Append it to the URL path as `?teamId=value`.
**Warning signs:** Error response mentioning missing team or organization context.

### Pitfall 3: Pricing Dimension List Endpoint May Not Exist as GET
**What goes wrong:** GET request to `/v2/api/sources/ai/models/{modelId}/pricing/dimensions` returns 405 or unexpected response.
**Why it happens:** The OpenAPI spec only shows POST and PUT at the collection level -- no GET is documented.
**How to avoid:** Try the GET endpoint first. If it fails, consider: (a) checking if pricing dimensions are embedded in the model GET response, (b) using the model's `_links` HATEOAS links, or (c) flagging as a known limitation.
**Warning signs:** 405 Method Not Allowed on pricing list command.

### Pitfall 4: Pricing Dimension Schema Unknown
**What goes wrong:** Create/update commands have wrong flags, or table columns don't match actual fields.
**Why it happens:** PricingDimensionResource_Read schema is not fully documented in the OpenAPI spec.
**How to avoid:** First implement the list command, use `--json` to inspect actual response fields, then define table columns and create/update flags based on real data. This is why `map[string]interface{}` is the right approach.
**Warning signs:** Flags don't map to actual API fields; table shows empty columns.

### Pitfall 5: Cobra init() Ordering for Nested Subcommands
**What goes wrong:** `pricingCmd` init adds subcommands before the var is initialized, or duplicate command registration.
**Why it happens:** Go `init()` functions in the same package run in file order (alphabetical by filename).
**How to avoid:** Define `pricingCmd` as a package-level var in `pricing.go`. Use a separate `init()` in `pricing.go` to add pricing subcommands to `pricingCmd`. In `models.go` init(), add `pricingCmd` to `Cmd`. Since `pricing.go` comes after `models.go` alphabetically... actually, this could be an issue. **Better approach:** Don't use init() for pricing subcommand registration. Instead, add pricing subcommands explicitly in the `models.go` init() or use a helper function called from models.go init().
**Warning signs:** Nil pointer panic on startup, or "unknown command" errors.

### Pitfall 6: Float64 Flag Default Values
**What goes wrong:** Zero (0.0) is a valid price, but it's also the default for float64 flags. `Flags().Changed()` correctly handles this -- it tracks whether the flag was explicitly set, not whether the value differs from default.
**Why it happens:** Confusion between "zero value" and "not set."
**How to avoid:** Always use `cmd.Flags().Changed("flag-name")` to detect explicit setting, not value comparison. This is the same pattern as string flags in sources update.
**Warning signs:** Unable to set a price to 0.

## Code Examples

### Registration in main.go
```go
// main.go init() - add alongside sources registration
cmd.RegisterCommand(models.Cmd, "resources")
```

### Model Get Command
```go
// cmd/models/get.go
func newGetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "get <id>",
        Short: "Get an AI model by ID",
        Args:  cobra.ExactArgs(1),
        Example: `  # Get a model by ID
  revenium models get abc-123

  # Get a model as JSON
  revenium models get abc-123 --json`,
        RunE: func(c *cobra.Command, args []string) error {
            var model map[string]interface{}
            if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/ai/models/"+args[0], nil, &model); err != nil {
                return err
            }
            return renderModel(model)
        },
    }
}
```

### Shared Helpers in models.go
```go
// cmd/models/models.go

var modelTableDef = output.TableDef{
    Headers: []string{"ID", "Name", "Provider", "Mode"},
}

func toModelRows(models []map[string]interface{}) [][]string {
    rows := make([][]string, len(models))
    for i, m := range models {
        rows[i] = []string{
            str(m, "id"),
            str(m, "name"),
            str(m, "provider"),
            str(m, "mode"),
        }
    }
    return rows
}

func renderModel(model map[string]interface{}) error {
    rows := [][]string{{
        str(model, "id"),
        str(model, "name"),
        str(model, "provider"),
        str(model, "mode"),
    }}
    return cmd.Output.Render(modelTableDef, rows, model)
}

func str(m map[string]interface{}, key string) string {
    if v, ok := m[key]; ok && v != nil {
        return fmt.Sprint(v)
    }
    return ""
}
```

### Pricing Create Command (tentative -- fields need discovery)
```go
// cmd/models/pricing_create.go
func newPricingCreateCmd() *cobra.Command {
    var name, dimType string
    var price float64

    c := &cobra.Command{
        Use:   "create <model-id>",
        Short: "Create a pricing dimension for an AI model",
        Args:  cobra.ExactArgs(1),
        Example: `  # Create a pricing dimension
  revenium models pricing create abc-123 --name "Input Tokens" --type input --price 0.003`,
        RunE: func(c *cobra.Command, args []string) error {
            modelID := args[0]
            body := map[string]interface{}{
                "name":  name,
                "type":  dimType,
                "price": price,
            }
            // Add other changed fields as discovered

            var result map[string]interface{}
            path := "/v2/api/sources/ai/models/" + modelID + "/pricing/dimensions"
            if err := cmd.APIClient.Do(c.Context(), "POST", path, body, &result); err != nil {
                return err
            }
            return renderPricingDimension(result)
        },
    }

    // Flags are tentative -- actual field names need API discovery
    c.Flags().StringVar(&name, "name", "", "Dimension name")
    c.Flags().StringVar(&dimType, "type", "", "Dimension type")
    c.Flags().Float64Var(&price, "price", 0, "Unit price")

    return c
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| PUT for all updates | PATCH for AI model pricing updates | API design | Must use PATCH, not PUT, for model updates |
| Flat resource CRUD | Nested resources (models -> pricing) | Phase 4 | First nested subcommand pattern in this CLI |
| Create for all resources | No create for auto-discovered models | API design | Models discovered by platform; CLI only manages existing |

## Open Questions

1. **Pricing dimension GET list endpoint**
   - What we know: POST and PUT (batch) exist at the collection level. Individual PUT and DELETE exist.
   - What's unclear: Whether GET at the collection level returns a list. The spec does not document it.
   - Recommendation: Try GET first. If 405, check if dimensions are embedded in model GET response. Fall back to documenting as limitation.

2. **PricingDimensionResource_Read fields**
   - What we know: Schema is referenced but fields not enumerated in the OpenAPI spec.
   - What's unclear: Exact field names, types, and which are required for create.
   - Recommendation: Implement list/get first with `--json` output, inspect actual API responses, then define table columns and create/update flags.

3. **teamId scope**
   - What we know: Required for PATCH. May be required for other operations.
   - What's unclear: Whether list/get/delete also require teamId.
   - Recommendation: Start with teamId only on update (where spec confirms it's required). Add to other commands if API returns errors.

4. **Model GET endpoint**
   - What we know: The spec does not explicitly show GET on `/v2/api/sources/ai/models/{id}`, but it does show GET on the collection.
   - What's unclear: Whether individual model GET works.
   - Recommendation: Try it -- RESTful convention strongly suggests it exists. The spec listing may be incomplete (it also didn't show collection GETs initially).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.11.1 |
| Config file | none (Go convention: `go test ./...`) |
| Quick run command | `go test ./cmd/models/... -v -count=1` |
| Full suite command | `make test` (`go test ./... -v -count=1`) |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AIMD-01 | List models renders table | unit | `go test ./cmd/models/... -run TestListModels -v -count=1` | No - Wave 0 |
| AIMD-01 | List models empty shows message | unit | `go test ./cmd/models/... -run TestListModelsEmpty -v -count=1` | No - Wave 0 |
| AIMD-01 | List models JSON output | unit | `go test ./cmd/models/... -run TestListModelsJSON -v -count=1` | No - Wave 0 |
| AIMD-02 | Get model by ID renders single row | unit | `go test ./cmd/models/... -run TestGetModel -v -count=1` | No - Wave 0 |
| AIMD-03 | Update model sends PATCH with changed fields | unit | `go test ./cmd/models/... -run TestUpdateModel -v -count=1` | No - Wave 0 |
| AIMD-03 | Update model includes teamId query param | unit | `go test ./cmd/models/... -run TestUpdateModelTeamId -v -count=1` | No - Wave 0 |
| AIMD-03 | Update model with no fields errors | unit | `go test ./cmd/models/... -run TestUpdateModelNoFields -v -count=1` | No - Wave 0 |
| AIMD-04 | Delete model with --yes | unit | `go test ./cmd/models/... -run TestDeleteModel -v -count=1` | No - Wave 0 |
| AIMD-04 | Delete model JSON mode auto-confirms | unit | `go test ./cmd/models/... -run TestDeleteModelJSON -v -count=1` | No - Wave 0 |
| AIMD-05 | List pricing dimensions for model | unit | `go test ./cmd/models/... -run TestPricingList -v -count=1` | No - Wave 0 |
| AIMD-05 | List pricing dimensions empty | unit | `go test ./cmd/models/... -run TestPricingListEmpty -v -count=1` | No - Wave 0 |
| AIMD-06 | Create pricing dimension | unit | `go test ./cmd/models/... -run TestPricingCreate -v -count=1` | No - Wave 0 |
| AIMD-07 | Update pricing dimension | unit | `go test ./cmd/models/... -run TestPricingUpdate -v -count=1` | No - Wave 0 |
| AIMD-08 | Delete pricing dimension | unit | `go test ./cmd/models/... -run TestPricingDelete -v -count=1` | No - Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/models/... -v -count=1`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/models/list_test.go` -- covers AIMD-01 (list with data + empty + JSON)
- [ ] `cmd/models/get_test.go` -- covers AIMD-02
- [ ] `cmd/models/update_test.go` -- covers AIMD-03 (PATCH, teamId, no-fields error)
- [ ] `cmd/models/delete_test.go` -- covers AIMD-04
- [ ] `cmd/models/pricing_list_test.go` -- covers AIMD-05
- [ ] `cmd/models/pricing_create_test.go` -- covers AIMD-06
- [ ] `cmd/models/pricing_update_test.go` -- covers AIMD-07
- [ ] `cmd/models/pricing_delete_test.go` -- covers AIMD-08

### Test Pattern (from existing sources tests)
```go
func TestListModels(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/v2/api/sources/ai/models", r.URL.Path)
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, `[
            {"id": "mdl-1", "name": "gpt-4o", "provider": "OpenAI", "mode": "chat"}
        ]`)
    }))
    defer srv.Close()

    var buf bytes.Buffer
    cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
    cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

    c := newListCmd()
    c.SetOut(&buf)
    err := c.Execute()

    require.NoError(t, err)
    assert.Contains(t, buf.String(), "mdl-1")
    assert.Contains(t, buf.String(), "gpt-4o")
    assert.Contains(t, buf.String(), "OpenAI")
}

func TestUpdateModelPATCH(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "PATCH", r.Method)
        assert.Equal(t, "team-1", r.URL.Query().Get("teamId"))
        // Verify body contains only changed fields
        var body map[string]interface{}
        json.NewDecoder(r.Body).Decode(&body)
        assert.Contains(t, body, "inputCostPerToken")
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, `{"id": "mdl-1", "name": "gpt-4o", "provider": "OpenAI", "mode": "chat"}`)
    }))
    defer srv.Close()

    var buf bytes.Buffer
    cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
    cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

    c := newUpdateCmd()
    c.SetOut(&buf)
    c.SetArgs([]string{"mdl-1", "--team-id", "team-1", "--input-cost-per-token", "0.003"})
    err := c.Execute()

    require.NoError(t, err)
}
```

## Sources

### Primary (HIGH confidence)
- Revenium OpenAPI spec at `https://api.dev.hcapp.io/profitstream/api-docs/v2` -- AI model endpoints, PATCH schema, pricing dimension endpoints
- Existing codebase: `cmd/sources/` -- established CRUD pattern, test patterns, shared helpers
- Existing codebase: `cmd/root.go` -- RegisterCommand, global flags, command groups
- Existing codebase: `internal/resource/resource.go` -- ConfirmDelete helper

### Secondary (MEDIUM confidence)
- AIModelResource_Read field list -- visible in spec schema section
- Collection GET endpoints -- inferred from spec patterns (confirmed for other resources)

### Tertiary (LOW confidence)
- PricingDimensionResource_Read fields -- not documented in spec, needs API discovery
- GET /v2/api/sources/ai/models/{modelId}/pricing/dimensions -- not documented, needs validation
- GET /v2/api/sources/ai/models/{id} (individual) -- not explicitly in spec but highly likely

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - using only existing dependencies, zero new libraries
- Architecture (models CRUD): HIGH - direct replication of sources pattern with PATCH instead of PUT
- Architecture (pricing nesting): HIGH - native Cobra subcommand pattern, straightforward
- API endpoints (models): HIGH - PATCH/DELETE confirmed, list/get highly likely
- API endpoints (pricing dimensions): MEDIUM - POST/PUT/DELETE confirmed, GET list unconfirmed
- Pricing dimension schema: LOW - fields not documented, requires API discovery
- Pitfalls: HIGH - based on direct API spec analysis and established codebase patterns

**Research date:** 2026-03-12
**Valid until:** 2026-04-12 (stable -- Go ecosystem and project dependencies settled)
