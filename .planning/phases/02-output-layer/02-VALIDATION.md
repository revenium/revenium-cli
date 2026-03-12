---
phase: 2
slug: output-layer
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + testify v1.11.1 |
| **Config file** | None needed (Go convention) |
| **Quick run command** | `go test ./internal/output/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/output/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | FNDN-08 | unit | `go test ./internal/output/... -run TestRenderTable -count=1` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | FNDN-08 | unit | `go test ./internal/output/... -run TestSingleRowTable -count=1` | ❌ W0 | ⬜ pending |
| 02-01-03 | 01 | 1 | FNDN-08 | unit | `go test ./internal/output/... -run TestTruncate -count=1` | ❌ W0 | ⬜ pending |
| 02-01-04 | 01 | 1 | FNDN-09 | unit | `go test ./internal/output/... -run TestRenderJSON -count=1` | ❌ W0 | ⬜ pending |
| 02-01-05 | 01 | 1 | FNDN-09 | unit | `go test ./internal/output/... -run TestRenderJSONError -count=1` | ❌ W0 | ⬜ pending |
| 02-01-06 | 01 | 1 | FNDN-10 | unit | `go test ./internal/output/... -run TestNoColor -count=1` | ❌ W0 | ⬜ pending |
| 02-01-07 | 01 | 1 | FNDN-10 | unit | `go test ./internal/output/... -run TestNonTTY -count=1` | ❌ W0 | ⬜ pending |
| 02-01-08 | 01 | 1 | FNDN-16 | unit | `go test ./internal/output/... -run TestQuiet -count=1` | ❌ W0 | ⬜ pending |
| 02-01-09 | 01 | 1 | FNDN-16 | unit | `go test ./internal/output/... -run TestQuietWithJSON -count=1` | ❌ W0 | ⬜ pending |
| 02-01-10 | 01 | 1 | FNDN-17 | unit | `go test ./cmd/... -run TestVerboseFlag -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/output/table_test.go` — stubs for FNDN-08 (table rendering, truncation, single-row)
- [ ] `internal/output/json_test.go` — stubs for FNDN-09 (JSON output, JSON errors)
- [ ] `internal/output/output_test.go` — stubs for FNDN-10, FNDN-16 (TTY detection, quiet mode, NO_COLOR)
- [ ] `cmd/root_test.go` — new tests for --json and --quiet flag registration

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Table renders with rounded borders and colors in terminal | FNDN-08 | Visual styling verification | Run a list command with sample data, verify rounded borders and colored headers |
| NO_COLOR env strips all ANSI from output | FNDN-10 | Environment-dependent visual | Run `NO_COLOR=1 revenium sources list`, verify no color codes |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
