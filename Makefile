VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

MODULE  = github.com/revenium/revenium-cli/internal/build
LDFLAGS = -X $(MODULE).Version=$(VERSION) -X $(MODULE).Commit=$(COMMIT) -X $(MODULE).Date=$(DATE)

.PHONY: build test test-race lint clean release-dry release-check coverage-audit coverage-audit-offline \
        coverage-audit-dev field-audit field-audit-offline field-backtest report-check report-write

build:
	go build -ldflags="$(LDFLAGS)" -o revenium .

test:
	go test ./... -v -count=1

test-race:
	go test ./... -v -count=1 -race

lint:
	golangci-lint run ./...

# Internal maintenance tooling — see scripts/ci/coverage-audit.sh. Revalidates
# the cached OpenAPI documents (unchanged ones are not re-downloaded), then
# reports every spec operation the CLI does not call.
# NOTE: this Makefile is mirrored to the public repo, but the scripts these
# targets call are INTERNAL_ONLY. Each guards itself so a public user gets an
# explanation instead of a bare 'No such file or directory'.
coverage-audit:
	@test -x scripts/ci/coverage-audit.sh || { echo "make coverage-audit: internal maintenance tooling — scripts/ci/coverage-audit.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	scripts/ci/coverage-audit.sh

# The same report with no network access at all, against the cache already on
# disk. This is the form to run repeatedly while working through the gaps.
coverage-audit-offline:
	@test -x scripts/ci/coverage-audit.sh || { echo "make coverage-audit-offline: internal maintenance tooling — scripts/ci/coverage-audit.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	scripts/ci/coverage-audit.sh --offline

# The same report against the DEV platform document instead of the published
# production one. This is the measurement the v1.7 build plan is derived from,
# so it needs to be a repeatable command rather than a hand-downloaded file.
#
# Metering and analytics still come from the published production catalog — the
# dev host returns 403 for the meter document and does not publish analytics —
# so only the platform surface changes, and it changes by a constant in
# scripts/ci/fetch-openapi-specs.sh, never by an argument.
#
# Uses its own cache root: .cache/openapi-dev by default, or SPEC_CACHE_DIR if
# one is set, since that variable overrides the target's default rather than
# choosing between roots. The two platform documents are cached under the same
# filename, so sharing a root means one silently overwrites the other and the
# next ordinary run 304s the survivor into permanence — pointing SPEC_CACHE_DIR
# at the production root under --dev is therefore refused, exiting 2 before
# anything is fetched or created. The guarantee is enforced in
# scripts/ci/fetch-openapi-specs.sh, not merely advised here.
#
# WHAT MAKES THAT TRUE, stated because these sentences are what the next
# reviewer will check the invariant against (26-REVIEW.md WR-06). THREE controls,
# all in fetch-openapi-specs.sh. The first two are PATH-shaped and sit above
# --print-cache-dir; the third keys on the cache's CONTENTS and so has to run
# later, once meta.json has been read:
#   1. the target/root agreement comparison is CANONICAL — taken on the physical
#      directory, not on the argument text — so a trailing '//', a '/./'
#      segment, a relative path typed from a subdirectory and a checkout reached
#      through a symlink are all refused. Until 26-10 it was a string comparison
#      and all four walked past it;
#   2. a separate basename check refuses any root whose name collides with the
#      other target's ('openapi' under --dev, 'openapi-dev' without it), which
#      catches a COPY of a root reachable under its own name;
#   3. a recorded-target refusal that does not look at the path at all. A
#      successful run records its target in the cache's meta.json and a later
#      disagreeing run is refused whatever the root is called, which is what
#      makes a `cp -a` to /tmp/devcache refusable. An UNMARKED root — every cache
#      predating the marker, including the two this repository carries — is not
#      refused on that ground, but its target is INFERRED from the platform URL
#      its cache metadata records and an unambiguous disagreement is refused
#      there too (26-REVIEW.md WR-04).
#
# THE RESIDUAL, narrowed by 26-14 and again by the WR-04 inference rather than
# closed, and this block previously misstated it (26-REVIEW.md WR-02: it still
# said a root named neither 'openapi' nor 'openapi-dev' "is detected by neither
# control", which control 3 had already made false). What remains is an OFF
# SWITCH: all of control 3 reads meta.json, so a root with no meta.json — deleted,
# never written, or restored without it — has no recorded target, nothing to
# infer one from and no recorded digest either, and the separation then rests
# entirely on controls 1 and 2. `rm .cache/openapi/meta.json` is the whole of it.
# The run says so on stderr. A root recording BOTH platform URLs is undecidable
# in the same way. So .cache/openapi is protected against every SPELLING of its
# own path, against its own NAME, and by what its metadata records — which is not
# the same as protected unconditionally.
#
# Read-only with respect to committed artifacts — it writes no report and no
# inventory.
coverage-audit-dev:
	@test -x scripts/ci/coverage-audit.sh || { echo "make coverage-audit-dev: internal maintenance tooling — scripts/ci/coverage-audit.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	scripts/ci/coverage-audit.sh --dev

# The field dimension of the same audit, through the same wrapper: one TSV row
# per request-body property or query parameter that appears on only one side of
# the join. Same fetch behaviour as coverage-audit above, so the field report
# and the endpoint report are never built from different cache states.
# Recipe echo suppressed with @, unlike the coverage-audit pair above. Those
# print a human table; these print TSV that 21-06 and the backtest read with cut
# and comm, and make's own echo of the recipe line would arrive as the first row
# of it.
field-audit:
	@test -x scripts/ci/coverage-audit.sh || { echo "make field-audit: internal maintenance tooling — scripts/ci/coverage-audit.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	@scripts/ci/coverage-audit.sh --fields

# The field report with no network access at all. This is the form to run
# repeatedly while working through the rows.
field-audit-offline:
	@test -x scripts/ci/coverage-audit.sh || { echo "make field-audit-offline: internal maintenance tooling — scripts/ci/coverage-audit.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	@scripts/ci/coverage-audit.sh --offline --fields

# The live half of SC2: run the field audit over a git archive of the pinned
# baseline commit and over the working tree, and assert the delta is exactly the
# ten rows v1.4.0 and v1.5.0 closed.
#
# Deliberately NOT in CI. .github/workflows/go-test.yml checks out at
# fetch-depth 1 — its own comment says nothing in the job reads git history — so
# `git archive 3b6855a` would fail there. The CI case is covered by the hermetic
# gate in tools/coverage-audit/fieldbacktest_test.go, which reads a frozen spec
# fixture and needs no history at all. This target is the live counterpart, run
# on demand, and the one that notices when upstream renames a property.
field-backtest:
	@test -x scripts/ci/field-backtest.sh || { echo "make field-backtest: internal maintenance tooling — scripts/ci/field-backtest.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	@scripts/ci/field-backtest.sh

# The two DRIFT-07 report modes, through the same wrapper. --offline on both:
# regenerating the committed report from a cache that changed underneath is how
# a "no diff" check turns into a diff nobody asked for.
#
# report-check is the one a CI job runs — it prints nothing when the document is
# current and exits 1 naming the stale region when it is not. report-write is
# what an author runs after editing the rationale file, and its diff is reviewed
# like any other change.
report-check:
	@test -x scripts/ci/coverage-audit.sh || { echo "make report-check: internal maintenance tooling — scripts/ci/coverage-audit.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	@scripts/ci/coverage-audit.sh --offline --check

report-write:
	@test -x scripts/ci/coverage-audit.sh || { echo "make report-write: internal maintenance tooling — scripts/ci/coverage-audit.sh is not present in this repository (it is internal-only; see public-allowlist.txt)" >&2; exit 2; }
	@scripts/ci/coverage-audit.sh --offline --write-report

clean:
	rm -f revenium

release-dry:
	goreleaser release --snapshot --clean

release-check:
	goreleaser check
