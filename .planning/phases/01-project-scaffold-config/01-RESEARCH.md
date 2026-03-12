# Phase 1: Project Scaffold & Config - Research

**Researched:** 2026-03-11
**Domain:** Go CLI scaffold with Cobra, Viper config, HTTP client, Lip Gloss error styling
**Confidence:** HIGH

## Summary

Phase 1 establishes the foundational Go binary: module initialization, Cobra command tree with root + config + version commands, YAML config management at `~/.config/revenium/config.yaml`, an HTTP client with `x-api-key` auth, centralized error handling with Lip Gloss styled error boxes, and build-time version embedding. This phase creates the infrastructure that every subsequent phase builds on.

The stack is well-established and well-documented. Cobra v1.10.2 + Viper v1.21.0 is the standard Go CLI combination. The primary complexity lies in getting Viper config precedence right (flags > env vars > config file > defaults), properly handling the `config set` write path (Viper does not create directories), and building the HTTP client with proper timeouts and response body cleanup from day one.

**Primary recommendation:** Build bottom-up: config package first, then API client, then error handling/styling, then Cobra commands (root, config, version). Every subsequent phase inherits these patterns.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Noun-verb pattern: `revenium sources list`, `revenium models get abc-123`
- Plural resource names: `sources`, `models`, `subscriptions` (not singular)
- Subcommand nesting for child resources: `revenium models pricing list <model-id>`
- Resource IDs as positional arguments: `revenium sources get abc-123` (not `--id`)
- Create/update via flags: `revenium sources create --name "My API" --type rest`
- Top-level utility commands: `config` and `version` only
- Root help groups commands by category (Core Resources, Monitoring, Configuration)
- Config file at `~/.config/revenium/config.yaml` (XDG standard)
- Subcommands: `revenium config set key <val>`, `revenium config set api-url <val>`, `revenium config show`
- Default API URL baked in: `https://api.revenium.ai/profitstream` -- most users never change it
- No config -> clear error: "No API key configured. Run `revenium config set key <your-key>` to fix."
- No interactive setup wizard -- error + guidance approach
- Helpful + concise tone: "Error: Invalid API key. Run `revenium config set key <your-key>` to fix."
- Full Lip Gloss styled error box with border -- distinctive visual treatment
- Network failures: "Could not connect to api.revenium.ai. Check your network connection."
- `--verbose` shows full HTTP context on errors: method, URL, status code, response body
- Cobra default help template with examples section
- 2-3 examples per command covering common use cases
- Root help groups commands by category
- No branding/tagline -- `revenium version` shows clean `revenium v1.0.0 (abc1234)` only

### Claude's Discretion
- Go module structure and package organization
- Cobra initialization patterns
- HTTP client timeout values and retry behavior
- Exact Lip Gloss error box styling
- Config file YAML structure details

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FNDN-01 | CLI binary named `revenium` with Cobra-based command structure and root help | Cobra v1.10.2 with AddGroup for categorized help; factory pattern for commands |
| FNDN-02 | Config file at `~/.config/revenium/config.yaml` storing API key and API URL | Viper v1.21.0 with manual path construction (os.UserHomeDir + /.config/revenium); NOT os.UserConfigDir which returns ~/Library/Application Support on macOS |
| FNDN-03 | `revenium config set key <value>` and `revenium config set api-url <value>` commands | Viper Set() + WriteConfigAs(); must os.MkdirAll the config directory before writing |
| FNDN-04 | Environment variable override (`REVENIUM_API_KEY`, `REVENIUM_API_URL`) taking precedence over config file | Viper SetEnvPrefix("REVENIUM") + AutomaticEnv() + BindEnv(); read via viper.GetString(), never flag vars |
| FNDN-05 | HTTP client with x-api-key auth header, proper timeouts, and response body cleanup | net/http stdlib with configured Client{Timeout: 30s}; centralized do() method with defer resp.Body.Close() and io.Copy(io.Discard) drain |
| FNDN-06 | Helpful error messages mapping HTTP status codes to actionable guidance | Centralized APIError type; status-to-message map (401->invalid key, 403->forbidden, 404->not found, 5xx->server error) |
| FNDN-07 | Non-zero exit codes on all error paths | Cobra SilenceErrors=true, SilenceUsage=true on root; os.Exit(1) in main.go when Execute() returns error |
| FNDN-12 | `revenium version` command with build-time version/commit/date embedding | Package-level vars set via ldflags: -X 'main.version=...' -X 'main.commit=...' -X 'main.date=...' |
| FNDN-13 | `--help` with usage examples on every command | Cobra Example field on each command; 2-3 examples per command |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.24+ | Language | Required by Charm v2 libraries (Glamour v2 needs Go 1.24+) |
| Cobra | v1.10.2 | Command framework | Industry standard. Used by kubectl, docker, gh, hugo. Provides subcommand routing, flag parsing, help generation, command groups. |
| Viper | v1.21.0 | Configuration | Natural Cobra companion. Reads YAML, supports env var overrides, handles config precedence. |
| Lip Gloss | v2.0.2 | Styled output | Error box styling with borders. Import: `charm.land/lipgloss/v2`. Stable since Mar 2025. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testify | v1.10.0+ | Test assertions | `require` and `assert` packages for all tests |
| net/http (stdlib) | stdlib | HTTP client | All API communication. Thin wrapper, no external dependency. |
| encoding/json (stdlib) | stdlib | JSON handling | API response parsing and `--json` output |
| gopkg.in/yaml.v3 | v3 | YAML config | Pulled in by Viper automatically |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Viper | Direct yaml.v3 + os.Getenv | Simpler for just 2 config values, but loses Cobra flag binding and env var prefix support |
| net/http | go-resty v2.17.2 | Unnecessary dependency for simple REST with one auth header |

