# Releasing the Revenium CLI

This document is the steady-state runbook for cutting Revenium CLI releases through the
canonical GoReleaser pipeline (`.github/workflows/release.yml` + `.goreleaser.yml`).

It assumes `revenium/homebrew-tap` and `TAP_GITHUB_TOKEN` are already configured. For the
one-time provisioning that got us here, see `.planning/phases/17-release-pipeline-finish/`.

## Prerequisites

Before cutting any release tag, confirm every item below.

1. **Clean working tree on `main`.** `git status` shows no uncommitted changes, and
   `main` is at the commit you intend to release. If you need a pre-release fix, land
   it on `main` first.

2. **Test suite green.** Run the full Go test suite:

   ```sh
   go test ./... -count=1
   ```

   All tests must pass. A `-race` run is recommended for non-trivial releases:

   ```sh
   go test ./... -count=1 -race
   ```

3. **`goreleaser` installed locally.** The maintainer workstation does NOT ship
   `goreleaser` by default. Install one of:

   ```sh
   brew install goreleaser
   # OR
   go install github.com/goreleaser/goreleaser/v2@latest
   ```

   Confirm with `goreleaser --version` (expect v2.x).

4. **`CHANGELOG.md` updated.** Promote the contents of the `## [Unreleased]` section
   to a new dated version section in `CHANGELOG.md`:

   - Add a `## [X.Y.Z] - YYYY-MM-DD` header (ISO-8601 date; bracket-wrapped version).
   - Add an `### Added` / `### Changed` / `### Fixed` / `### Removed` / `### Deprecated` /
     `### Security` subsection for each category that has entries (omit empty ones).
   - Add a compare-URL link reference at the file bottom:
     `[X.Y.Z]: https://github.com/revenium/revenium-cli/compare/vPREV...vX.Y.Z`
   - Update the `[Unreleased]` reference to compare from the new tag.

   The version header MUST use the bracketed form `## [X.Y.Z]` — the
   `scripts/extract-release-notes.sh` awk pattern requires it.

5. **Validate locally (recommended).** Two cheap pre-flight checks:

   ```sh
   make release-check   # goreleaser check — schema/syntax validation, ~1s
   make release-dry     # goreleaser release --snapshot --clean — full local build, no publish
   ```

   These targets are wired in `Makefile` for convenience. Neither pushes anything; both
   write to the gitignored `dist/` directory.

## Release Flow

The pipeline always cuts a release-candidate (`-rc.N`) tag first, then the canonical tag.

### Pre-release validation (rc.N)

Always cut a release-candidate tag before the canonical version. The pipeline has historical
failure precedent and rc validation is the gate (D-06).

```sh
git tag -a vX.Y.Z-rc.1 -m "Release vX.Y.Z-rc.1"
git push origin vX.Y.Z-rc.1
gh run watch
```

Behavior:

- GoReleaser auto-classifies any tag with a `-rc.N` (or `-beta`, `-alpha`) suffix as a
  pre-release on GitHub Releases (`release.prerelease: auto`).
- The Homebrew tap formula update is **skipped** for pre-release tags
  (`brews[0].skip_upload: auto`). This preserves the live `Formula/revenium.rb` so
  `brew install revenium/tap/revenium` keeps installing the last canonical release.
- If the workflow run fails, fix the underlying issue and cut `rc.2`, etc. Per D-07,
  pre-release tags stay on the Releases page as artifacts of the validation history —
  do not delete them.

### Canonical release

Once an `rc.N` run is green, cut the canonical tag:

```sh
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
gh run watch
```

This tag:

- Publishes the GitHub Release with all 6 platform archives + checksums + the release
  body extracted from `CHANGELOG.md` (`--release-notes dist/release-notes.md`).
- Updates `Formula/revenium.rb` on `revenium/homebrew-tap` with the canonical
  6-platform shape including completion install lines.
- Receives the "Latest" badge on the GitHub Releases page (`release.make_latest: true`).

Tags MUST be annotated (`git tag -a`), not lightweight. The annotated form matches the
existing v1.0.x history and the convention from the retired `scripts/release.sh`.

