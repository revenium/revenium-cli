# Requirements: Revenium CLI

**Defined:** 2026-03-11
**Core Value:** Customers can manage every aspect of their Revenium account from the terminal with a tool that's both beautiful and scriptable.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Foundation

- [ ] **FNDN-01**: CLI binary named `revenium` with Cobra-based command structure and root help
- [ ] **FNDN-02**: Config file at `~/.revenium/config.yaml` storing API key and API URL
- [ ] **FNDN-03**: `revenium config set key <value>` and `revenium config set api-url <value>` commands
- [ ] **FNDN-04**: Environment variable override (`REVENIUM_API_KEY`, `REVENIUM_API_URL`) taking precedence over config file
- [ ] **FNDN-05**: HTTP client with x-api-key auth header, proper timeouts, and response body cleanup
- [ ] **FNDN-06**: Helpful error messages mapping HTTP status codes to actionable guidance (401 → "Invalid API key", etc.)
- [ ] **FNDN-07**: Non-zero exit codes on all error paths
- [ ] **FNDN-08**: Styled table output using Lip Gloss v2 as default display format
- [ ] **FNDN-09**: `--json` flag on all output commands for machine-readable output
- [ ] **FNDN-10**: TTY detection — disable colors/styling when output is piped, respect `NO_COLOR` env var
- [ ] **FNDN-11**: Shell completions for bash, zsh, and fish via Cobra built-in
- [ ] **FNDN-12**: `revenium version` command with build-time version/commit/date embedding
- [ ] **FNDN-13**: `--help` with usage examples on every command
- [ ] **FNDN-14**: Distribution via GoReleaser with cross-platform binaries
- [ ] **FNDN-15**: Homebrew tap for macOS/Linux installation
- [ ] **FNDN-16**: `--quiet` / `-q` flag to suppress non-error output
- [ ] **FNDN-17**: `--verbose` / `-v` flag to show HTTP request/response details for debugging

### Sources

- [ ] **SRCS-01**: User can list all sources with styled table output
- [ ] **SRCS-02**: User can get a source by ID with detailed view
- [ ] **SRCS-03**: User can create a new source
- [ ] **SRCS-04**: User can update an existing source
- [ ] **SRCS-05**: User can delete a source with confirmation prompt (`--yes` to skip)

### AI Models

- [ ] **AIMD-01**: User can list all AI models
- [ ] **AIMD-02**: User can get an AI model by ID
- [ ] **AIMD-03**: User can update AI model pricing (PATCH)
- [ ] **AIMD-04**: User can delete an AI model
- [ ] **AIMD-05**: User can list pricing dimensions for a model
- [ ] **AIMD-06**: User can create a pricing dimension for a model
- [ ] **AIMD-07**: User can update a pricing dimension
- [ ] **AIMD-08**: User can delete a pricing dimension

### Subscribers

- [ ] **SUBS-01**: User can list all subscribers
- [ ] **SUBS-02**: User can get a subscriber by ID
- [ ] **SUBS-03**: User can create a new subscriber
- [ ] **SUBS-04**: User can update a subscriber
- [ ] **SUBS-05**: User can delete a subscriber with confirmation prompt

### Subscriptions

- [ ] **SUBR-01**: User can list all subscriptions
- [ ] **SUBR-02**: User can get a subscription by ID
- [ ] **SUBR-03**: User can create a new subscription
- [ ] **SUBR-04**: User can update a subscription
- [ ] **SUBR-05**: User can partially update a subscription (PATCH)
- [ ] **SUBR-06**: User can delete a subscription with confirmation prompt

### Products

- [ ] **PROD-01**: User can list all products
- [ ] **PROD-02**: User can get a product by ID
- [ ] **PROD-03**: User can create a new product
- [ ] **PROD-04**: User can update a product
- [ ] **PROD-05**: User can delete a product with confirmation prompt

### Tools

- [ ] **TOOL-01**: User can list all tools
- [ ] **TOOL-02**: User can get a tool by ID
- [ ] **TOOL-03**: User can create a new tool
- [ ] **TOOL-04**: User can update a tool
- [ ] **TOOL-05**: User can delete a tool with confirmation prompt

