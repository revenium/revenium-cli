VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

MODULE  = github.com/revenium/revenium-cli/internal/build
LDFLAGS = -X $(MODULE).Version=$(VERSION) -X $(MODULE).Commit=$(COMMIT) -X $(MODULE).Date=$(DATE)

.PHONY: build test test-race lint clean release-dry release-check coverage-audit coverage-audit-offline \
        field-audit field-audit-offline field-backtest

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
coverage-audit:
	scripts/ci/coverage-audit.sh

# The same report with no network access at all, against the cache already on
# disk. This is the form to run repeatedly while working through the gaps.
coverage-audit-offline:
	scripts/ci/coverage-audit.sh --offline

# The field dimension of the same audit, through the same wrapper: one TSV row
# per request-body property or query parameter that appears on only one side of
# the join. Same fetch behaviour as coverage-audit above, so the field report
# and the endpoint report are never built from different cache states.
# Recipe echo suppressed with @, unlike the coverage-audit pair above. Those
# print a human table; these print TSV that 21-06 and the backtest read with cut
# and comm, and make's own echo of the recipe line would arrive as the first row
# of it.
field-audit:
	@scripts/ci/coverage-audit.sh --fields

# The field report with no network access at all. This is the form to run
# repeatedly while working through the rows.
field-audit-offline:
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
	@scripts/ci/field-backtest.sh

clean:
	rm -f revenium

release-dry:
	goreleaser release --snapshot --clean

release-check:
	goreleaser check
