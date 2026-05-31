## [1.1.2] - 2026-05-30

### Added

- **Guardrails filter operators:** `--filter` on `budget-rules create|update` now supports `CONTAINS`, `STARTS_WITH`, and `ENDS_WITH` operators in addition to `IS` and `IS_NOT` (e.g. `--filter MODEL:CONTAINS:gpt`, `--filter AGENT:STARTS_WITH:prod-`). The parser already passed operator strings through verbatim; this release documents the expanded set now supported by the API.
