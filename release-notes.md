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

