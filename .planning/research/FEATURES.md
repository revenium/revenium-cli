# Feature Landscape

**Domain:** Go CLI wrapping a REST API (API management platform client)
**Researched:** 2026-03-11
**Confidence:** HIGH -- based on analysis of gh, stripe, flyctl, railway, and AWS CLI patterns

## Table Stakes

Features users expect. Missing = product feels incomplete or amateurish.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| CRUD for all resources | Core purpose of the CLI. Users need list/get/create/update/delete for every resource type. | High (volume) | ~15+ resource types. Repetitive but large surface area. Template the implementation. |
| `--json` flag on all output commands | Every serious CLI supports machine-readable output. Required for scripting, CI/CD, piping to `jq`. | Medium | Implement at the output layer so it's automatic for all commands. gh, stripe, flyctl all do this. |
| Styled table output (default) | Human-readable default is expected. Plain text dumps feel broken. | Medium | Lip Gloss tables per PROJECT.md. Consistent column widths, truncation for long values. |
| Config file for auth (`~/.revenium/config.yaml`) | Users expect persistent auth without re-entering keys. gh uses `~/.config/gh/`, stripe uses `~/.config/stripe/`. | Low | Standard pattern. YAML is fine. |
| Environment variable override (`REVENIUM_API_KEY`) | Required for CI/CD pipelines and containers. Every production CLI supports this. | Low | Env var takes precedence over config file. Also support `REVENIUM_API_URL`. |
| `revenium config set` command | Users need a way to configure without manually editing YAML. | Low | `config set key`, `config set api-url`. |
| Shell completions (bash, zsh, fish) | Power users expect tab completion. Cobra provides this for free. | Low | Cobra has built-in `completion` command generation. Ship it from day one. |
| Helpful error messages | Raw HTTP errors or stack traces are unacceptable. Users need actionable messages. | Medium | Map HTTP status codes to human messages: 401 = "Invalid API key. Run `revenium config set key`", 404 = "Resource not found", 429 = rate limit, etc. |
| `--help` on every command | Cobra provides this automatically. But usage strings must be well-written with examples. | Low | Include examples in `Example:` field on every Cobra command. |
| Version command (`revenium version`) | Users and support need to know what version is running. | Low | Embed via `ldflags` at build time with goreleaser. |
| Homebrew + binary distribution | Users expect `brew install revenium` on macOS. Linux/Windows users expect downloadable binaries. | Medium | goreleaser handles this. Homebrew tap in a separate repo. |
| Non-zero exit codes on failure | Scripts must be able to detect failures. Exit 0 = success, exit 1 = error. | Low | Critical for CI/CD usage. Many CLIs get this wrong. |
| Quiet/silent mode (`--quiet` or `-q`) | Scripts often don't want any output, just the exit code. | Low | Suppress all non-error output. |
| Field selection on output | Users need to pick which fields to display in tables or JSON. | Medium | `--fields` flag like gh does: `revenium sources list --json --fields id,name,status`. |

## Differentiators

Features that set the product apart. Not expected, but valued highly when present.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| `--jq` flag for inline JSON filtering | Lets users filter/transform JSON output without piping to external `jq`. gh CLI pioneered this and developers love it. | Medium | Use `gojq` library. Example: `revenium sources list --json --jq '.[].name'`. |
| `--format` / `--template` flag (Go templates) | Power users can create custom output formats. gh CLI supports this with Go templates. | Medium | Example: `revenium sources list --template '{{.Name}}\t{{.Status}}'`. |
| Update notifications | On CLI start, check if a newer version is available and print a non-intrusive notice. gh and flyctl do this. | Medium | Check GitHub releases API. Cache the check (once per 24h). Never block on it. |
| `revenium api` raw command | Let power users hit arbitrary API endpoints: `revenium api GET /sources`. Escape hatch when CLI doesn't wrap an endpoint yet. | Medium | gh has `gh api`. Extremely valuable during API evolution. Reduces pressure to wrap every endpoint immediately. |
| Colored status indicators | Visual distinction for resource states (active = green, inactive = red, pending = yellow). | Low | Lip Gloss makes this easy. Respect `NO_COLOR` env var standard. |
| Metrics with time range flags | `revenium metrics ai --from 2026-01-01 --to 2026-03-01` with sensible defaults (last 30 days). | Medium | Time parsing with natural language support would be nice but not required. ISO 8601 is fine. |
| Confirmation prompts for destructive actions | `revenium sources delete <id>` should prompt "Are you sure?" with `--yes` / `-y` flag to skip in scripts. | Low | Detect TTY. If not interactive, require `--yes` flag or fail. |
| Output to file (`--output` / `-o`) | Direct output to a file instead of stdout. Useful for reports and exports. | Low | Simple but appreciated, especially for metrics/invoice exports. |
| Pagination handling | Automatically paginate through large result sets, or provide `--limit` and `--page` flags. | Medium | API likely returns paginated results. CLI should auto-paginate by default with `--limit` to cap. |
| Verbose/debug mode (`--verbose` / `-v`) | Show HTTP request/response details for debugging API issues. | Low | Print method, URL, status code, timing. Invaluable for troubleshooting. `--verbose` or `REVENIUM_DEBUG=1`. |
| Aliases and short names | `revenium src` instead of `revenium sources`. Common abbreviations for frequent commands. | Low | Cobra supports aliases natively. |
| Wait/polling for async operations | If any API operations are async, `--wait` flag to poll until complete. | Medium | Only if the API has async operations. Check API docs. |
| CSV/TSV output format | `--format csv` or `--format tsv` for spreadsheet-friendly output. Useful for finance/billing data especially. | Low | Revenium is a financial platform -- users will want to export to spreadsheets. |

