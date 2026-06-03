#!/bin/sh
# Source: free-tier substitute for GoReleaser Pro `release.header.from_file`
set -eu
# Strip leading 'v' from the tag passed by GoReleaser (or use $1)
VERSION="${1:-${GORELEASER_CURRENT_TAG:-}}"
VERSION="${VERSION#v}"
# Strip any pre-release suffix (-rc.N, -beta, -alpha, ...) so a pre-release tag
# (e.g. 1.2.0-rc.1) reuses its base version's CHANGELOG section (## [1.2.0]).
VERSION="${VERSION%%-*}"
[ -n "$VERSION" ] || { echo "extract-release-notes: VERSION empty" >&2; exit 1; }
# Write to repo root, NOT dist/ — GoReleaser asserts dist/ is empty after
# before-hooks run; populating dist/ here aborts the release (#dist-not-empty).
awk -v ver="$VERSION" '
  $0 ~ "^## \\[" ver "\\]" { in_section = 1; print; next }
  in_section && /^## \[/ { exit }
  in_section { print }
' CHANGELOG.md > release-notes.md
[ -s release-notes.md ] || { echo "extract-release-notes: empty section for $VERSION" >&2; exit 1; }
