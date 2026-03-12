VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

MODULE  = github.com/revenium/revenium-cli/internal/build
LDFLAGS = -X $(MODULE).Version=$(VERSION) -X $(MODULE).Commit=$(COMMIT) -X $(MODULE).Date=$(DATE)

.PHONY: build test test-race lint clean

build:
	go build -ldflags="$(LDFLAGS)" -o revenium .

test:
	go test ./... -v -count=1

test-race:
	go test ./... -v -count=1 -race

lint:
	golangci-lint run ./...

clean:
	rm -f revenium
