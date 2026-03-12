# Architecture Research

**Domain:** Go CLI tool wrapping a REST API (Revenium AI Economic Control platform)
**Researched:** 2026-03-11
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      User / Shell / Scripts                     │
├─────────────────────────────────────────────────────────────────┤
│                       Command Layer (Cobra)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ sources  │  │ models   │  │ metrics  │  │ config   │  ...   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
│       │              │              │              │             │
├───────┴──────────────┴──────────────┴──────────────┴────────────┤
│                      Output Layer (Rendering)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Table (Lip   │  │ JSON         │  │ Detail View  │          │
│  │   Gloss)     │  │ Serializer   │  │ (Glamour)    │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
├─────────────────────────────────────────────────────────────────┤
│                      API Client Layer                           │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  HTTP Client  (net/http + auth middleware)             │      │
│  │  ┌─────────┐  ┌──────────┐  ┌───────────┐            │      │
│  │  │ Request │  │ Response │  │  Error    │            │      │
│  │  │ Builder │  │ Decoder  │  │  Handler  │            │      │
│  │  └─────────┘  └──────────┘  └───────────┘            │      │
│  └───────────────────────────────────────────────────────┘      │
├─────────────────────────────────────────────────────────────────┤
│                      Configuration Layer                        │
│  ┌──────────────┐  ┌──────────────┐                             │
│  │ Config File  │  │ Env Vars     │                             │
│  │ (Viper/YAML) │  │ (overrides)  │                             │
│  └──────────────┘  └──────────────┘                             │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│              Revenium REST API (external)                        │
│              https://api.revenium.ai/profitstream                │
└─────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **main.go** | Bootstrap and entry point | Calls root command Execute(), sets version info |
| **Root command** | Global flags, PersistentPreRun config loading | `cmd/root.go` with Cobra rootCmd |
| **Resource commands** | Parse flags, validate input, call API client, render output | One package per resource domain (sources, models, etc.) |
| **API client** | HTTP communication, auth headers, request/response marshaling | `internal/api/` package with typed methods per resource |
| **Output renderer** | Format data as styled tables or JSON | `internal/output/` package wrapping Lip Gloss tables |
| **Config manager** | Load/save API key and base URL from file and env vars | `internal/config/` using Viper or direct YAML parsing |
| **Resource types** | Go structs matching API request/response shapes | `internal/types/` or `internal/models/` package |

## Recommended Project Structure

```
revenium-cli/
├── main.go                     # Entry point: calls cmd.Execute()
├── cmd/                        # Cobra command definitions
│   ├── root.go                 # Root command, global flags, PersistentPreRun
│   ├── sources/                # revenium sources [list|get|create|update|delete]
│   │   ├── sources.go          # Parent command
│   │   ├── list.go
│   │   ├── get.go
│   │   ├── create.go
│   │   ├── update.go
│   │   └── delete.go
│   ├── models/                 # revenium models [list|get|create|update|delete]
│   │   └── ...
│   ├── subscribers/            # revenium subscribers ...
│   │   └── ...
│   ├── subscriptions/          # revenium subscriptions ...
│   │   └── ...
│   ├── products/               # revenium products ...
│   │   └── ...
│   ├── tools/                  # revenium tools ...
│   │   └── ...
│   ├── teams/                  # revenium teams ...
│   │   └── ...
│   ├── users/                  # revenium users ...
│   │   └── ...
│   ├── metrics/                # revenium metrics [ai|completion|audio|api|...]
│   │   └── ...
│   ├── alerts/                 # revenium alerts ...
│   │   └── ...
│   ├── anomalies/              # revenium anomalies ...
│   │   └── ...
│   ├── credentials/            # revenium credentials ...
│   │   └── ...
│   ├── invoices/               # revenium invoices ...
│   │   └── ...
│   ├── config.go               # revenium config [set|get|show]
│   └── version.go              # revenium version
├── internal/                   # Private application packages
│   ├── api/                    # REST API client
│   │   ├── client.go           # HTTP client, auth, base URL, error handling
│   │   ├── sources.go          # Source resource API methods
│   │   ├── models.go           # Model resource API methods
│   │   ├── metrics.go          # Metrics query API methods
│   │   └── ...                 # One file per resource domain
│   ├── config/                 # Configuration management
│   │   └── config.go           # Load/save ~/.revenium/config.yaml, env var override
│   ├── output/                 # Output formatting
│   │   ├── table.go            # Lip Gloss table rendering
│   │   ├── json.go             # JSON output (--json flag)
│   │   ├── detail.go           # Single-resource detail view
│   │   └── styles.go           # Shared Lip Gloss style definitions
│   └── types/                  # API resource types (Go structs)
│       ├── source.go
│       ├── model.go
│       ├── subscriber.go
│       └── ...
├── go.mod
├── go.sum
├── .goreleaser.yaml            # Cross-platform release config
└── Makefile                    # Build, test, lint targets
```

