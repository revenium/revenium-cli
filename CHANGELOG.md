# Changelog

All notable changes to the Revenium CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.7.0] - 2026-09-26

This release carries two internal milestones at once — v1.6 (API drift sweep and release-path hardening) and v1.7 (jobs economics and dev-spec coverage) — neither of which was ever tagged publicly. It adds 26 commands, the largest of them a complete job-type economics and ROI surface. No public v1.6.0 release exists; this supersedes it.

### Added

- **Drift and coverage report published.** `docs/drift-coverage-report.md` now ships with the repository, at the path this changelog has advertised since v1.3.0. Its four finding tables — deliberate skips, gap triage, audit-visible deprecations, and spec asymmetries — are generated from the coverage audit rather than kept by hand, so the report is refreshed against the OpenAPI specs instead of drifting away from them. The judgement around them is still written by a person and says so.
- **Skill usage read-back.** `revenium skills list` ranks skills by attributed cost and `revenium skills get <skillId>` returns one skill's usage detail. Both accept `--period` (`HOUR`, `EIGHT_HOURS`, `TWENTY_FOUR_HOURS`, `SEVEN_DAYS`, `THIRTY_DAYS`, `NINETY_DAYS`, `SIX_MONTHS`, `TWELVE_MONTHS`; omitted, the server applies its `THIRTY_DAYS` default) and `--json`; `list` also accepts `--page`/`--page-size`. This closes the read side of the write-only skill tracking that shipped in v1.4.0. `--skill-invocation-trigger` stays write-only — neither response schema declares it.
- **Job type economics contracts.** `revenium jobs types economics get <type>` reads a job type's declared economics document and `revenium jobs types economics set <type> --file <doc>` upserts it. `set` replaces the WHOLE document — any metric, dimension or allowed value absent from the supplied JSON is removed — so it first reads the current contract and requires `--yes` only when the replace would remove something already declared. Creating a contract where none exists removes nothing and needs no confirmation, which makes it safe unattended. The read-only `jobType` and `currentBaseline` keys are stripped from input and reported on stderr, so a document from `economics get --json` can be edited and fed straight back. Six closed enums are validated locally with indexed messages before any request is built.
- **Immutable pre-AI baselines.** `revenium jobs types baselines list <type>` returns a job type's baseline versions newest first, and `revenium jobs types baselines append <type>` adds a new immutable version from nine scalars. Baselines are append-only and are never altered by `economics set`.
- **Declared outcome facts.** `revenium jobs types facts append <type> --file <entries>` appends `PERIOD` outcome facts and `revenium jobs outcome-metrics <agenticJobId> --file <entries>` appends late `PER_JOB` outcome metrics to a job that has already finished. Both take a JSON array (or `-` for stdin), send every entry in the order given without merging or deduplicating, and validate each entry by index before any request exists — a rejected entry means nothing was appended. Both endpoints append *declared* facts, so the job type's economics contract must exist first. Appended entries are permanent, cannot be amended, and the API exposes no read-back, which is why `facts` has no `list` verb.
- **Aggregate ROI.** `revenium jobs roi-summary` reports ROI across job types on the analytics host, with four filter flags translated to their API names and omitted when unset; `--metric-type` outside the declared enum is refused before any request is made.
- **Bulk job delete.** `revenium jobs delete` now accepts multiple ids and routes by arity to the single-item or bulk endpoint. A bodyless 2xx bulk delete is reported as UNCONFIRMED rather than as success, and any id the server reports that the operator never named is surfaced.
- **Optimistic concurrency on outcome updates.** `revenium jobs outcome-update` accepts `--expected-entity-version`, sent only when passed, so a caller can refuse to overwrite an outcome that changed underneath it.
- **Team settings.** `revenium teams pr-health get|set`, `revenium teams attribution-identity-policy get|set`, and `revenium teams verified-domains list|add|remove`. `get` renders effective values with a caption naming where each came from; `verified-domains remove` emits a parseable JSON success document.
- **Session attribution.** `revenium sessions attribution <sessionId>` reads a coding-assistant session's attribution. The sibling write is deliberately not exposed — it reads as an automated tooling callback keyed by an opaque session UUID rather than an operator action.
- **Seat and pull-request billing reads.** `revenium billing seats` returns the seat census, `revenium billing vcs-pr-health` reports pull-request health, and `revenium billing vcs-prs-by-org-unit` breaks pull requests down by org unit with a total line and a stable row order. Each renders an explicit empty state rather than a blank table.
- **Cost-control preview.** `revenium guardrails org-unit-group-preview` previews which org units a grouping rule would select. It creates nothing despite being a POST.

### Deprecated

- **`revenium billing users get`** warns on stderr that its endpoint has been withdrawn, naming `revenium billing users` and then `revenium billing vcs-pr-health` as successors in that order — the withdrawn endpoint returned per-user cost detail, so the surviving list is the closer redirect. `revenium billing users` itself is unaffected and does not warn.
- **`revenium billing claude-code-contributions`** warns on stderr before issuing its request, naming `revenium billing vcs-prs` as the successor. The warning is emitted before the call is built, so it appears even when the withdrawn endpoint 404s. stdout remains a single parseable JSON document and the exit code is unchanged.