## Verification Checklist

After the canonical workflow run reports green, confirm all three D-09 items:

- [ ] `gh release view vX.Y.Z --json assets` shows the full asset set (6 platform
      archives + checksums) named `revenium-cli_X.Y.Z_*`.
- [ ] `Formula/revenium.rb` on `revenium/homebrew-tap` shows `version "X.Y.Z"`, six
      platforms, valid SHA256s, and the completion install lines from
      `.goreleaser.yml`'s `brews.install` block. Inspect via:

      ```sh
      gh -R revenium/homebrew-tap api 'repos/revenium/homebrew-tap/contents/Formula/revenium.rb' --jq '.content' | base64 -d | head -40
      ```

- [ ] At least one archive (for example, darwin/arm64) downloaded and inspected to
      confirm both the `revenium` binary AND the `completions/revenium.{bash,zsh,fish}`
      files are present in the archive.

The workflow-green signal alone is the maintainer-attention bar (D-08); the three checks
above run regardless and are the spec-mandated verification (RLSE-03).

## Troubleshooting

Common failure modes, with warning signs and one-line fixes. If the workflow's `Run
GoReleaser` step fails, search the log for any of these strings first.

### Token expired or under-scoped

**Warning sign:** 403 / "Permission denied" / "resource not accessible by integration"
in the `brews` publish step, after archive uploads have already succeeded.

**Fix:** Re-issue a fine-grained PAT scoped to `revenium/homebrew-tap` with
`Contents: Read and write`, and store it as the `TAP_GITHUB_TOKEN` repository secret
on `revenium/revenium-cli` (Settings → Secrets and variables → Actions). Fine-grained
PATs expire after 90 days by default; calendar a reminder.

If your org policy blocks fine-grained PATs, fall back to a classic PAT with the `repo`
scope.

### Brews push rejected by tap

**Warning sign:** "branch is protected" / "non-fast-forward" / "pull request required"
in the `brews` step.

**Fix:** On `revenium/homebrew-tap` → Settings → Branches, relax or remove the
protection rule on `main`. Alternatively, configure `brews.pull_request: { enabled:
true, base: { branch: main } }` in `.goreleaser.yml` to route the formula update
through a PR instead of a direct push.

### Canonical run rebuilds the rc and fails with `already_exists`

**Warning sign:** the canonical `vX.Y.Z` run logs `releasing tag=vX.Y.Z-rc.N` and archive
names containing `-rc.N`, then fails with
`422 Validation Failed [{Resource:ReleaseAsset Field:name Code:already_exists}]`.

**Cause:** the rc and canonical tags sit on the same commit, because this runbook has you
cut both with no commits in between. GoReleaser derives its version from `git describe`
when the tag is not supplied, and `describe` cannot rank two tags on one commit reliably —
in CI it resolved to the rc, rebuilt the rc's artifacts, and tried to upload them over the
rc release's existing assets.

**Fix:** the workflow now passes `GORELEASER_CURRENT_TAG: ${{ github.ref_name }}`, which
pins the run to the tag that triggered it. If you see this on an older workflow revision,
add that env var rather than deleting the rc tag (D-07 keeps rc tags). Re-cut the canonical
tag on the commit carrying the fix so one commit holds one release tag.

### GoReleaser version mismatch / schema drift

**Warning sign:** "field X is unsupported" / "field X is unknown" / "field X is removed"
in the GoReleaser step.

**Fix:** The workflow now pins `version: "~> v2.15"` in
`.github/workflows/release.yml`'s `goreleaser-action` step rather than the loose `"~> v2"`
range, so a new v2 minor cannot change release behaviour unannounced. To move to a newer
minor, bump that pin deliberately after running `make release-check` locally against the
new version.

### Pending: `brews` is soft-deprecated in favour of `homebrew_casks`

`goreleaser check` currently emits `brews is being phased out in favor of homebrew_casks`.
This is a **warning, not an error** — `make release-check` exits 0 and releases are
unaffected. It is recorded here because the eventual removal will be a hard break.

Migration is config-ready but **not applied**, because it requires coordinated changes to
`revenium/homebrew-tap` and affects existing users. What it involves:

- The tap artifact moves from `Formula/revenium.rb` to `Casks/revenium.rb`. The old
  formula must be deleted from the tap, otherwise formula users silently pin to the last
  formula version while cask users keep updating.
- `tap_migrations.json` must be added to the tap so existing installs migrate on
  `brew update` instead of stranding users.
- The cask needs a `postflight` `xattr -dr com.apple.quarantine` hook, since the binaries
  are not Apple-notarized.

Two things verified empirically against GoReleaser v2.15.4 and Homebrew's own source, both
contradicting the common assumption that casks are macOS-only:

- **Linux support survives.** The generated cask carries `on_linux` blocks for amd64 and
  arm64. Homebrew's `extend/os/linux/cask/installer.rb` refuses a cask on Linux only when
  it declares `depends_on macos:`, which this one does not.
- **Completions survive**, but only if configured. A naive migration drops them silently;
  the `completions:` mapping below restores the bash/zsh/fish install lines.

The verified replacement block, if and when this is applied:

```yaml
homebrew_casks:
  - name: revenium
    repository:
      owner: revenium
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    skip_upload: auto
    directory: Casks
    homepage: https://github.com/revenium/revenium-cli
    description: Manage your Revenium account from the command line
    license: MIT
    binaries:
      - revenium
    completions:
      bash: completions/revenium.bash
      zsh: completions/revenium.zsh
      fish: completions/revenium.fish
    hooks:
      post:
        install: |
          if OS.mac?
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/revenium"]
          end
