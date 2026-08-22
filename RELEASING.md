# Releasing the Revenium CLI

This document is the steady-state runbook for cutting Revenium CLI releases through the
canonical GoReleaser pipeline (`.github/workflows/release.yml` + `.goreleaser.yml`).

It assumes `revenium/homebrew-tap` and `TAP_GITHUB_TOKEN` are already configured. For the
one-time provisioning that got us here, see `.planning/phases/17-release-pipeline-finish/`.

## Release Path

There is exactly one supported way to ship a release. Development happens in the internal
repository, the public repository receives a mirrored snapshot of the allowlisted paths, and
tags are cut on the public repository. In order:

1. **Land the work on `main` in the internal repo, `revenium/revenium-cli-internal`.** All
   development — features, fixes, tests — goes there first.

2. **Promote `CHANGELOG.md` there** (Prerequisites item 4 below). The changelog is written in
   the internal repo and reaches the public repo through the sync, never the other way round.

3. **Mirror into the public clone:** `scripts/public-sync.sh vX.Y.Z`. This copies only the
   paths listed in `public-allowlist.txt` from the internal repo into the public clone
   (default `../revenium-cli`) on a `vX.Y.Z-public-sync` branch. `public-allowlist.txt` is the
   sole control file for what crosses; anything not listed there never reaches public. The
   script does not push, tag, or release.

4. **Push the sync branch, open a PR, and merge it to `main` on `revenium/revenium-cli`.**

5. **Cut `vX.Y.Z-rc.N`, then `vX.Y.Z`, from the merged PUBLIC `main`** — see Release Flow
   below. That is the only place `.github/workflows/release.yml` fires.

The tail of that sequence — review the branch, push it, merge the PR, then cut the tags — is
exactly what `scripts/public-sync.sh` prints under "Next steps" when it finishes. The script and
this runbook are meant to tell the same story; if they ever disagree, the script is what
actually runs.

### What is not supported

- **Committing release work directly to the public repo.** The internal repo is authoritative
  for every allowlisted path, so the next routine sync deletes public-only files and reverts
  public-only edits underneath those paths. Work committed only to public is work waiting to be
  un-shipped — see "Sync refuses: public-only files would be deleted" under Troubleshooting for
  what this has already cost.

- **Tagging from the internal repo.** No release workflow runs there. An internal tag publishes
  nothing to the GitHub Releases page and updates no Homebrew formula; it only looks like a
  release.

**Note for readers of the public repository:** `scripts/public-sync.sh` and
`public-allowlist.txt` are internal-only tooling and are deliberately not mirrored, so they are
not present in this repository — searching for them here will turn up nothing. Everything else in
this runbook — the tags, the GoReleaser pipeline, the verification checklist, the rollback
sequence — applies here exactly as written.

## Prerequisites

Before cutting any release tag, confirm every item below.

1. **Clean working tree on internal `main`.** In the internal repo,
   `revenium/revenium-cli-internal`, `git status` shows no uncommitted changes and `main`
   is at the commit you intend to release. If you need a pre-release fix, land it on
   internal `main` first — not on the public repo (see Release Path). The public clone
   must be clean too: `scripts/public-sync.sh` refuses to run against a public clone that
   has uncommitted changes.

2. **Test suite green.** Run the full Go test suite in the internal repo:

   ```sh
   go test ./... -count=1
   ```

   All tests must pass. A `-race` run is recommended for non-trivial releases:

   ```sh
   go test ./... -count=1 -race
   ```

   The sync guards described under Troubleshooting carry their own regression harness,
   `scripts/ci/public-sync-guards.test.sh` (20 cases), which
   `.github/workflows/sync-guards.yml` runs on every pull request and every push to `main`.
   Both files are internal-only tooling and are not mirrored — like `public-sync.sh` itself,
   they guard the step before the release rather than the release workflow.

3. **`goreleaser` installed locally.** The maintainer workstation does NOT ship
   `goreleaser` by default. Install one of:

   ```sh
   brew install goreleaser
   # OR
   go install github.com/goreleaser/goreleaser/v2@latest
   ```

   Confirm with `goreleaser --version` (expect v2.x). This is a workstation tool rather
   than a per-repo one, and `.goreleaser.yml` is mirrored, so it behaves the same in either
   clone.

4. **`CHANGELOG.md` updated in the internal repo.** Promote the contents of the
   `## [Unreleased]` section to a new dated version section in the internal
   `CHANGELOG.md`:

   - Add a `## [X.Y.Z] - YYYY-MM-DD` header (ISO-8601 date; bracket-wrapped version).
   - Add an `### Added` / `### Changed` / `### Fixed` / `### Removed` / `### Deprecated` /
     `### Security` subsection for each category that has entries (omit empty ones).
   - Add a compare-URL link reference at the file bottom:
     `[X.Y.Z]: https://github.com/revenium/revenium-cli/compare/vPREV...vX.Y.Z`
   - Update the `[Unreleased]` reference to compare from the new tag.

   The version header MUST use the bracketed form `## [X.Y.Z]` — the
   `scripts/extract-release-notes.sh` awk pattern requires it.

   Do this in the internal repo and let the sync carry it to public. Editing the public
   `CHANGELOG.md` directly puts released sections on one side only, which the sync then
   refuses to mirror over — see "Sync refuses: diverged CHANGELOG history".