### Structure Rationale

- **`cmd/` with one package per resource:** Mirrors the CLI command hierarchy directly. Each resource domain is self-contained. Adding a new resource means adding one directory. This is the pattern used by gh, kubectl, and other mature Go CLIs.
- **`internal/api/` as a single client package:** All HTTP logic in one place. Commands never construct HTTP requests directly. The client handles auth headers, base URL, error parsing, and response deserialization. One file per resource domain keeps files manageable without over-nesting.
- **`internal/output/` for all rendering:** Commands call `output.Table(data)` or `output.JSON(data)` based on the `--json` flag. This keeps rendering logic out of command files and makes it trivial to maintain consistent styling across 15+ resource types.
- **`internal/types/` for API structs:** Shared between `api/` (deserialization) and `output/` (rendering). Keeping them separate from both avoids circular imports. Structs map to the API's read/write resource variants.
- **`internal/config/` for configuration:** Isolates the config file format, location, and env var override logic. Commands access config through a simple interface, never reading files directly.

## Architectural Patterns

### Pattern 1: Factory / Options Pattern for Commands

**What:** Each command subpackage exports a `NewCmd*()` function that receives shared dependencies (API client, output writer, config) and returns a configured `*cobra.Command`.
**When to use:** Every command. This is the standard Cobra pattern.
**Trade-offs:** Slightly more boilerplate per command, but enables testability and clean dependency flow.

**Example:**
```go
// cmd/sources/list.go
package sources

import (
    "github.com/spf13/cobra"
    "github.com/revenium/revenium-cli/internal/api"
    "github.com/revenium/revenium-cli/internal/output"
)

type ListOptions struct {
    Client   *api.Client
    Output   *output.Writer
    JSONMode bool
}

func NewCmdList(client *api.Client, out *output.Writer) *cobra.Command {
    opts := &ListOptions{
        Client: client,
        Output: out,
    }
    cmd := &cobra.Command{
        Use:   "list",
        Short: "List all sources",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runList(opts)
        },
    }
    cmd.Flags().BoolVar(&opts.JSONMode, "json", false, "Output as JSON")
    return cmd
}

func runList(opts *ListOptions) error {
    sources, err := opts.Client.ListSources()
    if err != nil {
        return err
    }
    if opts.JSONMode {
        return opts.Output.JSON(sources)
    }
    return opts.Output.SourceTable(sources)
}
```

### Pattern 2: Centralized API Client with Resource Methods

**What:** A single `api.Client` struct holds the HTTP client, base URL, and API key. Resource-specific methods are organized into separate files but all operate on the same client.
**When to use:** Always for REST API wrappers. This is the gh CLI pattern.
**Trade-offs:** Simple and direct. For 15+ resources, the client surface area grows, but file-per-resource keeps it organized. Avoid splitting into sub-clients unless you exceed 30+ resources.

**Example:**
```go
// internal/api/client.go
package api

import (
    "net/http"
    "encoding/json"
)

type Client struct {
    BaseURL    string
    APIKey     string
    HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        BaseURL:    baseURL,
        APIKey:     apiKey,
        HTTPClient: &http.Client{},
    }
}

func (c *Client) do(method, path string, body, result interface{}) error {
    // Build request, set x-api-key header, execute, decode response
    // Centralized error handling for 401, 403, 404, 500, etc.
}

// internal/api/sources.go
func (c *Client) ListSources() ([]types.Source, error) { ... }
func (c *Client) GetSource(id string) (*types.Source, error) { ... }
func (c *Client) CreateSource(s *types.SourceWrite) (*types.Source, error) { ... }
func (c *Client) UpdateSource(id string, s *types.SourceWrite) (*types.Source, error) { ... }
func (c *Client) DeleteSource(id string) error { ... }
```

### Pattern 3: Output Writer with Dual Mode (Styled / JSON)

**What:** A single output abstraction that every command uses. It checks the `--json` flag and routes to either Lip Gloss table rendering or JSON serialization. Styling definitions are centralized.
**When to use:** Every command that displays data.
**Trade-offs:** Small upfront investment, massive consistency payoff across 15+ resource types. Keeps command files focused on logic, not formatting.