**Installation:**
```bash
go mod init github.com/revenium/revenium-cli
go get github.com/spf13/cobra@v1.10.2
go get github.com/spf13/viper@v1.21.0
go get charm.land/lipgloss/v2@v2.0.2
go get github.com/stretchr/testify@latest
```

## Architecture Patterns

### Recommended Project Structure (Phase 1 scope)
```
revenium-cli/
├── main.go                     # Entry point: calls cmd.Execute(), version vars
├── cmd/                        # Cobra command definitions
│   ├── root.go                 # Root command, PersistentPreRunE, global flags
│   ├── version.go              # revenium version
│   └── config/                 # revenium config [set|show]
│       ├── config.go           # Parent config command
│       ├── set.go              # revenium config set key|api-url <value>
│       └── show.go             # revenium config show
├── internal/
│   ├── api/                    # HTTP client wrapper
│   │   └── client.go           # Client struct, do() method, auth, error handling
│   ├── config/                 # Configuration management
│   │   └── config.go           # Load/save ~/.config/revenium/config.yaml, env var override
│   ├── errors/                 # Error types and formatting
│   │   └── errors.go           # APIError type, Lip Gloss error box rendering
│   └── build/                  # Build-time version info
│       └── build.go            # Version, Commit, Date variables
├── go.mod
├── go.sum
└── Makefile                    # Build, test, lint targets
```

### Pattern 1: Config Loading via PersistentPreRunE

**What:** Root command's `PersistentPreRunE` loads config before any subcommand runs. Config commands skip API client initialization since they don't need it.
**When to use:** Root command setup.

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "revenium",
    Short: "Manage your Revenium account",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Skip config loading for config and version commands
        if cmd.Name() == "version" || cmd.Parent().Name() == "config" {
            return nil
        }
        return initConfig(cmd)
    },
}

func initConfig(cmd *cobra.Command) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    if cfg.APIKey == "" {
        return fmt.Errorf("No API key configured. Run `revenium config set key <your-key>` to fix.")
    }
    // Store client in command context or package-level var
    return nil
}
```

### Pattern 2: Config Path Construction (XDG on all platforms)

**What:** Use `os.UserHomeDir()` + `/.config/revenium` to get XDG-compliant path. Do NOT use `os.UserConfigDir()` which returns `~/Library/Application Support` on macOS.
**When to use:** Config package initialization.

```go
// internal/config/config.go
func configDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("cannot determine home directory: %w", err)
    }
    return filepath.Join(home, ".config", "revenium"), nil
}

