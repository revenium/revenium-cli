---
phase: 11
slug: distribution-shell-completions
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-12
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + goreleaser check + shell validation |
| **Config file** | .goreleaser.yml |
| **Quick run command** | `goreleaser check && go build -ldflags="$(make -s print-ldflags 2>/dev/null || echo '')" -o /dev/null .` |
| **Full suite command** | `go test ./... -v -count=1 && goreleaser check` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `goreleaser check` (validates .goreleaser.yml)
- **After every plan wave:** Run `go test ./... -v -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | FNDN-14, FNDN-15 | config validation | `goreleaser check` | ❌ W0 | ⬜ pending |
| 11-02-01 | 02 | 2 | FNDN-11 | unit + shell | `revenium completion bash > /dev/null && revenium completion zsh > /dev/null && revenium completion fish > /dev/null` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] GoReleaser installed locally (`goreleaser --version`)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GitHub Release created with binaries | FNDN-14 | Requires git tag + GitHub push | Tag a release, run `goreleaser release --clean` |
| Homebrew formula installs correctly | FNDN-15 | Requires tap repo + published release | Run `brew install revenium/tap/revenium` after release |
| Shell completions work when sourced | FNDN-11 | Requires interactive shell | Source completion script, verify tab completion works |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
