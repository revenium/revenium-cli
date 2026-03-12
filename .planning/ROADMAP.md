# Roadmap: Revenium CLI

## Overview

The Revenium CLI delivers a beautiful, scriptable command-line interface for the Revenium AI Economic Control platform. The roadmap starts with project scaffolding and the output layer, proves the CRUD pattern with a single resource (Sources), then systematically applies that pattern across all platform resources. Metrics and distribution close out the project. Every phase delivers verifiable capability.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Project Scaffold & Config** - Go binary, Cobra root command, config management, HTTP client, error handling (completed 2026-03-12)
- [ ] **Phase 2: Output Layer** - Styled table rendering, JSON output, TTY detection, quiet/verbose flags
- [ ] **Phase 3: First Resource (Sources)** - Prove the full CRUD vertical slice end-to-end
- [ ] **Phase 4: AI Models & Pricing** - Model CRUD plus nested pricing dimension management
- [ ] **Phase 5: Subscribers & Subscriptions** - Consumer management and source-to-subscriber mappings
- [ ] **Phase 6: Products & Tools** - Product catalog and tool registration CRUD
- [ ] **Phase 7: Teams & Users** - Team management with settings, user account CRUD
- [ ] **Phase 8: Anomalies & Alerts** - AI anomaly detection rules and alert/budget threshold management
- [ ] **Phase 9: Credentials & Charts** - Provider credential management (masked) and chart definitions
- [ ] **Phase 10: Metrics** - AI, completion, audio, image, video, trace, squad, API, and tool event metrics
- [ ] **Phase 11: Distribution & Shell Completions** - GoReleaser, Homebrew tap, shell completions

## Phase Details

### Phase 1: Project Scaffold & Config
**Goal**: User can install and configure the CLI with API credentials, and all commands share consistent error handling
**Depends on**: Nothing (first phase)
**Requirements**: FNDN-01, FNDN-02, FNDN-03, FNDN-04, FNDN-05, FNDN-06, FNDN-07, FNDN-12, FNDN-13
**Success Criteria** (what must be TRUE):
  1. Running `revenium` displays help with usage examples; running `revenium version` shows version/commit/date
  2. User can run `revenium config set key <value>` and `revenium config set api-url <value>` and values persist in `~/.config/revenium/config.yaml`
  3. Setting `REVENIUM_API_KEY` overrides the config file value for authentication
  4. API calls include `x-api-key` header, and a 401 response produces "Invalid API key" (not a raw HTTP error)
  5. All error paths exit with non-zero status codes
**Plans:** 2/2 plans complete

Plans:
- [ ] 01-01-PLAN.md — Go module scaffold + internal packages (config, build, errors)
- [ ] 01-02-PLAN.md — API client + Cobra commands (root, config, version) + main.go + Makefile

### Phase 2: Output Layer
**Goal**: All commands render beautiful styled tables by default, with JSON/quiet/verbose alternatives
**Depends on**: Phase 1
**Requirements**: FNDN-08, FNDN-09, FNDN-10, FNDN-16, FNDN-17
**Success Criteria** (what must be TRUE):
  1. List/get commands display data in styled Lip Gloss v2 tables with colors and alignment
  2. Adding `--json` to any output command produces valid, parseable JSON
  3. Piping output to another command (e.g., `revenium sources list | wc -l`) produces clean unstyled text and respects `NO_COLOR`
  4. `--quiet` suppresses all non-error output; `--verbose` shows HTTP request/response details
**Plans:** 2 plans

Plans:
- [ ] 02-01-PLAN.md — Output package: Formatter, styled tables, JSON rendering, TTY detection
- [ ] 02-02-PLAN.md — Wire --json/--quiet flags to root command + JSON error handling in main.go