func configPath() (string, error) {
    dir, err := configDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "config.yaml"), nil
}
```

### Pattern 3: Viper Config with Env Var Override

**What:** Set up Viper to read YAML config file and automatically check environment variables with REVENIUM_ prefix.
**When to use:** Config loading.

```go
func Load() (*Config, error) {
    dir, err := configDir()
    if err != nil {
        return nil, err
    }

    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(dir)

    // Environment variable overrides
    viper.SetEnvPrefix("REVENIUM")
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

    // Defaults
    viper.SetDefault("api-url", "https://api.revenium.ai/profitstream")

    // Read config file (ignore "not found" error)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("error reading config: %w", err)
        }
    }

    return &Config{
        APIKey: viper.GetString("api-key"),
        APIURL: viper.GetString("api-url"),
    }, nil
}
```

### Pattern 4: Config Set with Directory Creation

**What:** `config set` must create the `~/.config/revenium/` directory if it does not exist. Viper does NOT create directories automatically.
**When to use:** Config set command.

```go
func Set(key, value string) error {
    dir, err := configDir()
    if err != nil {
        return err
    }

    // Create directory if needed
    if err := os.MkdirAll(dir, 0o700); err != nil {
        return fmt.Errorf("cannot create config directory: %w", err)
    }

    path, _ := configPath()

    // Load existing config first
    viper.SetConfigFile(path)
    _ = viper.ReadInConfig() // ignore not-found

    viper.Set(key, value)
    return viper.WriteConfigAs(path)
}
```

### Pattern 5: Centralized HTTP Client with Error Mapping

**What:** Single `do()` method handles all HTTP requests. Always closes response body. Maps status codes to actionable errors.
**When to use:** All API communication.

```go
// internal/api/client.go
type Client struct {
    BaseURL    string
    APIKey     string
    HTTPClient *http.Client
    Verbose    bool
}

func NewClient(baseURL, apiKey string, verbose bool) *Client {
    return &Client{
        BaseURL: baseURL,
        APIKey:  apiKey,
        HTTPClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        Verbose: verbose,
    }
}

func (c *Client) do(ctx context.Context, method, path string, body, result interface{}) error {
    // Build request
    var bodyReader io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil {
            return fmt.Errorf("failed to marshal request: %w", err)
        }
        bodyReader = bytes.NewReader(b)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
    if err != nil {
        return err
    }

    req.Header.Set("x-api-key", c.APIKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    req.Header.Set("User-Agent", fmt.Sprintf("revenium-cli/%s", build.Version))

    // Verbose logging (to stderr)
    if c.Verbose {
        fmt.Fprintf(os.Stderr, "> %s %s\n", method, c.BaseURL+path)
    }

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return &APIError{Message: "Could not connect to api.revenium.ai. Check your network connection."}
    }
    defer func() {
        io.Copy(io.Discard, resp.Body) // drain for connection reuse
        resp.Body.Close()
    }()

    // Verbose response logging
    if c.Verbose {
        fmt.Fprintf(os.Stderr, "< %d %s\n", resp.StatusCode, resp.Status)
    }

    if resp.StatusCode >= 400 {
        return mapHTTPError(resp)
    }

    if result != nil {
        return json.NewDecoder(resp.Body).Decode(result)
    }
    return nil
}
```

### Pattern 6: Lip Gloss Error Box

**What:** Distinctive styled error box with border for terminal error display.
**When to use:** All user-facing errors.

```go
// internal/errors/errors.go
import (
    lipgloss "charm.land/lipgloss/v2"
)

var errorBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("196")). // Red
    Foreground(lipgloss.Color("196")).
    Padding(0, 1)

func RenderError(msg string) string {
    return errorBoxStyle.Render(msg)
}
```

### Pattern 7: Version Command with ldflags

**What:** Package-level string vars set at build time via `-ldflags -X`.
**When to use:** Version command and User-Agent header.

```go
// internal/build/build.go
package build

var (
    Version = "dev"
    Commit  = "none"
    Date    = "unknown"
)

