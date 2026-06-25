# Changelog

All notable changes to the Revenium CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/revenium/revenium-cli/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/revenium/revenium-cli/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/revenium/revenium-cli/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/revenium/revenium-cli/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/revenium/revenium-cli/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/revenium/revenium-cli/compare/v1.0.3...v1.1.0
[1.0.3]: https://github.com/revenium/revenium-cli/releases/tag/v1.0.3