### Phase 3: First Resource (Sources)
**Goal**: User can fully manage Sources, proving the CRUD pattern that all subsequent resources will follow
**Depends on**: Phase 2
**Requirements**: SRCS-01, SRCS-02, SRCS-03, SRCS-04, SRCS-05
**Success Criteria** (what must be TRUE):
  1. `revenium sources list` displays all sources in a styled table
  2. `revenium sources get <id>` displays a single source with all fields
  3. `revenium sources create` creates a source and displays the result
  4. `revenium sources update <id>` updates a source and displays the result
  5. `revenium sources delete <id>` prompts for confirmation (skippable with `--yes`) and deletes the source
**Plans**: TBD

Plans:
- [ ] 03-01: TBD
- [ ] 03-02: TBD

### Phase 4: AI Models & Pricing
**Goal**: User can manage AI models and their pricing dimensions
**Depends on**: Phase 3
**Requirements**: AIMD-01, AIMD-02, AIMD-03, AIMD-04, AIMD-05, AIMD-06, AIMD-07, AIMD-08
**Success Criteria** (what must be TRUE):
  1. `revenium models list` and `revenium models get <id>` display AI model data in styled tables
  2. `revenium models update <id>` patches model pricing and `revenium models delete <id>` removes a model
  3. `revenium models pricing list <model-id>` displays pricing dimensions for a specific model
  4. `revenium models pricing create/update/delete` manages individual pricing dimensions
**Plans**: TBD

Plans:
- [ ] 04-01: TBD
- [ ] 04-02: TBD

### Phase 5: Subscribers & Subscriptions
**Goal**: User can manage API consumers and their subscription mappings to sources
**Depends on**: Phase 3
**Requirements**: SUBS-01, SUBS-02, SUBS-03, SUBS-04, SUBS-05, SUBR-01, SUBR-02, SUBR-03, SUBR-04, SUBR-05, SUBR-06
**Success Criteria** (what must be TRUE):
  1. `revenium subscribers list/get/create/update/delete` fully manages subscriber records
  2. `revenium subscriptions list/get/create/update/delete` fully manages subscription records
  3. `revenium subscriptions update <id>` supports both full update (PUT) and partial update (PATCH via `--patch` or similar)
  4. Delete commands for both resources prompt for confirmation with `--yes` override
**Plans**: TBD

Plans:
- [ ] 05-01: TBD
- [ ] 05-02: TBD

### Phase 6: Products & Tools
**Goal**: User can manage product catalog entries and tool registrations
**Depends on**: Phase 3
**Requirements**: PROD-01, PROD-02, PROD-03, PROD-04, PROD-05, TOOL-01, TOOL-02, TOOL-03, TOOL-04, TOOL-05
**Success Criteria** (what must be TRUE):
  1. `revenium products list/get/create/update/delete` fully manages product records
  2. `revenium tools list/get/create/update/delete` fully manages tool records
  3. Both resources follow the established CRUD pattern with styled tables, JSON output, and delete confirmation
**Plans**: TBD

Plans:
- [ ] 06-01: TBD
- [ ] 06-02: TBD

### Phase 7: Teams & Users
**Goal**: User can manage team structures (including team-level settings) and user accounts
**Depends on**: Phase 3
**Requirements**: TEAM-01, TEAM-02, TEAM-03, TEAM-04, TEAM-05, TEAM-06, TEAM-07, USER-01, USER-02, USER-03, USER-04, USER-05
**Success Criteria** (what must be TRUE):
  1. `revenium teams list/get/create/update/delete` fully manages team records
  2. `revenium teams prompt-capture get <team-id>` and `revenium teams prompt-capture set <team-id>` view and update prompt capture settings
  3. `revenium users list/get/create/update/delete` fully manages user records
  4. Both resources follow the established CRUD pattern with styled tables, JSON output, and delete confirmation
**Plans**: TBD

Plans:
- [ ] 07-01: TBD
- [ ] 07-02: TBD