// cmd/version.go
var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print the version",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("revenium %s (%s)\n", build.Version, build.Commit[:7])
    },
}
```

Build command:
```bash
go build -ldflags="-X 'github.com/revenium/revenium-cli/internal/build.Version=v1.0.0' -X 'github.com/revenium/revenium-cli/internal/build.Commit=$(git rev-parse HEAD)' -X 'github.com/revenium/revenium-cli/internal/build.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o revenium .
```

### Pattern 8: Command Group Registration

**What:** Cobra's AddGroup API for categorized help output.
**When to use:** Root command setup.

```go
// cmd/root.go
func init() {
    rootCmd.AddGroup(
        &cobra.Group{ID: "resources", Title: "Core Resources:"},
        &cobra.Group{ID: "monitoring", Title: "Monitoring:"},
        &cobra.Group{ID: "config", Title: "Configuration:"},
    )

    // config and version commands
    configCmd.GroupID = "config"
    versionCmd.GroupID = "config"
    rootCmd.AddCommand(configCmd)
    rootCmd.AddCommand(versionCmd)

    // Help and completion also in config group
    rootCmd.SetHelpCommandGroupID("config")
    rootCmd.SetCompletionCommandGroupID("config")
}
```

### Anti-Patterns to Avoid
- **Reading flag variables instead of Viper:** After BindPFlag, always read via `viper.GetString()`. The flag variable gets the default, not the resolved value.
- **Using os.UserConfigDir():** Returns `~/Library/Application Support` on macOS. User specified `~/.config/revenium/` (XDG). Use `os.UserHomeDir()` + `/.config/revenium`.
- **HTTP logic in command files:** All HTTP goes through `internal/api/`. Commands never construct requests.
- **Raw http.DefaultClient:** Always configure explicit timeouts.
- **Forgetting resp.Body.Close():** Centralize in `do()` method. Never allow direct HTTP calls outside API client.
- **fmt.Println for styled output:** Use `lipgloss.Println()` for automatic color downsampling based on terminal capabilities.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Command parsing | Custom arg parsing | Cobra v1.10.2 | Flag parsing, help generation, completions, command groups |
| Config file + env vars | Custom YAML parser + os.Getenv logic | Viper v1.21.0 | Handles precedence, type coercion, env prefix, config discovery |
| Terminal border styling | Manual ANSI escape sequences | Lip Gloss v2 | Cross-platform, color downsampling, composable styles |
| Command help groups | Custom help template from scratch | Cobra AddGroup API | Built-in since Cobra 1.6, handles rendering and ordering |

**Key insight:** The Cobra + Viper + Lip Gloss stack handles all Phase 1 needs. The only custom code is the thin HTTP client wrapper (~100 lines) and the error message mapping.

## Common Pitfalls

### Pitfall 1: Viper Config Precedence Trap
**What goes wrong:** Developers read flag values from the bound Go variable instead of from Viper. The flag's default value silently overrides config file and env var values.
**Why it happens:** `BindPFlag` sounds bidirectional but data only flows flag-to-Viper.
**How to avoid:** ALWAYS read via `viper.GetString("api-key")`, never from the flag variable. Write integration tests that verify: flag > env > file > default.
**Warning signs:** Users report "config file is ignored" or "env var doesn't work."

### Pitfall 2: HTTP Response Body Leak
**What goes wrong:** Unclosed response bodies leak TCP connections, eventually causing file descriptor exhaustion.
**Why it happens:** Go requires `resp.Body.Close()` even when body is unused or on error paths.
**How to avoid:** Single `do()` method with `defer resp.Body.Close()` and `io.Copy(io.Discard, resp.Body)` drain before close.
**Warning signs:** Goroutine count increasing. CLOSE_WAIT sockets in `lsof`.

### Pitfall 3: No HTTP Client Timeout
**What goes wrong:** `http.DefaultClient` has zero timeout. Hung API response blocks CLI indefinitely.
**Why it happens:** Go's default `&http.Client{}` has no timeout.
**How to avoid:** Always `&http.Client{Timeout: 30 * time.Second}`.
**Warning signs:** CLI hangs silently with no error.

### Pitfall 4: os.UserConfigDir() Returns Wrong Path on macOS
**What goes wrong:** `os.UserConfigDir()` returns `~/Library/Application Support` on macOS, not `~/.config`.
**Why it happens:** macOS follows Apple conventions, not XDG.
**How to avoid:** Construct path manually: `os.UserHomeDir()` + `/.config/revenium`.
**Warning signs:** Config file created in unexpected location on macOS.

### Pitfall 5: Viper Does Not Create Directories
**What goes wrong:** `viper.WriteConfigAs()` fails if the parent directory doesn't exist. First-time `config set` fails.
**Why it happens:** Viper writes files but does not create directory trees.
**How to avoid:** Call `os.MkdirAll(dir, 0o700)` before `viper.WriteConfigAs()`.
**Warning signs:** "no such file or directory" error on first `config set`.

### Pitfall 6: Cobra Prints Usage on Error
**What goes wrong:** Every error also prints the full usage/help text, cluttering the error message.
**Why it happens:** Cobra defaults to showing usage whenever RunE returns an error.
**How to avoid:** Set `SilenceErrors: true` and `SilenceUsage: true` on the root command. Handle error display in `main.go` after `Execute()`.
**Warning signs:** Error messages buried under walls of help text.

### Pitfall 7: Verbose Mode Leaks API Key
**What goes wrong:** `--verbose` mode logs full HTTP headers including `x-api-key`. Users paste verbose output into issues.
**Why it happens:** Debug logging dumps all headers without filtering.
**How to avoid:** Mask API key in verbose output: show `x-api-key: rev_****abcd` (last 4 chars only).
**Warning signs:** API keys visible in GitHub issue pastes.

## Code Examples

### YAML Config File Structure
```yaml
# ~/.config/revenium/config.yaml
api-key: "your-api-key-here"
api-url: "https://api.revenium.ai/profitstream"  # optional, has default
```

### main.go Entry Point
```go
// main.go
package main

