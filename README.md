# Revenium CLI

The official command-line interface for [Revenium](https://revenium.ai) — the AI Economic Control platform. Manage sources, models, subscriptions, alerts, and metrics from your terminal.

```
$ revenium sources list
╭──────────┬───────────────┬───────┬──────────╮
│ ID       │ Name          │ Type  │ Status   │
├──────────┼───────────────┼───────┼──────────┤
│ vKab65   │ Default       │ AI    │ active   │
│ x9Rt42   │ Chat Gateway  │ API   │ active   │
╰──────────┴───────────────┴───────┴──────────╯
```

## Installation

See [INSTALL.md](INSTALL.md) for detailed installation instructions.

**Quick install (macOS):**

```sh
brew install revenium/tap/revenium
```

**Quick install (from source):**

```sh
go install github.com/revenium/revenium-cli@latest
```

## Quick Start

```sh
# 1. Set your API key (from https://app.revenium.ai)
revenium config set key your-api-key

# 2. Set your team ID
revenium config set team-id your-team-id

# 3. Verify configuration
revenium config show

# 4. Start using the CLI
revenium sources list
revenium models list
revenium metrics ai
```

## Configuration

The CLI stores configuration in `~/.config/revenium/config.yaml`.

| Key         | Description                          | Default                                |
|-------------|--------------------------------------|----------------------------------------|
| `key`       | Your Revenium API key                | *(required)*                           |
| `api-url`   | API base URL                         | `https://api.revenium.ai/profitstream` |
| `team-id`   | Team ID for multi-tenant access      | *(optional)*                           |
| `tenant-id` | Tenant ID                            | *(optional)*                           |
| `owner-id`  | Owner ID                             | *(optional)*                           |

```sh
revenium config set key your-api-key
revenium config set team-id your-team-id
revenium config set tenant-id your-tenant-id
revenium config set owner-id your-owner-id
revenium config set api-url https://custom.api.com/profitstream
revenium config show
```

### Environment Variables

Configuration can also be set via environment variables, which take precedence over the config file:

| Variable                | Overrides      |
|-------------------------|----------------|
| `REVENIUM_API_KEY`      | `key`          |
| `REVENIUM_API_URL`      | `api-url`      |
| `REVENIUM_TEAM_ID`      | `team-id`      |
| `REVENIUM_OUTPUT_FORMAT` | `--output` flag (set to `json` or `table`) |

## Commands

### Core Resources

Manage the primary resources in your Revenium account. All resource commands follow a consistent CRUD pattern: `list`, `get`, `create`, `update`, `delete`.

| Command           | Description                              |
|-------------------|------------------------------------------|
| `sources`         | Manage sources (APIs, AI services)       |
| `models`          | Manage AI models and pricing dimensions  |
| `products`        | Manage products                          |
| `subscribers`     | Manage subscribers                       |
| `subscriptions`   | Manage subscriptions                     |
| `tools`           | Manage tools                             |
| `teams`           | Manage teams and prompt capture settings |
| `users`           | Manage users                             |
| `anomalies`       | Manage AI anomaly detection rules        |
| `alerts`          | Manage AI alerts and budget thresholds   |
| `credentials`     | Manage provider credentials              |
| `charts`          | Manage chart definitions                 |

**Examples:**

```sh
# List all AI models
revenium models list

# Get a specific source
revenium sources get vKab65

# Create a new product
revenium products create --name "Enterprise Plan" --description "Full access"

# Update a subscriber
revenium subscribers update sub-123 --first-name "Jane" --last-name "Doe"

# Delete a tool (with confirmation prompt)
revenium tools delete tool-456

# Delete without confirmation
revenium tools delete tool-456 --yes
```

#### Model Pricing

Manage pricing dimensions for AI models:

```sh
revenium models pricing list <model-id>
revenium models pricing create <model-id> --name "Input Tokens" --type TOKEN --price 0.003
revenium models pricing update <model-id> <dimension-id> --price 0.005
revenium models pricing delete <model-id> <dimension-id>
```

#### Budget Alerts

Track and manage AI spending budgets:

```sh
revenium alerts budget list
revenium alerts budget get <alert-id>
revenium alerts budget create <anomaly-id> --threshold 100.00
revenium alerts budget update <anomaly-id> --threshold 200.00
```

#### Team Prompt Capture

Configure prompt capture settings for a team:

```sh
revenium teams prompt-capture get <team-id>
revenium teams prompt-capture set <team-id> --enabled true
```

### Monitoring

Query metrics and analytics across your AI infrastructure.

```sh
# AI metrics (defaults to last 24 hours)
revenium metrics ai

# Completion metrics with custom time range
revenium metrics completions --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z

# Other metric types
revenium metrics audio
revenium metrics image
revenium metrics video
revenium metrics traces
revenium metrics squads
revenium metrics api
revenium metrics tool-events
```

All metrics commands support `--from` and `--to` flags for filtering by time range (ISO 8601 format).

### Metering

Submit usage events to Revenium's metering API for tracking and billing.

| Subcommand     | Description                              |
|----------------|------------------------------------------|
| `event`        | Meter a generic event with custom payload |
| `api-request`  | Meter an API request                     |
| `api-response` | Meter an API response                    |
| `completion`   | Meter an AI completion                   |
| `image`        | Meter an AI image operation              |
| `audio`        | Meter an AI audio operation              |
| `video`        | Meter an AI video operation              |
| `tool-event`   | Meter a tool/function call               |

**Examples:**

```sh
# Meter a generic usage event
revenium meter event --transaction-id txn-123 --payload '{"apiCalls": 100, "storageGB": 15.5}'

# Meter an AI completion
revenium meter completion --model gpt-4 --provider openai \
  --input-tokens 500 --output-tokens 200 --total-tokens 700 \
  --stop-reason END --is-streamed \
  --request-time 2024-01-15T10:00:00Z \
  --completion-start-time 2024-01-15T10:00:01Z \
  --response-time 2024-01-15T10:00:05Z \
  --request-duration 5000

# Meter a completion with prompt/response content
revenium meter completion --model gpt-4 --provider openai \
  --input-tokens 500 --output-tokens 200 --total-tokens 700 \
  --stop-reason END --is-streamed \
  --request-time 2024-01-15T10:00:00Z \
  --completion-start-time 2024-01-15T10:00:01Z \
  --response-time 2024-01-15T10:00:05Z \
  --request-duration 5000 \
  --system-prompt "You are a helpful assistant" \
  --input-messages '[{"role":"user","content":"Hello"}]' \
  --output-response "Hi there! How can I help you?"

# Meter an AI image generation
revenium meter image --model dall-e-3 --provider openai \
  --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:00:05Z \
  --request-duration 5000 --actual-image-count 1 --billing-unit PER_IMAGE

# Meter an API request/response pair
revenium meter api-request --transaction-id txn-456 --method POST --resource /api/users
revenium meter api-response --transaction-id txn-456 --response-code 200 --total-duration 150

# Meter an audio transcription
revenium meter audio --model whisper-1 --provider openai \
  --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:00:10Z \
  --request-duration 10000 --billing-unit PER_SECOND --duration-seconds 120

# Meter a video generation
revenium meter video --model veo --provider google \
  --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:01:00Z \
  --request-duration 60000 --duration-seconds 10 --billing-unit PER_SECOND

# Meter a tool call
revenium meter tool-event --tool-id search-api --duration-ms 150 --success \
  --timestamp 2024-01-15T10:00:00Z

# Preview a metering event without submitting
revenium meter completion --model gpt-4 --provider openai \
  --input-tokens 500 --output-tokens 200 --total-tokens 700 \
  --stop-reason END --is-streamed \
  --request-time 2024-01-15T10:00:00Z \
  --completion-start-time 2024-01-15T10:00:01Z \
  --response-time 2024-01-15T10:00:05Z \
  --request-duration 5000 --dry-run
```

All metering commands support optional fields for cost tracking (`--total-cost`), organizational attribution (`--agent`, `--environment`, `--organization-name`, `--product-name`), distributed tracing (`--transaction-id`, `--trace-id`), and conversation content (`--system-prompt`, `--input-messages`, `--output-response`). Use `revenium meter <subcommand> --help` for the full list of flags.

## Output Formats

### Table (default)

Human-readable tables with styled borders, automatically sized to your terminal width.

### JSON

Use `--output json` (or the legacy `--json` flag) for machine-readable output, suitable for scripting and CI/CD pipelines:

```sh
revenium sources list --output json
```

```json
[
  {
    "id": "vKab65",
    "name": "Default",
    "type": "AI",
    "status": "active"
  }
]
```

You can also set the output format via environment variable to avoid passing the flag on every call:

```sh
export REVENIUM_OUTPUT_FORMAT=json
revenium sources list
```

**Resolution order:** `--json` flag > `--output` flag > `REVENIUM_OUTPUT_FORMAT` env > default `table`.

Pipe to `jq` for further processing:

```sh
revenium models list --output json | jq '.[].name'
```

### Field Filtering

Use `--fields` to limit the fields included in output. Works with both JSON and table modes:

```sh
# Only return id and name in JSON output
revenium sources list --output json --fields id,name

# Filter table columns
revenium sources list --fields id,name,status
```

### Quiet Mode

Suppress non-error output with `--quiet` / `-q`. Useful in scripts where you only care about the exit code:

```sh
revenium sources delete src-123 --yes --quiet
```

## Dry Run

Preview any mutation (create, update, delete) without executing it using `--dry-run`. This is useful for validating commands before they make changes:

```sh
# See what would be sent to the API without creating anything
revenium sources create --name "My API" --type API --dry-run

# Combine with --output json for structured preview
revenium sources create --name "My API" --type API --dry-run --output json
```

```json
{
  "dry_run": true,
  "action": "create",
  "resource": "source",
  "path": "/v2/api/sources",
  "body": {
    "name": "My API",
    "type": "API",
    "version": "1.0.0"
  }
}
```

## Schema Introspection

Dump the full CLI command tree as machine-readable JSON with `revenium schema`. This is useful for programmatic discovery by AI agents, scripts, and automation tools:

```sh
revenium schema
```

The output includes all commands with their flags (types, defaults, required markers), mutating annotations, and exit code definitions.

## Global Flags

| Flag            | Short | Description                                           |
|-----------------|-------|-------------------------------------------------------|
| `--output`      |       | Output format: `json` or `table` (default `table`)    |
| `--fields`      |       | Comma-separated list of fields to include in output   |
| `--dry-run`     |       | Preview the action without executing it                |
| `--verbose`     | `-v`  | Enable verbose output (shows HTTP requests/responses)  |
| `--quiet`       | `-q`  | Suppress non-error output                              |
| `--yes`         | `-y`  | Skip confirmation prompts                              |
| `--help`        | `-h`  | Help for any command                                   |

## Shell Completions

Tab completion is available for bash, zsh, and fish. If installed via Homebrew, completions are set up automatically.

For manual setup:

```sh
# Bash
revenium completion bash > /etc/bash_completion.d/revenium

# Zsh
revenium completion zsh > "${fpath[1]}/_revenium"

# Fish
revenium completion fish > ~/.config/fish/completions/revenium.fish
```

## Verbose Mode

Use `--verbose` / `-v` to see HTTP request and response details. API keys are automatically masked in verbose output.

```
$ revenium sources list -v
> GET https://api.revenium.ai/profitstream/v2/api/sources?teamId=abc123
> x-api-key: ****7f3f
< 200 OK
╭──────────┬───────────────┬───────┬──────────╮
│ ID       │ Name          │ Type  │ Status   │
...
```

## Error Handling

The CLI provides clear, actionable error messages:

- **Invalid API key** — prompts you to run `revenium config set key`
- **Access denied** — indicates insufficient permissions
- **Resource not found** — the requested resource doesn't exist
- **Server errors** — suggests retrying or contacting support
- **Network errors** — prompts you to check connectivity

### Exit Codes

The CLI uses semantic exit codes for scripting and automation:

| Code | Meaning                          |
|------|----------------------------------|
| 0    | Success                          |
| 1    | General / unknown error          |
| 2    | Authentication error (401/403)   |
| 3    | Resource not found (404)         |
| 4    | Validation error (400/422)       |
| 5    | Network / connection failure     |

```sh
revenium sources get nonexistent-id; echo $?
# 3
```

In JSON mode, errors are written to stderr as structured JSON including the exit code:

```json
{
  "error": "Resource not found",
  "status": 404,
  "exit_code": 3
}
```

### Input Validation

Resource IDs are validated before any API call is made. IDs containing control characters, query parameters (`?`, `&`, `#`), path traversal sequences (`../`), or percent-encoded values (`%xx`) are rejected with a clear error message.

## Building from Source

```sh
git clone https://github.com/revenium/revenium-cli.git
cd revenium-cli
go build -o revenium .
./revenium version
```

To build with version information:

```sh
go build -ldflags "-X github.com/revenium/revenium-cli/internal/build.Version=1.0.0" -o revenium .
```

## Running Tests

```sh
go test ./...
```

## AI Agent Integration

For AI agents and automation tools, see [CONTEXT.md](CONTEXT.md) for a machine-oriented reference covering authentication, output modes, exit codes, dry-run usage, and `revenium schema` for programmatic command discovery.

## License

MIT
