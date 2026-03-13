# Revenium CLI — Agent Context

Machine-oriented reference for AI agents, scripts, and automation tools consuming the Revenium CLI.

## Authentication

Set your API key before use:

```bash
revenium config set key <your-api-key>
```

Or via environment variable:

```bash
export REVENIUM_API_KEY=<your-api-key>
```

Configuration is stored at `~/.revenium/config.json`.

## Output Modes

| Method | Example |
|---|---|
| `--output json` | `revenium sources list --output json` |
| `--output table` | `revenium sources list --output table` (default) |
| Environment variable | `REVENIUM_OUTPUT_FORMAT=json revenium sources list` |
| Legacy flag (hidden) | `revenium sources list --json` |

**Resolution order:** `--json` flag > `--output` flag > `REVENIUM_OUTPUT_FORMAT` env > default `table`.

### Field Filtering

Reduce response size with `--fields`:

```bash
revenium sources list --output json --fields id,name
revenium sources list --fields id,name,status   # works with table output too
```

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General / unknown error |
| 2 | Authentication error (401/403) |
| 3 | Resource not found (404) |
| 4 | Validation error (400/422) |
| 5 | Network / connection failure |

In JSON mode, error responses include the exit code:

```json
{
  "error": "Resource not found",
  "status": 404,
  "exit_code": 3
}
```

## Dry Run

Preview any mutation without executing it:

```bash
revenium sources create --name "Test" --type API --dry-run --output json
```

Returns the action, resource, API path, and request body without making the API call. Always use `--dry-run` for mutating operations when validating agent-generated commands.

## Programmatic Discovery

Dump the full command tree as JSON:

```bash
revenium schema
```

Returns all commands, flags (with types and defaults), required markers, mutating annotations, and exit code definitions. Use this for dynamic command discovery rather than parsing help text.

## Idiomatic Agent Usage

```bash
# Always use --output json for machine consumption
revenium sources list --output json

# Use --fields to minimize response size
revenium sources list --output json --fields id,name,status

# Preview mutations with --dry-run
revenium sources create --name "My API" --type API --dry-run --output json

# Execute mutations with --yes to skip confirmation prompts
revenium sources delete abc-123 --yes --output json

# Check exit codes for error handling
revenium sources get nonexistent-id --output json; echo $?
# Output: 3 (not found)

# Suppress non-error output with --quiet
revenium sources delete abc-123 --yes --quiet
```

## Resource Commands

All resources follow the same CRUD pattern:

```
revenium <resource> list [--page N] [--page-size N]
revenium <resource> get <id>
revenium <resource> create --name ... [--other-flags]
revenium <resource> update <id> --name ...
revenium <resource> delete <id> [--yes]
```

Resources: `sources`, `models`, `subscribers`, `subscriptions`, `products`, `tools`, `teams`, `users`, `anomalies`, `alerts`, `credentials`, `charts`.

Sub-resources: `alerts budget`, `models pricing`, `teams prompt-capture`.

## Stderr vs Stdout

- **stdout**: Data output (JSON or table)
- **stderr**: Errors, verbose logs, confirmation prompts

This separation allows safe piping: `revenium sources list --output json | jq '.[] | .id'`