## [1.5.0] - 2026-08-21

### Added

- **Ticket attribution.** `meter completion|audio|image|video` accept `--ticket-id`, sent only when passed, for attributing cost to an external ticket or issue (e.g. `JIRA-123`, `LINEAR-456`; max 256 characters). The tool-event schema does not define the field, so `meter tool-event` does not expose the flag.

## [1.4.0] - 2026-08-21

### Added

- **Skill tracking.** `meter completion` accepts `--skill-name`, `--skill-invocation-trigger`, `--skill-source`, `--skill-kind`, `--skill-plugin-name`, and `--skill-marketplace-name`, sent only when passed. These fields are specific to the completion endpoint — the audio, image, video, and tool-event schemas do not define them.

## [1.3.0] - 2026-07-27

This release reconciles the CLI against all three Revenium OpenAPI specs — platform, metering, and analytics — and adds squad support across metering and metrics. It also fixes a data-truncation bug that made scripted queries silently return partial results; read the Changed section before upgrading if you parse CLI output in automation.

### Added

- **Squad support.** `meter completion|audio|image|video` accept `--squad-id`, `--squad-name`, and `--squad-role`, sent only when passed. A new `squads` command reads the other side: `squads list`, `squads get <id>`, `squads timeline <id>`, and `squads executions`. The `metrics ai|completions|audio|video|traces` commands render a `Squad` column and accept `--squad-id` to filter on it. That filter runs client-side over the full result set — the API exposes no squad query parameter — and the help text says so.
- **Seven command packages:** `billing`, `invoices`, `refunds`, `period-charges`, `workspaces`, `tenants`, and `metering-elements`. The first six are read-only; `metering-elements` supports full CRUD over metering element definitions.
- **Binary transfer.** `invoices download <id>` retrieves an invoice and writes it to disk. `products logo set|delete`, `sources logo set|delete`, and `teams logo set|delete` upload and remove logos. Both directions share one client implementation rather than repeating it per call site.
- **Analytics host.** The client now reaches `app.revenium.ai` as a distinct base URL with bearer auth, configured by the `analytics-api-url` key or `REVENIUM_ANALYTICS_API_URL`. `metrics dimensions <name>` queries the twelve filter-option lookups over it; `metrics completions get|prompts|reference-data` add completion detail lookups on the platform host.
- **Metering field parity.** Roughly 85 optional fields across the four `meter` commands, covering trace naming, error reporting, cost typing, cache-token accounting, and nested subscriber identity. Each is gated on its flag, so request bodies carry only what you pass.
- **Coverage on existing resources.** `models create|clone|history|rates`, `models pricing coverage|bulk-save`, `users me` with its credentials, invoices, subscriptions, and period-charges sub-lists, `subscriptions billed-amount|quota`, `jobs outcome-update|outcome-history`, `teams tags|children` and `teams coding-assistant-filter get|set`, `tools lookup`, `products history`, `sources history`, `alerts events`, and `anomalies clear|dimensions`.
- **Drift report** at `docs/drift-coverage-report.md`, recording every endpoint deliberately skipped as unsuitable for a CLI, every command backing a deprecated or removed endpoint, and every asymmetry found between the specs and the CLI.

### Changed

- **List commands now return every page, in every output mode.** Previously `--output json` and `--json` fetched a single page, so `revenium metrics ai --from … --to … --output json` returned at most 100 records — the server's per-page cap — while appearing to return the whole result set. Scripts had no way to tell the data was partial. **If you parse CLI output in automation, expect larger result sets and more requests per invocation.** Pass `--page N` to fetch one specific page.
- **`--page-size` selects the batch size instead of disabling pagination.** It previously forced single-page mode, so asking for more records per request was what limited you to one request.
- **`subscriptions create|update` prefer `--subscriber-email`.** `--client-email` still works and takes lower precedence. The hardcoded `clientEmailAddress` backfill moved out of the shared update path into the subscriptions package, so it no longer affects other resources.

### Deprecated

- **`subscriptions --client-email`** in favor of `--subscriber-email`. Passing it prints a warning to stderr and continues.

### Fixed

- **Silent truncation is now visible.** When `--page` limits a request to one page and more pages exist, the CLI reports `Note: showing 1 of N pages.` on stderr. It unwraps the API's pagination envelope before rendering, so that count never reached stdout and callers could not detect a partial result. The note goes to stderr, leaving stdout a clean JSON array, and `--quiet` suppresses it.
- **`metrics audio` and `metrics video` read `durationSeconds`.** They previously read a field the API does not return, rendering every duration as 0.
- **`metrics tool-events` reads `toolName`, `callCount`, and `costUsd`**, replacing three field names the API does not return.
- **`metrics image` reads `actualImageCount`**, which had rendered every count as 0.
- **`models get` and `metrics api` fail immediately with an explanation** instead of calling endpoints the API no longer serves.