**Example:**
```go
// internal/output/writer.go
package output

import (
    "encoding/json"
    "os"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/lipgloss/table"
)

type Writer struct {
    JSONMode bool
}

func (w *Writer) RenderTable(headers []string, rows [][]string) error {
    if w.JSONMode {
        // Marshal rows as JSON array and write to stdout
    }
    t := table.New().
        Border(lipgloss.ThickBorder()).
        BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
        Headers(headers...).
        Rows(rows...)
    // Apply header and row styling
    fmt.Println(t)
    return nil
}
```

### Pattern 4: Config Precedence Chain

**What:** Configuration values resolve in order: CLI flags > environment variables > config file > defaults. Cobra's PersistentPreRunE on the root command loads config before any subcommand runs.
**When to use:** Root command setup.
**Trade-offs:** Viper adds a dependency but handles this precedence natively with Cobra flag binding. For just two config values (API key + base URL), you could also hand-roll this with a simple YAML parser + os.Getenv, avoiding the Viper dependency. Use Viper if you expect config to grow; skip it if you want minimal dependencies.

## Data Flow

### Command Execution Flow

```
User types: revenium sources list --json
    |
    v
main.go -> cmd.Execute()
    |
    v
Cobra parses "sources list --json"
    |
    v
root.PersistentPreRunE:
    - Load ~/.revenium/config.yaml
    - Check REVENIUM_API_KEY env var (overrides file)
    - Check REVENIUM_API_URL env var (overrides file)
    - Initialize api.Client with resolved config
    |
    v
cmd/sources/list.go -> runList(opts)
    |
    v
opts.Client.ListSources()
    - GET /profitstream/sources
    - x-api-key: <key>
    - Decode JSON response -> []types.Source
    |
    v
opts.Output.RenderTable() or opts.Output.JSON()
    - --json flag: marshal to JSON, write to stdout
    - default: build Lip Gloss table, write to stdout
    |
    v
stdout -> user's terminal or pipe
```

### Configuration Resolution Flow

```
CLI flags (--api-key, --api-url)
    |  (highest priority)
    v
Environment variables (REVENIUM_API_KEY, REVENIUM_API_URL)
    |
    v
Config file (~/.revenium/config.yaml)
    |
    v
Defaults (api_url: https://api.revenium.ai/profitstream)
    |  (lowest priority)
    v
Resolved config passed to api.Client
```

### Key Data Flows

1. **CRUD command flow:** User input (flags/args) -> command validation -> API client method -> HTTP request to Revenium API -> JSON response -> Go struct -> output renderer -> stdout
2. **Config command flow:** User runs `revenium config set api-key <key>` -> writes to `~/.revenium/config.yaml` -> subsequent commands read this file
3. **Error flow:** API returns non-2xx -> client.do() parses error body -> returns typed error -> command prints styled error message to stderr -> exits with non-zero code

## Anti-Patterns

### Anti-Pattern 1: HTTP Logic in Command Files

**What people do:** Build HTTP requests, set headers, and parse responses directly inside Cobra RunE functions.
**Why it's wrong:** Duplicates auth/error handling across every command. Makes testing commands require HTTP mocking. Changes to the API (URL paths, auth headers) require touching every command file.
**Do this instead:** All HTTP logic lives in `internal/api/`. Commands only call typed methods like `client.ListSources()`.

### Anti-Pattern 2: Rendering Logic Mixed with Business Logic

**What people do:** Build Lip Gloss tables inline within command RunE functions, with styling scattered across 50+ command files.
**Why it's wrong:** Inconsistent styling, duplicated table setup code, hard to maintain the `--json` flag behavior uniformly.
**Do this instead:** Centralize in `internal/output/`. Commands call `output.Table()` or `output.JSON()`. Style definitions live in one `styles.go` file.

### Anti-Pattern 3: One Giant cmd/root.go Registering All Commands

**What people do:** Define all 50+ commands in root.go or a single commands file.
**Why it's wrong:** File becomes unmaintainable. Merge conflicts in teams. Hard to find commands. No encapsulation.
**Do this instead:** Each resource domain gets its own package under `cmd/`. Root command only registers top-level parent commands. Parent commands register their own subcommands.

### Anti-Pattern 4: Skipping Error Type Differentiation