### Phase 8: Anomalies & Alerts
**Goal**: User can manage AI anomaly detection rules, alert configurations, and budget thresholds
**Depends on**: Phase 3
**Requirements**: ALRT-01, ALRT-02, ALRT-03, ALRT-04, ALRT-05, ALRT-06, ALRT-07, ALRT-08
**Success Criteria** (what must be TRUE):
  1. `revenium anomalies list/get/create/update/delete` manages anomaly detection rules
  2. `revenium alerts list` displays AI alert configurations; `revenium alerts create` creates new alert rules
  3. `revenium alerts budget` manages budget alert thresholds
  4. All commands follow the established output pattern (styled tables, `--json`, delete confirmation)
**Plans**: TBD

Plans:
- [ ] 08-01: TBD
- [ ] 08-02: TBD

### Phase 9: Credentials & Charts
**Goal**: User can manage provider credentials (with sensitive data masked) and chart definitions
**Depends on**: Phase 3
**Requirements**: CRED-01, CRED-02, CRED-03, CRED-04, CRED-05, CHRT-01, CHRT-02, CHRT-03, CHRT-04, CHRT-05
**Success Criteria** (what must be TRUE):
  1. `revenium credentials list` and `revenium credentials get <id>` display credentials with secret values masked (e.g., `sk-****7f3a`)
  2. `revenium credentials create/update/delete` manages credential records; delete can also deactivate
  3. `revenium charts list/get/create/update/delete` fully manages chart definitions
  4. Both resources follow the established output pattern (styled tables, `--json`, delete confirmation)
**Plans**: TBD

Plans:
- [ ] 09-01: TBD
- [ ] 09-02: TBD

### Phase 10: Metrics
**Goal**: User can query all Revenium metric types with time range filtering and meaningful output
**Depends on**: Phase 2
**Requirements**: METR-01, METR-02, METR-03, METR-04, METR-05, METR-06, METR-07, METR-08, METR-09
**Success Criteria** (what must be TRUE):
  1. `revenium metrics ai --from <date> --to <date>` queries AI metrics with time range filtering
  2. `revenium metrics completions`, `metrics audio`, `metrics image`, `metrics video` each query their respective metric type
  3. `revenium metrics traces` displays AI traces aggregated by traceId; `revenium metrics squads` displays multi-agent workflow metrics
  4. `revenium metrics api` queries API metrics; `revenium metrics tool-events` queries tool event metrics
  5. All metric commands support `--json` output and display results in styled tables
**Plans**: TBD

Plans:
- [ ] 10-01: TBD
- [ ] 10-02: TBD
- [ ] 10-03: TBD

### Phase 11: Distribution & Shell Completions
**Goal**: User can install the CLI via Homebrew or download a binary, and set up shell completions
**Depends on**: Phase 10
**Requirements**: FNDN-11, FNDN-14, FNDN-15
**Success Criteria** (what must be TRUE):
  1. `brew install revenium/tap/revenium` installs the CLI on macOS/Linux
  2. GitHub Releases contain cross-platform binaries built by GoReleaser with embedded version info
  3. `revenium completion bash/zsh/fish` outputs valid shell completion scripts that work when sourced
**Plans**: TBD

Plans:
- [ ] 11-01: TBD
- [ ] 11-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> ... -> 11

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Project Scaffold & Config | 1/2 | Complete    | 2026-03-12 |
| 2. Output Layer | 0/2 | Not started | - |
| 3. First Resource (Sources) | 0/2 | Not started | - |
| 4. AI Models & Pricing | 0/2 | Not started | - |
| 5. Subscribers & Subscriptions | 0/2 | Not started | - |
| 6. Products & Tools | 0/2 | Not started | - |
| 7. Teams & Users | 0/2 | Not started | - |
| 8. Anomalies & Alerts | 0/2 | Not started | - |
| 9. Credentials & Charts | 0/2 | Not started | - |
| 10. Metrics | 0/3 | Not started | - |
| 11. Distribution & Shell Completions | 0/2 | Not started | - |