```

Note `binaries:` (plural) — the singular `binary:` field is itself deprecated.

### `completions.sh` failing on CI

**Warning sign:** Error immediately after the `before hook 2/N` log line in the
GoReleaser output.

**Fix:** Confirm `go mod tidy` is the first `before.hooks` entry (cache warm-up). If
the failure persists, check the shebang on `scripts/completions.sh` and consider
`GOFLAGS=-mod=mod` in the workflow env.

### Empty release notes / extractor fails

**Warning sign:** `extract-release-notes: empty section for X.Y.Z` in the GoReleaser
output, build halts before publishing.

**Fix:** Confirm the `## [X.Y.Z]` header exists in `CHANGELOG.md` with the exact
bracket-wrapped version and ISO-8601 date format (no extra spaces, no different
date format). The extractor matches the `^## \[X.Y.Z\]` awk pattern verbatim.

### Phase 14 enforcement-events time-field empty in live API (optional pre-release smoke)

**Warning sign:** `revenium guardrails enforcement-events list` against the live API
shows an empty time column (D-18).

**Fix:** Low-confidence path; not a release blocker. Run the command against the live
API as an optional pre-release smoke test. If the time field is empty, file an issue
to investigate the candidate-field list in the column resolver — but do not gate the
release on it.

## Rollback

If a canonical release ships broken and must be retracted, run the following sequence:

1. **Delete the tag locally and remotely.**

   ```sh
   git tag -d vX.Y.Z
   git push origin --delete vX.Y.Z
   ```

2. **Delete the GitHub Release.**

   ```sh
   gh release delete vX.Y.Z
   ```

3. **Revert the tap formula commit.** The commit that GoReleaser pushed to
   `revenium/homebrew-tap` updates `Formula/revenium.rb`; revert it so the formula
   resolves back to the previous version:

   ```sh
   git -C ../homebrew-tap revert <sha>
   git -C ../homebrew-tap push
   ```

   Alternatively, do the revert through the GitHub UI on the tap repo.

**Important caveat:** Homebrew users who already installed the bad version stay on it
until the next release ships. The formula revert only affects new installs
(`brew install revenium/tap/revenium`) and `brew upgrade` runs. Communicate the
breakage through the GitHub Release page (mark the bad release as a draft or add a
prominent note) and roll forward to a `vX.Y.(Z+1)` patch release as soon as the fix
is ready.