**What people do:** Return generic `fmt.Errorf("request failed")` for all API errors.
**Why it's wrong:** Users can't distinguish between auth failures (bad API key), not-found (wrong ID), validation errors (bad input), and server errors (API is down). Scripts can't handle errors programmatically.
**Do this instead:** Parse API error responses into typed errors. Display actionable messages: "Authentication failed -- check your API key with `revenium config show`" vs "Source 'abc123' not found".

### Anti-Pattern 5: Premature Interface Abstraction

**What people do:** Define interfaces for everything before writing any implementation (ClientInterface, RendererInterface, ConfigInterface).
**Why it's wrong:** Go idiom is "accept interfaces, return structs." Over-abstracting a CLI tool with one API backend, one output target, and one config source adds complexity with no benefit.
**Do this instead:** Use concrete types. Extract interfaces only when you actually need them for testing (e.g., an `HTTPDoer` interface to mock the HTTP client in tests).

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Revenium REST API | HTTP client with `x-api-key` header | Base URL configurable; ~15 resource endpoints + metric endpoints |
| File system (`~/.revenium/`) | YAML config file read/write | Create directory on first `config set` if it doesn't exist |
| Homebrew tap | goreleaser publishes tap formula | Separate GitHub repo for the tap |
| GitHub Releases | goreleaser creates release artifacts | Multi-platform binaries (darwin/linux, amd64/arm64) |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| cmd/* -> internal/api | Direct function calls | Commands receive `*api.Client` via dependency injection from root |
| cmd/* -> internal/output | Direct function calls | Commands receive `*output.Writer` or call package-level functions |
| internal/api -> internal/types | Import types for marshal/unmarshal | Types package has zero dependencies on api or cmd |
| internal/output -> internal/types | Import types for rendering | Output renders typed structs into tables or JSON |
| root cmd -> internal/config | PersistentPreRunE loads config | Config is resolved once, passed down to api.Client |

## Suggested Build Order

The dependency graph dictates a natural bottom-up build order:

```
Phase 1: Foundation (no external dependencies between components)
    internal/types/     <- API struct definitions
    internal/config/    <- Config loading
    internal/api/       <- HTTP client core (client.go with do() method)
    internal/output/    <- Table + JSON rendering

Phase 2: First Resource (proves the full vertical slice)
    internal/api/sources.go   <- One resource's API methods
    cmd/root.go               <- Root command with config loading
    cmd/sources/              <- Full CRUD commands for sources
    cmd/config.go             <- Config management command
    cmd/version.go            <- Version command

Phase 3: Remaining Resources (repeat the pattern)
    Each resource is independent -- can be built in any order
    models, subscribers, subscriptions, products, tools,
    teams, users, anomalies, alerts, credentials, charts

Phase 4: Specialized Commands
    cmd/metrics/              <- Multiple metric types, different query patterns
    cmd/invoices/             <- Billing operations
    Traces, squad metrics

Phase 5: Distribution
    .goreleaser.yaml          <- Release configuration
    Homebrew tap setup
    CI/CD pipeline
```

**Why this order:**
- Phase 1 builds the reusable foundation that every command depends on
- Phase 2 proves the entire pattern end-to-end with one resource before committing to 15+
- Phase 3 is parallelizable and mechanical once the pattern is proven
- Phase 4 tackles the non-CRUD endpoints that may need different patterns (query params, date ranges)
- Phase 5 is independent of feature work

## Sources

- [GitHub CLI architecture analysis](https://www.augmentcode.com/open-source/cli/cli) - gh project structure and factory pattern (MEDIUM confidence - third-party analysis of open source project)
- [Cobra CLI framework](https://cobra.dev/) - Official Cobra documentation (HIGH confidence)
- [Go CLI architecture example](https://github.com/skport/golang-cli-architecture) - Layered architecture reference (MEDIUM confidence)
- [Structuring Go CLI applications](https://www.bytesizego.com/blog/structure-go-cli-app) - Project structure best practices (MEDIUM confidence)
- [Lip Gloss table package](https://pkg.go.dev/github.com/charmbracelet/lipgloss/table) - Official table rendering docs (HIGH confidence)
- [Cobra + Viper integration](https://cobra.dev/docs/tutorials/12-factor-app/) - Config management patterns (HIGH confidence)
- [Go project structure patterns](https://www.glukhov.org/post/2025/12/go-project-structure/) - Community best practices (MEDIUM confidence)

---
*Architecture research for: Go CLI wrapping Revenium REST API*
*Researched: 2026-03-11*
