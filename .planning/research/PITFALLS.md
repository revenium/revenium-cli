# Domain Pitfalls

**Domain:** Go CLI wrapping a REST API with styled terminal output
**Project:** Revenium CLI
**Researched:** 2026-03-11

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

### Pitfall 1: HTTP Response Body Leaks

**What goes wrong:** Every `http.Do()` call allocates a response body. If not closed -- even when you ignore the body or encounter an error -- TCP connections stay in CLOSE_WAIT, goroutines leak in `net/http.(*persistConn).readLoop`, and the process eventually hits `EMFILE` (too many open files). In a CLI making dozens of API calls per session (listing resources, then getting details), this surfaces as mysterious failures after the tool has been running in scripts.

**Why it happens:** Go's HTTP client requires `resp.Body.Close()` even when the body is unused. Developers often skip it in error paths or when they only care about the status code.

**Consequences:** File descriptor exhaustion, memory leaks, connection pool starvation. Especially dangerous in scripting scenarios where the CLI is called in loops.

**Prevention:**
- Establish a single `doRequest()` helper that always defers `resp.Body.Close()` and drains the body with `io.Copy(io.Discard, resp.Body)` before closing (to enable connection reuse).
- Use `bodyclose` linter (part of `golangci-lint`) to catch unclosed bodies at CI time.
- Never allow raw `http.Client.Do()` calls outside the API client package.

**Detection:** Goroutine count increasing over time. `lsof` showing many CLOSE_WAIT sockets. Flaky test failures in CI.

**Phase:** Foundation / API client layer (Phase 1). Must be right from the first HTTP call.

---

### Pitfall 2: Using Go's Default HTTP Client

**What goes wrong:** `http.DefaultClient` has no timeout. A hung or slow Revenium API response blocks the CLI goroutine indefinitely. Users experience a CLI that "freezes" with no feedback.

**Why it happens:** Go's `net/http` package ships with a zero-timeout default. It is easy to write `http.Get(url)` or use `&http.Client{}` without setting timeouts.

**Consequences:** Hung CLI processes. In CI/CD pipelines, jobs hang until the runner kills them. Users lose trust in the tool.

**Prevention:**
- Create a configured `http.Client` with explicit `Timeout` (e.g., 30s) in the API client constructor.
- Use `context.WithTimeout` per-request for finer control (e.g., list operations get 60s, simple GETs get 15s).
- Always `defer cancel()` immediately after creating a timeout context.
- Display a spinner or progress indicator during requests so users know the CLI is working.

**Detection:** CLI hangs in CI. No timeout errors in logs, just silence.

**Phase:** Foundation / API client layer (Phase 1).

---

### Pitfall 3: Cobra + Viper Config Precedence Trap

**What goes wrong:** The expected precedence is `flag > env var > config file > default`, but developers read flag values from the bound variable (e.g., `apiKey` string bound via `StringVarP`) instead of from Viper. The flag's default value silently overrides the config file and environment variable because `cobra.Command.Flag()` returns the default, not the resolved value.

**Why it happens:** `BindPFlag` implies bidirectional binding, but data flows from flag to Viper only. Reading the original Go variable gets the flag default, not the Viper-resolved value.

**Consequences:** Users set `REVENIUM_API_KEY` env var or put it in `~/.revenium/config.yaml`, but the CLI ignores it and reports "no API key configured." This is the number one support issue for Cobra+Viper CLIs.

**Prevention:**
- After binding, always read values via `viper.GetString("api-key")`, never from the flag variable.
- Write an integration test that sets config file, env var, and flag, then asserts precedence order.
- Use `viper.AllSettings()` in a `--debug` mode to dump resolved config for troubleshooting.
- Document the precedence order in `revenium config --help`.

**Detection:** Users report "config file is ignored" or "env var doesn't work." Debug with `viper.AllSettings()`.

**Phase:** Foundation / config layer (Phase 1). Must be correct before any commands use config values.

---

### Pitfall 4: Styled Output Breaking Pipes and Scripts

**What goes wrong:** Lipgloss ANSI escape codes are written to stdout even when piped to `grep`, `jq`, `less`, or redirected to a file. The `--json` flag exists but styled output leaks into it, or ANSI codes corrupt JSON output. Users get garbled output in automation.

