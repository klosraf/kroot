BINARY   := kroot
BIN_DIR  := bin
GO       := go

# The tag's leading "v" is stripped, so main.version carries the SemVer value
# ("0.1.0") rather than the tag name ("v0.1.0"). docs/enterprise/versioning-policy.md
# documents `kroot version` printing the former, and a release binary that
# contradicted its own versioning policy would be a drift of exactly the kind
# that document exists to prevent.
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' | grep . || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILT_AT)

# Pinned so that local runs and CI cannot drift apart (see docs/adr/0001-adopt-enterprise-charter.md).
GOLANGCI_VERSION := v2.14.0
# x/vuln v1.2.0 and later declare `go 1.25.0`, and v1.8.0 declares `go 1.26.0`.
# This module is pinned to Go 1.23.12, so the scanner is pinned to the newest
# release that still builds with the toolchain the project declares.
GOVULN_VERSION   := v1.1.4
GOLANGCI         := $(BIN_DIR)/golangci-lint
GOVULNCHECK      := $(BIN_DIR)/govulncheck

# Declared, never inherited. With the default GOTOOLCHAIN=auto, Go silently
# downloads whichever toolchain a dependency asks for: the scanner was built by
# Go 1.26.8 while the project declares Go 1.23.12, so the version that actually
# ran the gate appeared nowhere in the repository. Pinning it to `local` turns
# that drift into a loud failure and keeps make and CI on one compiler.
export GOTOOLCHAIN := local

.DEFAULT_GOAL := help

.PHONY: help tools build install run fmt fmt-check vet lint test test-race cover bench vuln tidy clean ci

help:
	@echo "kroot - available targets"
	@echo ""
	@echo "  Development"
	@echo "    build        compile $(BIN_DIR)/$(BINARY) with version metadata"
	@echo "    install      go install into the current GOBIN"
	@echo "    run          run from source (pass ARGS=... for arguments)"
	@echo "    tidy         tidy go.mod and go.sum"
	@echo ""
	@echo "  Quality"
	@echo "    fmt          format sources in place"
	@echo "    fmt-check    fail if any source is unformatted"
	@echo "    vet          go vet ./..."
	@echo "    lint         golangci-lint run ($(GOLANGCI_VERSION))"
	@echo "    test         go test ./..."
	@echo "    test-race    go test -race -cover ./..."
	@echo "    cover        run test-race and print the coverage summary"
	@echo "    bench        run benchmarks"
	@echo "    vuln         govulncheck ./..."
	@echo "    ci           fmt-check + vet + lint + test-race + vuln + build"
	@echo ""
	@echo "  Housekeeping"
	@echo "    tools        install pinned dev tools into $(BIN_DIR)/"
	@echo "    clean        remove build artifacts"

tools: $(GOLANGCI) $(GOVULNCHECK)

$(GOLANGCI):
	@mkdir -p $(BIN_DIR)
	@echo "==> installing golangci-lint $(GOLANGCI_VERSION)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(BIN_DIR) $(GOLANGCI_VERSION)

$(GOVULNCHECK):
	@mkdir -p $(BIN_DIR)
	@echo "==> installing govulncheck $(GOVULN_VERSION)"
	@GOBIN=$(abspath $(BIN_DIR)) $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULN_VERSION)

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

install:
	$(GO) install -trimpath -ldflags "$(LDFLAGS)" .

run:
	$(GO) run . $(ARGS)

# Format through golangci-lint's configured formatter set (gofmt, gofumpt and
# goimports, see .golangci.yaml). That is the set the lint gate and CI enforce and
# the set docs/enterprise/coding-standards.md documents. Calling plain `gofmt`
# here accepted files that gofumpt rejects, so `make fmt` could leave a tree that
# then failed `make lint`, and the two gates disagreed about "formatted".
fmt: $(GOLANGCI)
	$(GOLANGCI) fmt

# --diff makes the check non-destructive and exits non-zero on any difference.
fmt-check: $(GOLANGCI)
	@$(GOLANGCI) fmt --diff

vet:
	$(GO) vet ./...

lint: $(GOLANGCI)
	$(GOLANGCI) run

test:
	$(GO) test ./...

test-race:
	@mkdir -p $(BIN_DIR)
	$(GO) test -race -covermode=atomic -coverprofile=$(BIN_DIR)/coverage.out ./...

cover: test-race
	$(GO) tool cover -func=$(BIN_DIR)/coverage.out | tail -1

# -run=XXX matches no test, so only benchmarks are executed.
bench:
	$(GO) test -run=XXX -bench=. -benchmem ./...

vuln: $(GOVULNCHECK)
	$(GOVULNCHECK) ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR) .tmp

ci: fmt-check vet lint test-race vuln build
