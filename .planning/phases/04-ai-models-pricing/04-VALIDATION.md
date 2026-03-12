---
phase: 4
slug: ai-models-pricing
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + testify v1.11.1 |
| **Config file** | None needed (Go convention) |
| **Quick run command** | `go test ./cmd/models/... -v -count=1` |
| **Full suite command** | `go test ./... -v -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/models/... -v -count=1`
- **After every plan wave:** Run `go test ./... -v -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | AIMD-01, AIMD-02, AIMD-03, AIMD-04 | unit | `go test ./cmd/models/... -run TestModel -v -count=1` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 2 | AIMD-05, AIMD-06, AIMD-07, AIMD-08 | unit | `go test ./cmd/models/... -run TestPricing -v -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/models/list_test.go` — stubs for AIMD-01 (list models)
- [ ] `cmd/models/get_test.go` — stubs for AIMD-02 (get model)
- [ ] `cmd/models/update_test.go` — stubs for AIMD-03 (PATCH update)
- [ ] `cmd/models/delete_test.go` — stubs for AIMD-04 (delete model)
- [ ] `cmd/models/pricing_list_test.go` — stubs for AIMD-05 (list pricing dimensions)
- [ ] `cmd/models/pricing_create_test.go` — stubs for AIMD-06 (create pricing dimension)
- [ ] `cmd/models/pricing_update_test.go` — stubs for AIMD-07 (update pricing dimension)
- [ ] `cmd/models/pricing_delete_test.go` — stubs for AIMD-08 (delete pricing dimension)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Models table renders with correct alignment | AIMD-01 | Visual verification | Run `revenium models list` with sample data |
| Pricing dimensions table shows under model context | AIMD-05 | Visual verification | Run `revenium models pricing list <model-id>` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
