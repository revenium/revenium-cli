# Revenium CLI

## What This Is

A beautiful, full-featured command-line interface for the Revenium AI Economic Control platform. Built in Go with Charm libraries (Lip Gloss, Glamour, Huh) for styled output, it gives Revenium customers complete CRUD access to all platform resources — sources, AI models, subscriptions, subscribers, metrics, alerts, billing, and more — with gorgeous table rendering, colored output, and a `--json` flag for scripting.

## Core Value

Customers can manage every aspect of their Revenium account from the terminal with a tool that's both beautiful and scriptable.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] CLI binary named `revenium` with Cobra-based command structure
- [ ] Authentication via config file (~/.revenium/config.yaml) with env var override (REVENIUM_API_KEY)
- [ ] Full CRUD for Sources (API products being tracked)
- [ ] Full CRUD for AI Models and model pricing dimensions
- [ ] Full CRUD for Subscribers (API consumers)
- [ ] Full CRUD for Subscriptions (subscriber-to-source mappings)
- [ ] Full CRUD for Products
- [ ] Full CRUD for Tools
- [ ] Full CRUD for Teams and team settings
- [ ] Full CRUD for Users
- [ ] AI anomaly management (list, get, create, update, delete)
- [ ] AI alerts and budget alert management
- [ ] Provider credential management (CRUD, masked display)
- [ ] Chart definition management
- [ ] Metrics querying — AI metrics, completion metrics, audio/image/video metrics, API metrics, tool event metrics
- [ ] AI traces and squad (multi-agent workflow) metrics
- [ ] Invoice and billing operations
- [ ] Payment received viewing
- [ ] Styled table output using Lip Gloss for all list/get commands
- [ ] `--json` flag on all commands for machine-readable output
- [ ] Config command for setting API URL and key
- [ ] Distribution via Homebrew tap and GitHub releases (goreleaser)

### Out of Scope

- Interactive TUI / Bubble Tea screens — this is a standard CLI with beautiful output, not a TUI app
- Multiple environment profiles — single API URL + key at a time
- OAuth / login flow — API key only
- Docker distribution — Homebrew + binary releases only
- Real-time streaming / live dashboards — standard request/response commands

## Context

- **Revenium** is an AI Economic Control System providing financial visibility and governance for AI spending
- **API**: REST API at `https://api.revenium.ai/profitstream` using `x-api-key` authentication
- **API Docs**: OpenAPI/Swagger at `https://api.dev.hcapp.io/profitstream/api-docs/v2`
- **Charm libraries**: Lip Gloss (styling), Glamour (markdown rendering), Huh (forms), Glow (markdown reader) — from https://charm.sh
- The API follows a consistent resource pattern: list, get, create, update, delete with standard REQ-ID conventions
- Resources have read/write variants (e.g., `SourceResource_Read` vs `SourceResource_Write`)
- The API has ~15+ resource types and specialized metric endpoints

## Constraints

- **Language**: Go — required
- **CLI Framework**: Cobra for command structure — standard Go CLI pattern
- **Styling**: Charm libraries (Lip Gloss minimum) — required for visual quality
- **Auth**: x-api-key header — API limitation
- **API Base**: `https://api.revenium.ai/profitstream` — production endpoint

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + Cobra for CLI framework | Industry standard, excellent for distribution | — Pending |
| Charm Lip Gloss for styling (not Bubble Tea TUI) | Beautiful output without interactive complexity | — Pending |
| Config file + env var for auth | Flexible — file for convenience, env for CI/scripts | — Pending |
| `--json` flag for machine output | Enables scripting and piping without sacrificing default beauty | — Pending |
| Single environment config | Simplicity — users manage one environment at a time | — Pending |
| Homebrew + goreleaser distribution | Easy install for macOS/Linux users, cross-platform binaries | — Pending |

---
*Last updated: 2026-03-11 after initialization*
