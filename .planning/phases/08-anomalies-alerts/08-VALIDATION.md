---
phase: 8
slug: anomalies-alerts
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + testify v1.11.1 |
| **Config file** | None needed (Go convention) |
| **Quick run command** | `go test ./cmd/anomalies/... ./cmd/alerts/... -v -count=1` |
| **Full suite command** | `go test ./... -v -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/anomalies/... ./cmd/alerts/... -v -count=1`
- **After every plan wave:** Run `go test ./... -v -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | ALRT-01-05 | unit | `go test ./cmd/anomalies/... -v -count=1` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 1 | ALRT-06-08 | unit | `go test ./cmd/alerts/... -v -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/anomalies/*_test.go` — stubs for ALRT-01-05
- [ ] `cmd/alerts/*_test.go` — stubs for ALRT-06-08

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tables render with correct alignment | ALRT-01, ALRT-06 | Visual verification | Run list commands with sample data |
| Currency formatting displays correctly | ALRT-08 | Visual verification | Run budget list with sample monetary data |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
