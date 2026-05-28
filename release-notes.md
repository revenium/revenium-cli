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

