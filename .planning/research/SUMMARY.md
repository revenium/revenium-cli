# Research Summary: Revenium CLI

**Domain:** Go CLI wrapping a REST API with styled terminal output
**Researched:** 2026-03-11
**Overall confidence:** HIGH

## Executive Summary

The Revenium CLI is a well-scoped Go CLI project wrapping a REST API with beautiful terminal output using Charm libraries. The ecosystem for this type of tool is mature and well-established -- Cobra + Viper for command/config, Lip Gloss v2 for styling, net/http for API calls, GoReleaser for distribution. There are no technology risks or unknowns in this stack.

The Charm ecosystem recently shipped v2 of its core libraries (Lip Gloss v2.0.2, Glamour v2.0.0, Huh v2.0.3) in early 2025, with a new vanity import domain (`charm.land/`). These are stable releases. The project should start on v2 from day one to avoid a migration later. Lip Gloss v2 includes built-in table rendering, eliminating the need for a separate table library.

The biggest risk is not technology but scope: the API has ~15+ resource types, each needing CRUD operations, plus specialized metric endpoints. This means 60+ commands. The architecture must be designed for mechanical repetition -- prove the pattern once with a single resource, then replicate. The project structure (one package per resource under `cmd/`) and shared output layer are critical to managing this volume.

The most dangerous pitfalls are operational, not technical: Cobra+Viper config precedence bugs (env vars silently ignored), styled output corrupting piped/scripted usage (no TTY detection), and HTTP response body leaks in the API client. All are well-documented and preventable with the right foundation patterns.

## Key Findings

**Stack:** Go 1.24+ with Cobra v1.10.2, Viper v1.21.0, Lip Gloss v2.0.2, Glamour v2.0.0, net/http stdlib. No external HTTP client needed. GoReleaser for distribution.

**Architecture:** Four-layer architecture -- Command (Cobra), Output (Lip Gloss), API Client (net/http), Config (Viper). Commands are thin wrappers that parse flags, call the API client, and pass results to the output renderer. One package per resource type under `cmd/`.

**Critical pitfall:** Cobra+Viper config precedence trap -- always read config values via `viper.GetString()`, never from the flag variable directly. This is the number one support issue for Cobra+Viper CLIs.

## Implications for Roadmap

Based on research, suggested phase structure:

1. **Foundation** - Build the reusable layers: config, API client, output renderer, root command
   - Addresses: Config management, HTTP client, styled table + JSON output
   - Avoids: Monolithic cmd files, inconsistent error handling, response body leaks

2. **First Resource (Sources)** - Prove the full vertical slice end-to-end
   - Addresses: Sources CRUD, shell completions, version command
   - Avoids: Committing to 15+ resources before validating the pattern

3. **Core Resources** - Apply the proven pattern to remaining CRUD resources
   - Addresses: Models, Subscribers, Subscriptions, Products, Tools, Teams, Users
   - Avoids: Over-engineering -- these are mechanical replication of the proven pattern

4. **Specialized Resources** - Non-CRUD resources with different patterns
   - Addresses: Anomalies, Alerts, Credentials, Charts (may have different API shapes)
   - Avoids: Assuming all resources follow identical CRUD patterns

5. **Metrics & Reporting** - Query endpoints with time ranges, different response shapes
   - Addresses: AI metrics, completion metrics, API metrics, invoices, billing
   - Avoids: Treating metrics like CRUD resources -- they need different output patterns

6. **Distribution & Polish** - GoReleaser, Homebrew tap, update notifications
   - Addresses: Homebrew install, cross-platform binaries, version embedding
   - Avoids: Distribution issues by testing the full brew lifecycle

**Phase ordering rationale:**
- Foundation must come first because every command depends on config, HTTP client, and output
- First Resource before bulk implementation validates the architecture under real conditions
- Core Resources before Specialized because CRUD is the most uniform and predictable
- Metrics last because they have different query patterns, time ranges, and output shapes
- Distribution can partially overlap with other phases (configure GoReleaser early for CI builds)

**Research flags for phases:**
- Phase 1: Standard patterns, unlikely to need research. Charm v2 APIs are well-documented.
- Phase 3-4: May need API-specific research -- check OpenAPI spec for each resource's exact field set
- Phase 5: Needs deeper research into the Revenium metrics API response shapes and query parameters

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All libraries are mature, stable, and well-documented. Versions verified against pkg.go.dev and GitHub releases. |
| Features | HIGH | Based on analysis of gh, stripe CLI, flyctl patterns. Table stakes are well-established for API CLIs. |
| Architecture | HIGH | Four-layer pattern is the Go CLI standard. gh CLI validates this approach at massive scale. |
| Pitfalls | HIGH | All critical pitfalls sourced from official documentation, Cobra/Viper issue trackers, and Cloudflare engineering blog. |

## Gaps to Address

- **Revenium API response shapes:** Need to inspect the OpenAPI spec at `https://api.dev.hcapp.io/profitstream/api-docs/v2` to define Go structs. Consider generating from the spec.
- **Pagination pattern:** Need to verify whether the Revenium API uses offset/limit, cursor, or no pagination.
- **Metrics endpoint query parameters:** Time range format, available aggregations, response structure -- needs API-specific research during Phase 5.
- **Rate limiting:** Does the Revenium API have rate limits? If so, the CLI should handle 429 responses gracefully.