import (
    "fmt"
    "os"

    "github.com/revenium/revenium-cli/cmd"
    "github.com/revenium/revenium-cli/internal/errors"
)

func main() {
    if err := cmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, errors.RenderError(err.Error()))
        os.Exit(1)
    }
}
```

### Error Status Code Mapping
```go
func mapHTTPError(resp *http.Response) error {
    body, _ := io.ReadAll(resp.Body)

    switch resp.StatusCode {
    case 401:
        return &APIError{
            StatusCode: 401,
            Message:    "Invalid API key. Run `revenium config set key <your-key>` to fix.",
            Body:       string(body),
        }
    case 403:
        return &APIError{
            StatusCode: 403,
            Message:    "Access denied. Your API key may not have permission for this operation.",
            Body:       string(body),
        }
    case 404:
        return &APIError{
            StatusCode: 404,
            Message:    "Resource not found.",
            Body:       string(body),
        }
    default:
        if resp.StatusCode >= 500 {
            return &APIError{
                StatusCode: resp.StatusCode,
                Message:    "Revenium API error. Try again later or contact support.",
                Body:       string(body),
            }
        }
        return &APIError{
            StatusCode: resp.StatusCode,
            Message:    fmt.Sprintf("Request failed (HTTP %d).", resp.StatusCode),
            Body:       string(body),
        }
    }
}
```

### Makefile Targets
```makefile
VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X 'github.com/revenium/revenium-cli/internal/build.Version=$(VERSION)' \
           -X 'github.com/revenium/revenium-cli/internal/build.Commit=$(COMMIT)' \
           -X 'github.com/revenium/revenium-cli/internal/build.Date=$(DATE)'

.PHONY: build test lint

build:
	go build -ldflags="$(LDFLAGS)" -o revenium .

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Lip Gloss v1 (`github.com/charmbracelet/lipgloss`) | Lip Gloss v2 (`charm.land/lipgloss/v2`) | Mar 2025 | New vanity import path, built-in tables, deterministic styles |
| Cobra without command groups | Cobra AddGroup API | v1.6 (2022) | Categorized help output without custom templates |
| Manual table rendering | Lip Gloss v2 built-in table | Mar 2025 | No need for olekukonko/tablewriter |

