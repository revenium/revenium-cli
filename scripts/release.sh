#!/usr/bin/env bash
#
# release.sh — Build, release, and update the Homebrew tap for revenium-cli.
#
# Usage:
#   ./scripts/release.sh <version>
#
# Example:
#   ./scripts/release.sh 1.0.0
#
# Prerequisites:
#   - gh CLI installed and authenticated
#   - Write access to revenium/revenium-cli (public) and revenium/homebrew-tap
#   - No uncommitted changes in the working tree

set -euo pipefail

# ── Config ──────────────────────────────────────────────────────────────────
CLI_REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAP_REPO="/Users/johndemic/Development/projects/revenium/homebrew-tap"
GH_RELEASE_REPO="revenium/revenium-cli"
LDFLAGS_PKG="github.com/revenium/revenium-cli/internal/build"
BINARY_NAME="revenium"
DIST_DIR="${CLI_REPO_ROOT}/dist"

# ── Platforms to build ──────────────────────────────────────────────────────
PLATFORMS=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "linux/arm64"
)

# ── Parse args ──────────────────────────────────────────────────────────────
if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 1.0.0"
  exit 1
fi

VERSION="$1"
TAG="v${VERSION}"
COMMIT="$(git -C "${CLI_REPO_ROOT}" rev-parse HEAD)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# ── Preflight checks ───────────────────────────────────────────────────────
echo "==> Preflight checks"

if ! command -v gh &>/dev/null; then
  echo "Error: gh CLI is not installed. Install it with: brew install gh"
  exit 1
fi

if [[ -n "$(git -C "${CLI_REPO_ROOT}" status --porcelain)" ]]; then
  echo "Error: Working tree has uncommitted changes. Commit or stash them first."
  exit 1
fi

if git -C "${CLI_REPO_ROOT}" rev-parse "${TAG}" &>/dev/null; then
  echo "Error: Tag ${TAG} already exists. Bump the version or delete the tag."
  exit 1
fi

echo "  Version:  ${VERSION}"
echo "  Tag:      ${TAG}"
echo "  Commit:   ${COMMIT}"
echo ""
read -rp "Proceed with release? [y/N] " confirm
if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
  echo "Aborted."
  exit 0
fi

# ── Build binaries ──────────────────────────────────────────────────────────
echo "==> Building binaries"
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

LDFLAGS="-X ${LDFLAGS_PKG}.Version=${VERSION} -X ${LDFLAGS_PKG}.Commit=${COMMIT} -X ${LDFLAGS_PKG}.Date=${DATE}"

for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  archive_name="${BINARY_NAME}-${GOOS}-${GOARCH}"
  binary_path="${DIST_DIR}/${archive_name}/${BINARY_NAME}"

  echo "  Building ${GOOS}/${GOARCH}..."
  mkdir -p "${DIST_DIR}/${archive_name}"
  GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
    -ldflags "${LDFLAGS}" \
    -o "${binary_path}" \
    "${CLI_REPO_ROOT}"

  # Create tarball
  tar czf "${DIST_DIR}/${archive_name}.tar.gz" \
    -C "${DIST_DIR}/${archive_name}" \
    "${BINARY_NAME}"

  echo "    -> ${archive_name}.tar.gz"
done

# ── Tag and push ────────────────────────────────────────────────────────────
echo "==> Tagging ${TAG}"
git -C "${CLI_REPO_ROOT}" tag -a "${TAG}" -m "Release ${TAG}"
git -C "${CLI_REPO_ROOT}" push public "${TAG}"

# ── Create GitHub release ───────────────────────────────────────────────────
echo "==> Creating GitHub release ${TAG}"
gh release create "${TAG}" \
  --repo "${GH_RELEASE_REPO}" \
  --title "${TAG}" \
  --notes "Release ${VERSION}" \
  "${DIST_DIR}"/*.tar.gz

# ── Compute SHA256 hashes ───────────────────────────────────────────────────
echo "==> Computing checksums"
declare -A SHAS
for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  archive="${DIST_DIR}/${BINARY_NAME}-${GOOS}-${GOARCH}.tar.gz"
  sha="$(shasum -a 256 "${archive}" | awk '{print $1}')"
  SHAS["${GOOS}_${GOARCH}"]="${sha}"
  echo "  ${GOOS}/${GOARCH}: ${sha}"
done

# ── Update Homebrew tap ─────────────────────────────────────────────────────
echo "==> Updating Homebrew formula"
mkdir -p "${TAP_REPO}/Formula"

RELEASE_URL_BASE="https://github.com/${GH_RELEASE_REPO}/releases/download/${TAG}"

cat > "${TAP_REPO}/Formula/revenium.rb" <<FORMULA
class Revenium < Formula
  desc "CLI for the Revenium AI Economic Control platform"
  homepage "https://github.com/${GH_RELEASE_REPO}"
  version "${VERSION}"
  license "MIT"

  on_macos do
    on_arm do
      url "${RELEASE_URL_BASE}/${BINARY_NAME}-darwin-arm64.tar.gz"
      sha256 "${SHAS[darwin_arm64]}"
    end
    on_intel do
      url "${RELEASE_URL_BASE}/${BINARY_NAME}-darwin-amd64.tar.gz"
      sha256 "${SHAS[darwin_amd64]}"
    end
  end

  on_linux do
    on_intel do
      url "${RELEASE_URL_BASE}/${BINARY_NAME}-linux-amd64.tar.gz"
      sha256 "${SHAS[linux_amd64]}"
    end
    on_arm do
      url "${RELEASE_URL_BASE}/${BINARY_NAME}-linux-arm64.tar.gz"
      sha256 "${SHAS[linux_arm64]}"
    end
  end

  def install
    bin.install "${BINARY_NAME}"
  end

  test do
    assert_match "${VERSION}", shell_output("\#{bin}/${BINARY_NAME} version")
  end
end
FORMULA

# Commit and push the tap update
cd "${TAP_REPO}"
git add Formula/revenium.rb
git commit -m "Update revenium to ${VERSION}"
git push origin main

echo ""
echo "==> Release complete!"
echo "  GitHub:   https://github.com/${GH_RELEASE_REPO}/releases/tag/${TAG}"
echo "  Install:  brew install revenium/tap/revenium"
echo "  Upgrade:  brew upgrade revenium"
