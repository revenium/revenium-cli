# Phase 1: Project Scaffold & Config - Context

**Gathered:** 2026-03-11
**Status:** Ready for planning

<domain>
## Phase Boundary

Go binary with Cobra root command, config management (~/.config/revenium/config.yaml), HTTP client with x-api-key auth, error handling, and version command. No resource commands yet — those start in Phase 3.

</domain>

<decisions>
## Implementation Decisions

### Command Structure
- Noun-verb pattern: `revenium sources list`, `revenium models get abc-123`
- Plural resource names: `sources`, `models`, `subscriptions` (not singular)
- Subcommand nesting for child resources: `revenium models pricing list <model-id>`
- Resource IDs as positional arguments: `revenium sources get abc-123` (not `--id`)
- Create/update via flags: `revenium sources create --name "My API" --type rest`
- Top-level utility commands: `config` and `version` only
- Root help groups commands by category (Core Resources, Monitoring, Configuration)

### Config Experience
- Config file at `~/.config/revenium/config.yaml` (XDG standard)
- Subcommands: `revenium config set key <val>`, `revenium config set api-url <val>`, `revenium config show`
- Default API URL baked in: `https://api.revenium.ai/profitstream` — most users never change it
- No config → clear error: "No API key configured. Run `revenium config set key <your-key>` to fix."
- No interactive setup wizard — error + guidance approach

### Error Messaging
- Helpful + concise tone: "Error: Invalid API key. Run `revenium config set key <your-key>` to fix."
- Full Lip Gloss styled error box with border — distinctive visual treatment
- Network failures: "Could not connect to api.revenium.ai. Check your network connection."
- `--verbose` shows full HTTP context on errors: method, URL, status code, response body

### Help Text
- Cobra default help template with examples section
- 2-3 examples per command covering common use cases
- Root help groups commands by category
- No branding/tagline — `revenium version` shows clean `revenium v1.0.0 (abc1234)` only

### Claude's Discretion
- Go module structure and package organization
- Cobra initialization patterns
- HTTP client timeout values and retry behavior
- Exact Lip Gloss error box styling
- Config file YAML structure details

</decisions>

<specifics>
## Specific Ideas

- Command pattern modeled after gh CLI and stripe CLI conventions
- Error boxes should be visually distinctive using Lip Gloss borders — not just colored text
- Config at XDG standard path like gh CLI does

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield project

### Established Patterns
- None — this phase establishes the patterns

### Integration Points
- This phase creates the foundation that all subsequent phases build on
- HTTP client and error handling become shared infrastructure
- Config system used by every command

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-project-scaffold-config*
*Context gathered: 2026-03-11*