## Anti-Features

Features to explicitly NOT build. These add complexity without proportional value for this project.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Interactive TUI (Bubble Tea) | PROJECT.md explicitly scopes this out. TUI adds massive complexity, testing burden, and accessibility issues. This is a CLI, not a TUI. | Beautiful styled output with Lip Gloss. Let the output speak for itself. |
| Multiple environment/profile support | PROJECT.md scopes this out. Adds config complexity. Most users work with one environment. | Single config. Users can switch by re-running `config set`. Power users can use env vars per-invocation. |
| OAuth / browser login flow | API uses x-api-key. OAuth adds dependency on browser, callback server, token refresh logic. | API key via config file or env var. Simple and reliable. |
| Built-in scripting/macro language | Some CLIs try to add scripting. This is what bash/shell is for. | Good `--json` + `--jq` support makes the CLI composable with existing tools. |
| Plugin/extension system | gh has this but gh is a platform tool used by millions. Revenium CLI has a focused domain. Plugin systems add maintenance burden. | `revenium api` raw command serves as the escape hatch. |
| Real-time streaming / live dashboards | PROJECT.md scopes this out. WebSocket/SSE complexity is not justified for a management CLI. | Polling with `--watch` flag (refresh every N seconds) if needed, but defer this. |
| Docker distribution | PROJECT.md scopes this out. Binary distribution via Homebrew and goreleaser is sufficient. | Homebrew tap + GitHub releases. |
| Offline mode / caching | API management tool needs live data. Caching introduces staleness bugs and invalidation complexity. | Always fetch fresh. Cache only version-check results (24h TTL). |
| YAML/TOML output formats | JSON covers machine-readable needs. YAML output adds a dependency and is rarely actually used in practice. | JSON + table + CSV covers all real use cases. |
| Import/export of full configurations | Tempting but introduces versioning, migration, and conflict resolution nightmares. | Individual resource CRUD is sufficient. Bulk operations can be scripted with `--json` output and shell loops. |

## Feature Dependencies

```
Config system (auth) --> All API commands (everything depends on authentication)
HTTP client + error handling --> All API commands
Output formatter (table/JSON) --> All display commands
CRUD for Sources --> Metrics commands (metrics reference sources)
CRUD for Subscribers --> Subscriptions (subscriptions reference subscribers)
CRUD for Sources --> Subscriptions (subscriptions reference sources)
Shell completions --> Better UX for all commands (ship early)
--json flag --> --jq flag (jq operates on JSON output)
--json flag --> --template flag (templates operate on structured data)
--json flag --> --fields flag (field selection from structured data)
goreleaser setup --> Homebrew tap, update notifications
```

## MVP Recommendation

### Must ship in v0.1 (without these, CLI is not usable):
1. **Config system** -- `config set key`, `config set api-url`, env var overrides
2. **HTTP client with error handling** -- proper error messages, non-zero exit codes
3. **Output layer** -- table (default) + `--json` flag
4. **CRUD for 3-4 core resources** -- Sources, Subscribers, Subscriptions, AI Models (prove the pattern)
5. **Shell completions** -- free from Cobra, ship immediately
6. **Version command** -- embedded at build time
7. **Homebrew + goreleaser** -- distribution from day one

### Ship in v0.2 (expand coverage):
8. **CRUD for remaining resources** -- Products, Tools, Teams, Users, Anomalies, Alerts, Credentials, Charts
9. **Metrics querying** -- AI metrics, completion metrics, API metrics with time range flags
10. **`--jq` flag** -- inline JSON filtering
11. **Confirmation prompts** -- for delete operations
12. **Verbose/debug mode** -- `--verbose` flag

### Ship in v0.3 (polish and power features):
13. **`revenium api` raw command** -- escape hatch for any endpoint
14. **`--template` flag** -- Go template output formatting
15. **CSV output** -- `--format csv` for financial data export
16. **Update notifications** -- check for new versions
17. **Invoice/billing operations** -- financial reporting
18. **Pagination** -- auto-paginate large result sets

**Defer indefinitely:** Import/export, plugins, TUI, Docker, OAuth, streaming

## Sources

- [GitHub CLI Manual - Formatting](https://cli.github.com/manual/gh_help_formatting)
- [Stripe CLI Documentation](https://docs.stripe.com/stripe-cli)
- [flyctl - Fly.io CLI Docs](https://fly.io/docs/flyctl/)
- [Railway CLI Docs](https://docs.railway.com/cli)
- [Cobra - Shell Completions](https://cobra.dev/docs/how-to-guides/shell-completion/)
- [go-selfupdate library](https://github.com/creativeprojects/go-selfupdate)
- [gh CLI GitHub Repository](https://github.com/cli/cli)
- [GitHub CLI Table Formatting](https://heaths.dev/tips/2021/08/24/gh-table-formatting.html)
- [Cobra CLI Framework](https://cobra.dev/)
- [Stripe CLI GitHub Repository](https://github.com/stripe/stripe-cli)
