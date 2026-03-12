# Technology Stack

**Project:** Revenium CLI
**Researched:** 2026-03-11

## Recommended Stack

### Core Framework

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Go | 1.24+ | Language | Required by project. Compiles to single binary, excellent cross-platform support, fast startup time ideal for CLI tools. | HIGH |
| Cobra | v1.10.2 | Command framework | Industry standard for Go CLIs. Used by kubectl, docker, gh, hugo. Provides subcommand routing, flag parsing, help generation, shell completions. No v2 exists; v1.10.2 is current stable (Dec 2024). | HIGH |
| Viper | v1.21.0 | Configuration | Natural companion to Cobra. Reads YAML config files, supports env var overrides, handles config file discovery (~/.revenium/config.yaml). v1.21.0 is current stable (Sep 2024). | HIGH |

### Charm Libraries (Terminal Styling)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Lip Gloss | v2.0.2 | Styled output, tables, lists | The core of beautiful terminal output. v2 brings deterministic styles, built-in table/list/tree rendering, color downsampling. Import: `charm.land/lipgloss/v2`. Released Mar 2025. | HIGH |
| Glamour | v2.0.0 | Markdown rendering | Stylesheet-based markdown rendering for help text and rich descriptions. Integrates with Lip Gloss v2. Import: `charm.land/glamour/v2`. | MEDIUM |
| Huh | v2.0.3 | Interactive forms/prompts | For config setup wizard and confirmation prompts. NOT for full TUI -- just targeted prompts (e.g., `revenium config init`). Import: `charm.land/huh/v2`. | MEDIUM |

**Important: Charm v2 uses vanity import paths.** All Charm v2 libraries use `charm.land/` instead of `github.com/charmbracelet/`. Use `charm.land/lipgloss/v2`, NOT `github.com/charmbracelet/lipgloss/v2`. Both resolve to the same code, but `charm.land` is the canonical v2 path.

**Lip Gloss v2 standalone usage note:** For non-Bubble-Tea usage (which is our case), use `lipgloss.Println()` / `lipgloss.Printf()` instead of `fmt.Println()` to get automatic color downsampling based on terminal capabilities.

### HTTP Client

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| net/http (stdlib) | stdlib | HTTP client | Use Go's standard library. The Revenium API is straightforward REST with JSON. No need for resty or other wrappers -- the API surface is simple (x-api-key header, JSON bodies, standard CRUD). A thin internal client wrapper around net/http keeps dependencies minimal and gives full control over request/response handling. | HIGH |

**Why NOT go-resty:** Resty (v2.17.2) adds convenience for complex APIs with OAuth, retries, middleware chains. The Revenium API needs one custom header (`x-api-key`) and JSON serialization -- stdlib handles this cleanly. Adding resty means a dependency that provides marginal benefit for this use case. A ~100-line internal HTTP client wrapper is the better trade.

### JSON Handling

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| encoding/json (stdlib) | stdlib | JSON serialization | Standard library. Stable, well-understood. Go 1.25 will ship encoding/json/v2 as experimental, but for a CLI launching now, stick with proven v1. The `--json` flag output and API response parsing are both well-served by stdlib. | HIGH |

**Why NOT encoding/json/v2:** Still experimental as of Go 1.25. Performance gains are irrelevant for a CLI making single API calls. Stick with what every Go developer knows.

### Configuration

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Viper | v1.21.0 | Config file + env vars | Reads `~/.revenium/config.yaml`, supports `REVENIUM_API_KEY` and `REVENIUM_API_URL` env var overrides. Cobra+Viper integration is well-documented and battle-tested. | HIGH |
| YAML (gopkg.in/yaml.v3) | v3 | Config format | YAML for the config file format. Human-readable, standard for CLI tools. Pulled in by Viper automatically. | HIGH |

### Distribution & Build

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| GoReleaser | latest (OSS) | Build & release | Automates cross-compilation, GitHub releases, Homebrew formula generation, checksums, changelogs. The standard for Go CLI distribution. Free OSS tier is sufficient. | HIGH |
| Homebrew Tap | - | macOS/Linux install | GoReleaser auto-generates and pushes the formula to a `homebrew-tap` repository. Users install via `brew install revenium/tap/revenium`. | HIGH |
| GitHub Actions | - | CI/CD | Standard CI. GoReleaser has a first-party GitHub Action (`goreleaser/goreleaser-action`). | HIGH |