5. **Validate locally (recommended).** Two cheap pre-flight checks, run in the internal
   repo before syncing:

   ```sh
   make release-check   # goreleaser check — schema/syntax validation, ~1s
   make release-dry     # goreleaser release --snapshot --clean — full local build, no publish
   ```

   These targets are wired in `Makefile` for convenience. Neither pushes anything; both
   write to the gitignored `dist/` directory. `Makefile` and `.goreleaser.yml` are both
   mirrored, so the same two targets work in the public clone after the merge if you want to
   re-check there before tagging.

## Release Flow

The pipeline always cuts a release-candidate (`-rc.N`) tag first, then the canonical tag.

### Pre-release validation (rc.N)

Always cut a release-candidate tag before the canonical version. The pipeline has historical
failure precedent and rc validation is the gate (D-06).

Run this in the public clone, on the merged `main` of `revenium/revenium-cli`. `origin` there
is the public repo, and pushing the tag to it is what triggers
`.github/workflows/release.yml`.

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

Once an `rc.N` run is green, cut the canonical tag — again in the public clone, on that
same merged `main` of `revenium/revenium-cli`:

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

The first three entries below are different: they cover `scripts/public-sync.sh` refusing
before it mirrors anything, which happens before a tag is ever cut.

All three guards run before the sync writes anything: before `git switch -c`, before the first
`rsync`. A refusal therefore leaves the public clone exactly as it found it — no branch, no file
change, nothing to undo. `--dry-run` prints the identical reports, then carries on to show what
would change without stopping and without writing, so it is the safe way to inspect a refusal.
The override flags suppress only the stop, never the report, and each waives only its own guard.

### Sync refuses: public-only files would be deleted

**Warning sign:** `scripts/public-sync.sh` prints `WARN: the public repo tracks N file(s) with
no counterpart in this internal repo:`, lists the paths, and exits non-zero with `error:
refusing to mirror over public-only files (back-port them, then re-run)`.

**Cause:** the mirror copies each allowlisted directory with `rsync -a --delete`, so a file that
exists only in the public repo is deleted by a routine sync. Nothing about that is a
malfunction — deleting what the source does not have is what a mirror does.

This is not hypothetical. On 2026-08-22, v1.4.0 (the `--skill-*` flags on `meter completion`) and v1.5.0 (`--ticket-id`
across the four AI metering commands — completion, audio, image, video) were developed and
released straight from the public repo, inverting the path above. v1.5.0 added `cmd/ticket.go`
and `cmd/meter/ticket_test.go`, which existed nowhere but public. A routine `public-sync.sh` run
at that moment would have deleted both files and reverted the skill-flag edits to
`cmd/meter/completion.go` — silently un-shipping two already-published releases, with a green
build and no error anywhere. What it cost: every one of those changes had to be back-ported into
the internal repo before any sync could safely run again, and the guards documented here had to
be written so the next occurrence refuses instead of proceeding.

**Fix:** back-port the listed files into the internal repo on `main`, commit them, and re-run
the sync. Pass `--allow-delete` only when you genuinely intend those files to disappear from
the public repo.

### Sync refuses: public commits not represented internally

**Warning sign:** `scripts/public-sync.sh` prints `WARN: public main carries N commit(s) with no
counterpart in this internal repo:`, lists each one as short-sha plus subject, and exits
non-zero with `error: refusing to mirror over public-only commits (back-port them, then
re-run)`.

**Cause:** public `main` carries work that never landed internally, and the mirror would
overwrite the files those commits touched. This is the half of the problem the file-level guard
above cannot see: a public-only commit that only *modifies* a file both repos already have
leaves every path in place, so there is no orphan to report — and the mirror reverts the edit
with nothing printed. The `--skill-*` changes to `cmd/meter/completion.go` in the 2026-08-22
incident are exactly that shape, which is why the orphan guard alone would not have covered it.

Commits are matched by normalized subject (a trailing GitHub ` (#N)` squash suffix is ignored on
both sides) with a `git patch-id --stable` fallback for back-ports that were reworded — never by
SHA. The two repos share no commit graph, because public history is built from squashed sync
commits, so a correctly back-ported commit legitimately has a different SHA and still matches.
The comparison window starts at the newest `vX.Y.Z ... public sync` commit in public history; if
no such marker is found, the entire public history is compared and the script says so.

**Fix:** back-port the listed commits into the internal repo, commit them, and re-run — a
reworded or squashed back-port still matches. `--allow-diverged` proceeds anyway, overwriting
the work those commits did.

### Sync refuses: diverged CHANGELOG history

**Warning sign:** `scripts/public-sync.sh` prints `WARN: public CHANGELOG.md has released
version(s) the internal CHANGELOG.md lacks:` with the version numbers, and exits non-zero with
`error: refusing to mirror over diverged CHANGELOG history (reconcile, then re-run)`.

**Cause:** `CHANGELOG.md` is an allowlisted file, so the sync copies the internal one over the
public one wholesale. Any released version section that exists only in the public changelog is
erased, taking public release history with it.

This guard compares version *sections*, not file trees, which is why it did not catch the
2026-08-22 incident and why the other two guards had to be added. Even when it does fire it
names no file and no commit, so it cannot tell you that `cmd/ticket.go` is about to be deleted;
and public-only work that leaves `CHANGELOG.md` alone never trips it at all.

**Fix:** add the missing version sections to the internal `CHANGELOG.md`, commit, and re-run.
This guard has no override flag — `--allow-delete` deliberately does not waive it, and neither
does `--allow-diverged`.

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