**Why it happens:** No TTY detection. The CLI always renders styled output regardless of whether stdout is a terminal.

**Consequences:** CLI is unusable in scripts, CI/CD pipelines, and shell compositions. The `--json` flag becomes the only usable mode, defeating the purpose of beautiful defaults.

**Prevention:**
- Detect TTY with `os.Stdout.Fd()` + `isatty` or use Lipgloss v2's built-in terminal awareness.
- Respect `NO_COLOR` environment variable (see https://no-color.org).
- When `--json` is active, write JSON to stdout and suppress all styled output.
- Provide `--no-color` flag as explicit override.
- Separate concerns: render logic should receive an "output mode" enum (`styled`, `plain`, `json`) and branch accordingly.
- Never mix human-readable progress/status messages with data output on stdout. Status goes to stderr.

**Detection:** Run `revenium sources list | cat` -- if output contains escape sequences, TTY detection is broken. Run `revenium sources list --json | jq .` -- if jq errors, JSON output is contaminated.

**Phase:** Foundation / output layer (Phase 1). Architecture decision that affects every command.

---

### Pitfall 5: Lipgloss v1 vs v2 Import Path Confusion

**What goes wrong:** Starting with Lipgloss v1 (`github.com/charmbracelet/lipgloss`) then needing v2 features (tables, compositing) forces a migration. The v2 import path changed to `charm.land/lipgloss/v2` with breaking API changes to colors, adaptive colors, and the compat package.

**Why it happens:** Lipgloss v2 shipped stable in February 2025 with a vanity domain import path. Tutorials and Stack Overflow answers still reference v1 patterns. Copilot/AI assistants trained on older data generate v1 code.

**Consequences:** Wasted time migrating. Possible subtle rendering bugs if v1 and v2 are mixed in the dependency tree.

**Prevention:**
- Start with Lipgloss v2 (`charm.land/lipgloss/v2`) from day one. v2.0.2 is stable as of March 2025.
- Use `charm.land/lipgloss/v2/table` for table rendering.
- Pin to `v2.0.x` in `go.mod`.
- Be cautious with AI-generated code -- verify imports match v2 paths.

**Detection:** Build errors mentioning missing methods. `go mod graph` showing both v1 and v2 in the dependency tree.

**Phase:** Foundation / project setup (Phase 1).

## Moderate Pitfalls

### Pitfall 6: Monolithic Command Files

**What goes wrong:** All 15+ resource types with CRUD operations (60+ commands) end up in a single `cmd/root.go` or a flat `cmd/` directory. Adding a new resource means editing the same files. Merge conflicts multiply.

**Prevention:**
- Organize by resource: `cmd/sources/`, `cmd/models/`, `cmd/subscribers/`, etc.
- Each resource package registers its own subcommands via an `init()` or explicit `Register(parent *cobra.Command)` function.
- Keep command definitions thin -- they parse flags, call the API client, and pass results to the output renderer. Business logic stays out of `cmd/`.

**Phase:** Foundation / project structure (Phase 1).

---

### Pitfall 7: Duplicating API Response Structs

**What goes wrong:** The Revenium API has ~15+ resource types with read/write variants (`SourceResource_Read` vs `SourceResource_Write`). Developers manually define Go structs for each, which drift from the API as it evolves. Fields get missed, types are wrong, optional fields crash with nil pointer dereferences.

**Prevention:**
- Generate Go structs from the OpenAPI spec at `https://api.dev.hcapp.io/profitstream/api-docs/v2` using `oapi-codegen` or `openapi-generator`.
- If generating is too heavy, at minimum define structs in a dedicated `api/types/` package and write tests that validate against a sample API response.
- Use pointer types for optional fields (`*string`, `*int`) to distinguish missing from zero-value.
- Use `json:",omitempty"` on write structs to avoid sending zero values as updates.

**Phase:** API client layer (Phase 1-2). Invest early to avoid struct drift.

---

### Pitfall 8: No Centralized Error Handling

**What goes wrong:** Each command handles API errors differently. Some print raw HTTP status codes, some swallow errors, some panic. Users see inconsistent error messages like `unexpected status 403` vs `Error: unauthorized` vs a Go stack trace.

**Prevention:**
- Define an `APIError` struct that captures status code, API error message, and request context.
- The API client layer translates all non-2xx responses into `APIError` before returning.
- The output layer formats `APIError` consistently: human-readable message for terminal, structured error for `--json`.
- Use `cobra.Command.SilenceErrors = true` and `SilenceUsage = true` to prevent Cobra from printing usage on every error. Handle errors in `RunE` return values.

**Phase:** Foundation / API client layer (Phase 1).

---

### Pitfall 9: Table Rendering Breaks on Wide Data

**What goes wrong:** Lipgloss tables render beautifully with short data but break when field values are long (URLs, UUIDs, descriptions). Tables overflow the terminal width, wrapping awkwardly or becoming unreadable. Column truncation is not handled.

**Prevention:**
- Detect terminal width via `lipgloss.Width()` or `term.GetSize()`.
- Define max column widths and truncate with ellipsis for non-essential columns.
- For detail views (`get` commands), use key-value vertical layout instead of single-row tables.
- Test with realistic data lengths, not just "foo" and "bar" test values.

**Phase:** Output layer (Phase 2, when building styled output for resources).

---

### Pitfall 10: JSON Output is an Afterthought

**What goes wrong:** Commands are built with styled output first, then `--json` is bolted on. The JSON structure doesn't match the styled output, fields are named differently, some data shown in tables is missing from JSON, or JSON wraps everything in an unnecessary envelope.

**Prevention:**
- Define the data structure first (a Go struct), render it two ways (styled table, JSON).
- Every command's `RunE` returns a result struct. The output layer checks the `--json` flag and either renders styled or marshals to JSON.
- JSON output should be the canonical data representation. Styled output is a view of that same data.
- Ensure JSON output for list commands returns an array at the top level (not `{"items": [...]}`) unless pagination metadata is included.
- Test both output modes for every command.

**Phase:** Foundation / output architecture (Phase 1). Design the pattern before building individual commands.

---

### Pitfall 11: Goreleaser + Homebrew Formula Issues

**What goes wrong:** Homebrew now prefers casks over formulae for CLI tool distribution. Existing users who installed via an older formula get errors on `brew upgrade`. The goreleaser config generates a formula, but the Homebrew tap repository naming or structure is wrong.

**Prevention:**
- Name the tap repository `homebrew-tap` (not `homebrew-revenium`).
- Test the full install flow: `brew tap`, `brew install`, `brew upgrade`, `brew uninstall`.
- Include proper `test` block in the formula that runs `revenium --version`.
- Set `ldflags` in goreleaser to embed version info: `-X main.version={{.Version}}`.
- Consider providing both formula and cask if targeting macOS app-like distribution.

**Phase:** Distribution (final phase). But configure goreleaser early so CI produces binaries from the start.

## Minor Pitfalls

### Pitfall 12: Config File Path Hardcoded to Home Directory

**What goes wrong:** `~/.revenium/config.yaml` works on personal machines but fails in containers, CI runners with non-standard home directories, or multi-user systems.

**Prevention:**
- Use `os.UserHomeDir()` (not `os.Getenv("HOME")` which is empty on some systems).
- Support `REVENIUM_CONFIG` env var to override the config file path.
- Allow `--config` flag as highest-priority override.

**Phase:** Foundation / config layer (Phase 1).

---

### Pitfall 13: Missing User-Agent Header

**What goes wrong:** API requests don't identify the CLI. When Revenium adds rate limiting or analytics, CLI traffic is indistinguishable from other clients. Debugging server-side issues for CLI users becomes impossible.

**Prevention:**
- Set `User-Agent: revenium-cli/<version> (Go/<go-version>)` on all requests.
- Include version at build time via `ldflags`.

**Phase:** API client layer (Phase 1). Trivial to add, painful to retrofit.

---

### Pitfall 14: Inconsistent Command Naming

**What goes wrong:** Some resources use `revenium source list`, others use `revenium ai-models get`, others use `revenium subscribers delete`. Pluralization, hyphenation, and verb order are inconsistent.

**Prevention:**
- Choose a convention and document it: `revenium <resource-plural> <verb>` (e.g., `revenium sources list`, `revenium models get`, `revenium subscribers create`).
- Use Cobra aliases for common alternatives (`ls` for `list`, `rm` for `delete`).
- Validate naming consistency in code review checklist.

**Phase:** Foundation / command structure (Phase 1).

---

### Pitfall 15: Not Handling Pagination

**What goes wrong:** `list` commands return only the first page of results. Users with hundreds of sources or subscribers see a truncated list with no indication that more data exists.

**Prevention:**
- Check if the Revenium API uses pagination (likely offset/limit or cursor-based).
- Default to fetching all pages for CLI output (with a reasonable limit).
- Support `--limit` and `--offset` flags for explicit control.
- Show "Showing X of Y total" in styled output.

**Phase:** API client layer (Phase 2, when building list commands).

---

### Pitfall 16: Credential Exposure in Debug Output

**What goes wrong:** A `--debug` or `--verbose` flag logs full HTTP requests including the `x-api-key` header. Users paste debug output into GitHub issues, exposing their API key.

**Prevention:**
- Mask sensitive headers in debug output: `x-api-key: rev_****abcd`.
- Never log request bodies for credential management endpoints.
- Document which information is safe to share in bug reports.

**Phase:** API client layer (Phase 1). Build masking into the HTTP debug logger from the start.

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Project setup | Lipgloss v1/v2 confusion | Start with `charm.land/lipgloss/v2` from day one |
| Config system | Viper precedence trap | Always read via `viper.GetString()`, never flag variables |
| API client | Response body leaks, no timeouts | Single `doRequest()` helper with defer close and configured client |
| API client | Struct drift from API | Generate or validate structs against OpenAPI spec |
| Output layer | Styled output in pipes | TTY detection, `NO_COLOR`, output mode enum |
| Output layer | JSON as afterthought | Data struct first, render second |
| Table rendering | Wide data overflow | Terminal width detection, column truncation |
| CRUD commands | Monolithic cmd files | Per-resource packages with thin command handlers |
| Error handling | Inconsistent errors | Centralized `APIError` type, Cobra `SilenceErrors` |
| Distribution | Homebrew formula issues | Test full brew lifecycle, use `homebrew-tap` naming |
| Security | Credential leaks in debug | Mask `x-api-key` in all logging |

## Sources

- [Go HTTP Client Timeout Guide](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/) - HIGH confidence
- [Don't Use Go's Default HTTP Client](https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779) - HIGH confidence
- [Response Body Must Be Closed](https://manishrjain.com/must-close-golang-http-response) - HIGH confidence
- [HTTP Resource Leak Mysteries in Go](https://coder.com/blog/go-leak-mysteries) - HIGH confidence
- [Goroutine Leak from Missing resp.Body.Close](https://dev.to/snhacker9/debugging-a-goroutine-leak-caused-by-missing-respbodyclose-in-go-4n6g) - HIGH confidence
- [Sting of the Viper: Cobra+Viper Integration](https://carolynvanslyck.com/blog/2020/08/sting-of-the-viper/) - HIGH confidence
- [Viper Issue #671: Flag Default Overrides Env Var](https://github.com/spf13/viper/issues/671) - HIGH confidence
- [Lipgloss v2 Release Notes](https://github.com/charmbracelet/lipgloss/releases) - HIGH confidence
- [Lipgloss v2 What's New Discussion](https://github.com/charmbracelet/lipgloss/discussions/506) - HIGH confidence
- [Charm v2 Blog Post](https://charm.land/blog/v2/) - HIGH confidence
- [NO_COLOR Standard](https://no-color.org) - HIGH confidence
- [GoReleaser Homebrew Taps](https://goreleaser.com/customization/homebrew/) - HIGH confidence
- [Homebrew Formula vs Cask Discussion](https://github.com/orgs/goreleaser/discussions/5563) - MEDIUM confidence
- [Cobra User Guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md) - HIGH confidence
- [Go CLI Best Practices with Cobra](https://dasroot.net/posts/2026/02/write-high-performance-go-clis-cobra/) - MEDIUM confidence