## [1.2.1] - 2026-06-25

### Added

- `revenium meter completion` now accepts an optional `--trace-type` flag that sets the `traceType` field on `POST /v2/ai/completions` for distributed-trace classification (distinct from the existing `--trace-id`).

## [1.2.0] - 2026-06-03

### Added

- **Global config-override flags:** five persistent root flags — `--api-key`, `--api-url`, `--team-id`, `--tenant-id`, `--owner-id` — override the corresponding config value for a single invocation, available on every subcommand, without editing the config file or exporting an environment variable. Precedence is `flag > env var > config file > default` (CFGO-01..07).
- Documented the previously-undocumented `REVENIUM_TENANT_ID` and `REVENIUM_OWNER_ID` environment variables in `revenium --help` and the README.

### Security

- One-shot override-flag values are never persisted to `config.yaml` by `revenium config set` — the config write path is isolated from the global flag bindings, so passing `--api-key` alongside a `config set` cannot write the secret to disk. The README notes that `--api-key` on the command line is still exposed to shell history and process listings; prefer the `REVENIUM_API_KEY` env var or the config file for sensitive use.

## [1.1.2] - 2026-05-30

### Added

- **Guardrails filter operators:** `--filter` on `budget-rules create|update` now supports `CONTAINS`, `STARTS_WITH`, and `ENDS_WITH` operators in addition to `IS` and `IS_NOT` (e.g. `--filter MODEL:CONTAINS:gpt`, `--filter AGENT:STARTS_WITH:prod-`). The parser already passed operator strings through verbatim; this release documents the expanded set now supported by the API.

## [1.1.1] - 2026-05-28

### Added

- **Guardrails filters:** `revenium guardrails budget-rules create|update` now accept `--filter dim:op:val` (repeatable) and `--filters-json '<JSON>'` (escape hatch, mutually exclusive with `--filter`) for scoping rules to specific dimensions (e.g. `--filter MODEL:IS:gpt-4`, `--filter AGENT:IS:hermes`). Known dimensions: AGENT, MODEL, PROVIDER, ORGANIZATION, CREDENTIAL, PRODUCT, SUBSCRIBER, TASK_TYPE. PATCH semantics: setting `--filter` on update replaces the entire array.
- **Guardrails notification channels:** `revenium guardrails budget-rules create|update` now accept `--notification-channel-id` (repeatable) to attach notification channels to a rule. PATCH on update replaces the entire array.
- **Guardrails get rendering:** `revenium guardrails budget-rules get` now surfaces `filters` and `notificationChannelIds` in both table and JSON output.

## [1.1.0] - 2026-05-19

### Added

- **Agentic Jobs:** `revenium jobs list|get|create|update|delete` with PATCH update semantics (JOBS-01..05)
- **Agentic Jobs sub-resources:** `revenium jobs outcome|roi|transactions|types|conversion-funnel` with clean 409 messaging on the immutable outcome endpoint (JOBS-06..10)
- **Guardrails:** `revenium guardrails` parent command with `budget-rules` CRUD (PATCH), `enforcement-rules get`, and `enforcement-events list` (GRDR-01..07)
- **Organizations:** `revenium organizations list|get|create|update|delete|tags|children` with PUT update semantics and parent-hierarchy navigation (ORGS-01..07)
- **Cheap-win lookups:** `revenium subscribers lookup --email`, `revenium users lookup --email`, `revenium models lookup --name` (LKUP-01..03)

### Changed

- Release pipeline is now canonical via GoReleaser.
- GitHub Release body is sourced from this CHANGELOG.md file via the `--release-notes` flag (free-tier GoReleaser path; section extracted at release time).

### Fixed

- Release pipeline failures (5 consecutive failed `Run GoReleaser` steps on v1.0.0..v1.0.3 tags) — root cause diagnosed and fixed in this release.

## [1.0.3] - 2026-03-16

(See https://github.com/revenium/revenium-cli/releases/tag/v1.0.3 — v1.0.x history is not back-filled in this CHANGELOG; only v1.1.0+ entries are curated.)

[Unreleased]: https://github.com/revenium/revenium-cli/compare/v1.7.0...HEAD
[1.7.0]: https://github.com/revenium/revenium-cli/compare/v1.5.0...v1.7.0
[1.5.0]: https://github.com/revenium/revenium-cli/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/revenium/revenium-cli/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/revenium/revenium-cli/compare/v1.2.1...v1.3.0
[1.2.1]: https://github.com/revenium/revenium-cli/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/revenium/revenium-cli/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/revenium/revenium-cli/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/revenium/revenium-cli/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/revenium/revenium-cli/compare/v1.0.3...v1.1.0
[1.0.3]: https://github.com/revenium/revenium-cli/releases/tag/v1.0.3
