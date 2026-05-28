# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build (embeds version/commit/date via ldflags)
make build

# Run all tests
make test

# Run tests with race detector
make test-race

# Run a single test
go test ./cmd/organizations/... -run TestCreateOrganization -v

# Lint (requires golangci-lint)
make lint

# Snapshot release (requires goreleaser)
make release-dry
```

## Architecture

This is a Cobra CLI for the Revenium API. Commands are grouped into three categories: **resources** (CRUD operations), **monitoring** (metrics, metering), and **config**.

### Circular import solution

Resource packages (e.g. `cmd/organizations/`) must import `cmd` to access the shared `cmd.APIClient` and `cmd.Output` globals. This means `cmd/root.go` cannot import them in return. The solution is that `main.go` acts as the wiring layer — it imports all resource packages and calls `cmd.RegisterCommand(pkg.Cmd, "groupID")` in `init()`. Never import resource sub-packages from `cmd/root.go`.

### Shared globals in `cmd/`

Two package-level vars are initialized in `rootCmd.PersistentPreRunE` and used by all subcommands:

- `cmd.APIClient` — `*api.Client`, use for all HTTP calls
- `cmd.Output` — `*output.Formatter`, use for all rendering

Helper functions on `rootCmd`: `cmd.JSONMode()`, `cmd.YesMode()`, `cmd.DryRun()`.

### `internal/api` — HTTP client

Key methods on `*api.Client`:

- `Do(ctx, method, path, body, result)` — raw HTTP; appends `teamId`/`tenantId` as query params automatically
- `DoList(ctx, path, opts, &result)` — handles both plain arrays and Spring HATEOAS `_embedded` responses; `FetchAll` mode paginates automatically
- `DoCreate(ctx, path, body, &result)` — POST; auto-injects `teamId`/`tenantId` into body
- `DoCreateWithOwner(...)` — like `DoCreate` but also injects `ownerId`
- `DoUpdate(ctx, path, updates, &result)` — GET → merge → PUT; extracts flat IDs from nested objects automatically (e.g. `"team": {"id": "x"}` → `"teamId": "x"`)

### `internal/output` — Rendering

`output.Formatter` renders either styled tables (lipgloss) or JSON. In tests, always use `output.NewWithWriter(&buf, &buf, jsonMode, quiet)` instead of `output.New(...)` to capture output without TTY detection or colorprofile wrapping.

`output.TableDef` specifies column headers and which column index holds a status value (for color styling). Use `StatusColumn: -1` when there is no status column.

`Render(tableDef, rows, rawData)` — in table mode renders rows; in JSON mode renders `rawData`.

### `internal/dryrun`

`dryrun.Render(f, action, resource, path, body)` — renders a dry-run summary without making HTTP calls. All mutating commands must check `cmd.DryRun()` before calling the API.

### Patterns for adding a new resource command

1. Create `cmd/<resource>/` package with a `Cmd` var.
2. Use `c.Flags().Changed("flag-name")` to gate optional fields — never include a field in the request body unless the user explicitly passed the flag.
3. Mark mutating commands: `Annotations: map[string]string{"mutating": "true"}`.
4. Call `cmd.AddListFlags(c)` and `cmd.ListOptsFromFlags(c)` on list commands to get `--page` / `--page-size` support with auto-pagination in table mode.
5. Register in `main.go` via `cmd.RegisterCommand(pkg.Cmd, "resources")`.

### Testing pattern

Tests use `net/http/httptest.NewServer` for a real HTTP stub — no mocks. Set `cmd.APIClient` and `cmd.Output` directly in each test:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ... }))
defer srv.Close()
var buf bytes.Buffer
cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
cmd.Output = output.NewWithWriter(&buf, &buf, false, false)
```

### Configuration

Config file: `~/.config/revenium/config.yaml` (XDG path, not macOS `~/Library/Application Support`). Valid keys: `key`, `api-url`, `team-id`, `tenant-id`, `owner-id`. Environment variables (`REVENIUM_API_KEY`, `REVENIUM_API_URL`, `REVENIUM_TEAM_ID`, `REVENIUM_OUTPUT_FORMAT`) take precedence over the file.

### Exit codes

Defined in `internal/errors/exitcodes.go`: 0 success, 1 general, 2 auth, 3 not found, 4 validation, 5 network. In JSON mode, errors go to stderr as structured JSON including the exit code.
