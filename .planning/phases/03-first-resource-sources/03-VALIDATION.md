---
phase: 3
slug: first-resource-sources
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + testify v1.11.1 |
| **Config file** | None needed (Go convention) |
| **Quick run command** | `go test ./cmd/sources/... ./internal/resource/... -v -count=1` |
| **Full suite command** | `go test ./... -v -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/sources/... ./internal/resource/... -v -count=1`
- **After every plan wave:** Run `go test ./... -v -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | SRCS-05 | unit | `go test ./internal/resource/... -run TestConfirm -v -count=1` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | SRCS-01 | unit | `go test ./cmd/sources/... -run TestList -v -count=1` | ❌ W0 | ⬜ pending |
| 03-01-03 | 01 | 1 | SRCS-02 | unit | `go test ./cmd/sources/... -run TestGet -v -count=1` | ❌ W0 | ⬜ pending |
| 03-01-04 | 01 | 1 | SRCS-03 | unit | `go test ./cmd/sources/... -run TestCreate -v -count=1` | ❌ W0 | ⬜ pending |
| 03-01-05 | 01 | 1 | SRCS-04 | unit | `go test ./cmd/sources/... -run TestUpdate -v -count=1` | ❌ W0 | ⬜ pending |
| 03-01-06 | 01 | 1 | SRCS-05 | unit | `go test ./cmd/sources/... -run TestDelete -v -count=1` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 2 | SRCS-01 | unit | `go test ./cmd/... -run TestSourcesRegistered -v -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/resource/resource_test.go` — stubs for ConfirmDelete helper (SRCS-05)
- [ ] `cmd/sources/list_test.go` — stubs for list with data + empty (SRCS-01)
- [ ] `cmd/sources/get_test.go` — stubs for get by ID (SRCS-02)
- [ ] `cmd/sources/create_test.go` — stubs for create (SRCS-03)
- [ ] `cmd/sources/update_test.go` — stubs for partial update (SRCS-04)
- [ ] `cmd/sources/delete_test.go` — stubs for delete with/without confirmation (SRCS-05)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Table renders with correct column alignment in terminal | SRCS-01 | Visual alignment verification | Run `revenium sources list` with sample data, verify columns align |
| Delete confirmation prompt displays correctly | SRCS-05 | Interactive terminal verification | Run `revenium sources delete <id>`, verify "Delete source abc? [y/N]" prompt |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
