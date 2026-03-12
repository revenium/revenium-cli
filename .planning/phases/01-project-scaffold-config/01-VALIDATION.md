---
phase: 1
slug: project-scaffold-config
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-11
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing stdlib + testify v1.10.0+ |
| **Config file** | None — Wave 0 creates initial test files |
| **Quick run command** | `go test ./... -count=1` |
| **Full suite command** | `go test ./... -v -count=1 -race` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -count=1`
- **After every plan wave:** Run `go test ./... -v -count=1 -race`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | FNDN-01 | unit | `go test ./cmd/ -run TestRootCommand -v` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | FNDN-02 | unit | `go test ./internal/config/ -run TestLoadConfig -v` | ❌ W0 | ⬜ pending |
| 01-01-03 | 01 | 1 | FNDN-03 | unit | `go test ./internal/config/ -run TestSetConfig -v` | ❌ W0 | ⬜ pending |
| 01-01-04 | 01 | 1 | FNDN-04 | unit | `go test ./internal/config/ -run TestEnvOverride -v` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 1 | FNDN-05 | unit | `go test ./internal/api/ -run TestClient -v` | ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 1 | FNDN-06 | unit | `go test ./internal/api/ -run TestErrorMapping -v` | ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 1 | FNDN-07 | integration | `go test ./cmd/ -run TestExitCode -v` | ❌ W0 | ⬜ pending |
| 01-03-02 | 03 | 1 | FNDN-12 | unit | `go test ./cmd/ -run TestVersionCommand -v` | ❌ W0 | ⬜ pending |
| 01-03-03 | 03 | 1 | FNDN-13 | unit | `go test ./cmd/ -run TestHelpExamples -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/config/config_test.go` — stubs for FNDN-02, FNDN-03, FNDN-04
- [ ] `internal/api/client_test.go` — stubs for FNDN-05, FNDN-06 (uses net/http/httptest)
- [ ] `internal/errors/errors_test.go` — stubs for error rendering
- [ ] `cmd/root_test.go` — stubs for FNDN-01, FNDN-07, FNDN-13
- [ ] `cmd/version_test.go` — stubs for FNDN-12
- [ ] `go get github.com/stretchr/testify@latest` — test assertion library

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Lip Gloss styled error box renders correctly in terminal | FNDN-06 | Visual styling verification | Run `revenium sources list` with invalid API key, verify styled error box with border |
| Help text groups commands by category | FNDN-13 | Visual layout verification | Run `revenium --help`, verify Core Resources/Monitoring/Configuration groups |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