**Deprecated/outdated:**
- `github.com/charmbracelet/lipgloss` (v1): Use `charm.land/lipgloss/v2` instead
- `cobra.Command.SetHelpTemplate` for groups: Use `AddGroup` + `GroupID` instead

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing stdlib + testify v1.10.0+ |
| Config file | None -- Wave 0 creates initial test files |
| Quick run command | `go test ./... -count=1` |
| Full suite command | `go test ./... -v -count=1 -race` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FNDN-01 | Root command exists with help | unit | `go test ./cmd/ -run TestRootCommand -v` | Wave 0 |
| FNDN-02 | Config loads from YAML file at correct path | unit | `go test ./internal/config/ -run TestLoadConfig -v` | Wave 0 |
| FNDN-03 | Config set writes key and api-url to file | unit | `go test ./internal/config/ -run TestSetConfig -v` | Wave 0 |
| FNDN-04 | Env vars override config file values | unit | `go test ./internal/config/ -run TestEnvOverride -v` | Wave 0 |
| FNDN-05 | HTTP client sends x-api-key header, handles timeouts | unit | `go test ./internal/api/ -run TestClient -v` | Wave 0 |
| FNDN-06 | HTTP status codes map to actionable error messages | unit | `go test ./internal/api/ -run TestErrorMapping -v` | Wave 0 |
| FNDN-07 | Non-zero exit on error (cobra SilenceErrors) | integration | `go test ./cmd/ -run TestExitCode -v` | Wave 0 |
| FNDN-12 | Version command prints version string | unit | `go test ./cmd/ -run TestVersionCommand -v` | Wave 0 |
| FNDN-13 | Help text includes examples | unit | `go test ./cmd/ -run TestHelpExamples -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./... -count=1`
- **Per wave merge:** `go test ./... -v -count=1 -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/config/config_test.go` -- covers FNDN-02, FNDN-03, FNDN-04
- [ ] `internal/api/client_test.go` -- covers FNDN-05, FNDN-06 (uses net/http/httptest)
- [ ] `internal/errors/errors_test.go` -- covers error rendering
- [ ] `cmd/root_test.go` -- covers FNDN-01, FNDN-07, FNDN-13
- [ ] `cmd/version_test.go` -- covers FNDN-12
- [ ] Framework install: `go get github.com/stretchr/testify@latest`

## Open Questions

1. **Config key naming: `api-key` vs `api_key` in YAML**
   - What we know: Viper normalizes keys internally. YAML files can use either hyphens or underscores.
   - What's unclear: Which convention the user prefers for the YAML file content
   - Recommendation: Use `api-key` (hyphenated) in YAML to match CLI flag style (`config set key`). Viper's SetEnvKeyReplacer handles the REVENIUM_API_KEY mapping.

2. **Verbose flag scope: global persistent or per-command?**
   - What we know: User wants `--verbose` to show full HTTP context
   - What's unclear: Whether it should be a root persistent flag or per-command
   - Recommendation: Root persistent flag `--verbose` / `-v` so it works on any command

3. **Config show format**
   - What we know: `revenium config show` displays current config
   - What's unclear: Whether to show resolved values (with env overrides) or file-only values
   - Recommendation: Show resolved values with source indication (e.g., `api-key: rev_****abcd (from config file)` or `api-url: https://... (default)`)

## Sources

### Primary (HIGH confidence)
- [Cobra official docs](https://cobra.dev/) - command structure, PersistentPreRunE, AddGroup
- [Cobra user guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md) - command patterns
- [Cobra 12-factor app tutorial](https://cobra.dev/docs/tutorials/12-factor-app/) - Viper integration
- [Viper pkg.go.dev](https://pkg.go.dev/github.com/spf13/viper) - WriteConfigAs, Set, AutomaticEnv
- [Lip Gloss v2 pkg.go.dev](https://pkg.go.dev/charm.land/lipgloss/v2) - border styling, NewStyle
- [Go os.UserConfigDir proposal #29960](https://github.com/golang/go/issues/29960) - macOS returns ~/Library/Application Support

### Secondary (MEDIUM confidence)
- [Cobra + Viper precedence](https://cobra.dev/docs/learning-resources/learning-journey/) - flag > env > file > default
- [Viper issue #671](https://github.com/spf13/viper/issues/671) - flag default overrides env var pitfall
- [DigitalOcean ldflags tutorial](https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications) - build-time version embedding
- [Lip Gloss v2 discussion #506](https://github.com/charmbracelet/lipgloss/discussions/506) - v2 migration guide

### Tertiary (LOW confidence)
- None -- all findings verified with official sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries are well-established with official documentation verified
- Architecture: HIGH - patterns from gh CLI, Cobra official docs, and ARCHITECTURE.md research
- Pitfalls: HIGH - documented in official issue trackers and verified by multiple sources
- Config path: HIGH - verified os.UserConfigDir behavior on macOS via Go issue tracker

**Research date:** 2026-03-11
**Valid until:** 2026-04-11 (stable stack, 30-day validity)