### Teams

- [ ] **TEAM-01**: User can list all teams
- [ ] **TEAM-02**: User can get a team by ID
- [ ] **TEAM-03**: User can create a new team
- [ ] **TEAM-04**: User can update a team
- [ ] **TEAM-05**: User can delete a team with confirmation prompt
- [ ] **TEAM-06**: User can view prompt capture settings for a team
- [ ] **TEAM-07**: User can update prompt capture settings for a team

### Users

- [ ] **USER-01**: User can list all users
- [ ] **USER-02**: User can get a user by ID
- [ ] **USER-03**: User can create a new user
- [ ] **USER-04**: User can update a user
- [ ] **USER-05**: User can delete a user with confirmation prompt

### Anomalies & Alerts

- [ ] **ALRT-01**: User can list AI anomalies
- [ ] **ALRT-02**: User can get an anomaly by ID
- [ ] **ALRT-03**: User can create an anomaly detection rule
- [ ] **ALRT-04**: User can update an anomaly rule
- [ ] **ALRT-05**: User can delete an anomaly rule
- [ ] **ALRT-06**: User can list AI alerts
- [ ] **ALRT-07**: User can create AI alert rules
- [ ] **ALRT-08**: User can manage budget alert thresholds

### Credentials & Charts

- [ ] **CRED-01**: User can list provider credentials (masked display)
- [ ] **CRED-02**: User can get a provider credential by ID (masked)
- [ ] **CRED-03**: User can create a provider credential
- [ ] **CRED-04**: User can update a provider credential
- [ ] **CRED-05**: User can delete/deactivate a provider credential
- [ ] **CHRT-01**: User can list chart definitions
- [ ] **CHRT-02**: User can get a chart definition by ID
- [ ] **CHRT-03**: User can create a chart definition
- [ ] **CHRT-04**: User can update a chart definition
- [ ] **CHRT-05**: User can delete a chart definition

### Metrics

- [ ] **METR-01**: User can query AI metrics with `--from` and `--to` time range flags
- [ ] **METR-02**: User can query AI completion metrics
- [ ] **METR-03**: User can query AI audio metrics
- [ ] **METR-04**: User can query AI image metrics
- [ ] **METR-05**: User can query AI video metrics
- [ ] **METR-06**: User can query AI traces (aggregated by traceId)
- [ ] **METR-07**: User can query squad metrics (multi-agent workflows)
- [ ] **METR-08**: User can query API metrics
- [ ] **METR-09**: User can query tool event metrics

## v2 Requirements

### Power Features

- **POWR-01**: `--jq` flag for inline JSON filtering
- **POWR-02**: `--template` flag for Go template output formatting
- **POWR-03**: `--format csv` for financial data export
- **POWR-04**: `revenium api` raw command for arbitrary API endpoints
- **POWR-05**: Update notifications (check GitHub releases, 24h cache)
- **POWR-06**: Command aliases (`src` for `sources`, etc.)
- **POWR-07**: Auto-pagination for large result sets

### Billing

- **BILL-01**: Invoice listing and viewing
- **BILL-02**: Payment received viewing
- **BILL-03**: Period charge operations

## Out of Scope

| Feature | Reason |
|---------|--------|
| Interactive TUI (Bubble Tea) | Adds massive complexity; this is a CLI with beautiful output, not a TUI |
| Multiple environment profiles | Single env at a time keeps config simple |
| OAuth / browser login flow | API uses x-api-key; OAuth adds unnecessary complexity |
| Plugin/extension system | Focused domain doesn't justify the maintenance burden |
| Real-time streaming / live dashboards | WebSocket/SSE complexity not justified for management CLI |
| Docker distribution | Homebrew + binary releases sufficient |
| Offline mode / caching | API management needs live data; caching introduces staleness |
| YAML/TOML output formats | JSON + table covers all real use cases |
| Import/export configurations | Versioning and conflict resolution nightmares |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| (Populated during roadmap creation) | | |

**Coverage:**
- v1 requirements: 89 total
- Mapped to phases: 0
- Unmapped: 89 ⚠️

---
*Requirements defined: 2026-03-11*
*Last updated: 2026-03-11 after initial definition*
