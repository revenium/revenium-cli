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

| Key       | Description                    | Default                                      |
|-----------|--------------------------------|----------------------------------------------|
| `key`     | Your Revenium API key          | *(required)*                                 |
| `api-url` | API base URL                   | `https://api.revenium.ai/profitstream`       |
| `team-id` | Your team ID for multi-tenant access | *(optional)*                            |

```sh
revenium config set key your-api-key
revenium config set team-id your-team-id
revenium config set api-url https://custom.api.com/profitstream
revenium config show
```

### Environment Variables

Configuration can also be set via environment variables, which take precedence over the config file:

| Variable            | Overrides  |
|---------------------|------------|
| `REVENIUM_API_KEY`  | `key`      |
| `REVENIUM_API_URL`  | `api-url`  |
| `REVENIUM_TEAM_ID`  | `team-id`  |

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

## Output Formats

### Table (default)

Human-readable tables with styled borders, automatically sized to your terminal width.

### JSON

Use `--json` for machine-readable output, suitable for scripting and CI/CD pipelines:

```sh
revenium sources list --json
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

Pipe to `jq` for further processing:

```sh
revenium models list --json | jq '.[].name'
```

### Quiet Mode

Suppress non-error output with `--quiet` / `-q`. Useful in scripts where you only care about the exit code:

```sh
revenium sources delete src-123 --yes --quiet
```

## Global Flags

| Flag            | Short | Description                    |
|-----------------|-------|--------------------------------|
| `--verbose`     | `-v`  | Enable verbose output (shows HTTP requests/responses) |
| `--json`        |       | Output as JSON                 |
| `--quiet`       | `-q`  | Suppress non-error output      |
| `--yes`         | `-y`  | Skip confirmation prompts      |
| `--help`        | `-h`  | Help for any command           |

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

In JSON mode, errors are written to stderr as structured JSON:

```json
{"error": "Invalid API key. Run `revenium config set key <your-key>` to fix.", "status": 401}
```

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

## License

MIT
