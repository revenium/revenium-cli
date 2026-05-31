# Changelog

All notable changes to the Revenium CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/revenium/revenium-cli/compare/v1.1.1...HEAD
[1.1.1]: https://github.com/revenium/revenium-cli/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/revenium/revenium-cli/compare/v1.0.3...v1.1.0
[1.0.3]: https://github.com/revenium/revenium-cli/releases/tag/v1.0.3