### Testing

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| testing (stdlib) | stdlib | Unit tests | Go's built-in testing. No framework needed. | HIGH |
| net/http/httptest (stdlib) | stdlib | HTTP mocking | Spin up test servers to mock the Revenium API. Clean, no dependencies. | HIGH |
| testify | v1.10.0+ | Assertions | `require` and `assert` packages reduce test boilerplate. Widely adopted. Optional but strongly recommended. | HIGH |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pflag | v1.0.6+ | POSIX flags | Pulled in by Cobra. Provides POSIX-compliant flag parsing. |
| cobra-cli | v2.0.3 | Scaffolding | One-time use during initial project setup to generate command boilerplate. `go install github.com/spf13/cobra-cli@latest` |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| CLI Framework | Cobra | urfave/cli v2 | Cobra is the industry standard. urfave/cli has less ecosystem tooling, fewer examples, smaller community. |
| CLI Framework | Cobra | Kong | Struct-tag based. Clever but less conventional. Cobra's explicit command registration is easier for large CLIs with 15+ resource types. |
| HTTP Client | net/http (stdlib) | go-resty v2 | Unnecessary dependency for a simple REST API with one auth header. |
| HTTP Client | net/http (stdlib) | hashicorp/go-retryablehttp | Retry logic is nice-to-have, not essential for a CLI. Can add later if needed. |
| Config | Viper | koanf | Viper is the natural Cobra companion. koanf is lighter but loses the Cobra integration. |
| Terminal Styling | Lip Gloss v2 | fatih/color | Lip Gloss is far more capable -- tables, borders, layouts. fatih/color is just foreground/background colors. |
| Terminal Styling | Lip Gloss v2 | pterm | pterm bundles everything (spinners, tables, bars). Lip Gloss is more composable and Charm ecosystem is more cohesive. |
| Tables | Lip Gloss v2 table | olekukonko/tablewriter | Lip Gloss v2 has built-in table rendering with full style control. No need for a separate table library. |
| JSON | encoding/json | goccy/go-json | Performance irrelevant for CLI; stdlib is safer and simpler. |
| Testing | testify | gomega/ginkgo | BDD frameworks add complexity. testify + stdlib is the Go standard. |

## Go Module Setup

```bash
# Initialize module
go mod init github.com/revenium/revenium-cli

# Core dependencies
go get github.com/spf13/cobra@v1.10.2
go get github.com/spf13/viper@v1.21.0
go get charm.land/lipgloss/v2@v2.0.2
go get charm.land/glamour/v2@latest
go get charm.land/huh/v2@v2.0.3

# Test dependencies
go get github.com/stretchr/testify@latest
```

## Project Structure

```
revenium-cli/
  cmd/              # Cobra command definitions
    root.go         # Root command, global flags (--json, --config)
    sources.go      # revenium sources [list|get|create|update|delete]
    models.go       # revenium models ...
    ...             # One file per resource type
  internal/
    api/            # HTTP client wrapper, request/response types
      client.go     # API client struct, auth, base URL
      sources.go    # Source-specific API methods
      ...
    config/         # Viper config loading
    output/         # Table rendering, JSON output, styling
      table.go      # Lip Gloss table helpers
      json.go       # --json flag output
      styles.go     # Shared Lip Gloss style definitions
  main.go           # Entry point
  .goreleaser.yaml  # Release configuration
```

## Version Compatibility Matrix

| Dependency | Minimum Go | Notes |
|------------|-----------|-------|
| Cobra v1.10.2 | Go 1.21+ | |
| Viper v1.21.0 | Go 1.21+ | |
| Lip Gloss v2.0.2 | Go 1.23+ | Charm v2 requires newer Go |
| Glamour v2.0.0 | Go 1.24+ | |
| Huh v2.0.3 | Go 1.23+ | |

**Target Go version: 1.24+** to satisfy all dependencies, particularly Glamour v2.

## Sources

- [Cobra GitHub](https://github.com/spf13/cobra) - v1.10.2, Dec 2024
- [Cobra releases](https://github.com/spf13/cobra/releases)
- [Viper GitHub](https://github.com/spf13/viper) - v1.21.0, Sep 2024
- [Lip Gloss v2 on pkg.go.dev](https://pkg.go.dev/charm.land/lipgloss/v2) - v2.0.2, Mar 2025
- [Lip Gloss GitHub releases](https://github.com/charmbracelet/lipgloss/releases)
- [Lip Gloss v2 Discussion](https://github.com/charmbracelet/lipgloss/discussions/506) - migration guide
- [Glamour v2 on pkg.go.dev](https://pkg.go.dev/charm.land/glamour/v2) - v2.0.0
- [Glamour GitHub releases](https://github.com/charmbracelet/glamour/releases)
- [Huh v2 on pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/huh/v2) - v2.0.3
- [Huh GitHub releases](https://github.com/charmbracelet/huh/releases)
- [go-resty GitHub](https://github.com/go-resty/resty) - v2.17.2
- [GoReleaser](https://goreleaser.com/)
- [Go encoding/json/v2 blog post](https://go.dev/blog/jsonv2-exp)
