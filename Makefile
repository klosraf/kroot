BINARY   := kroot
BIN_DIR  := bin
GO       := go

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILT_AT)

# Pinned so that local runs and CI cannot drift apart (see docs/adr/0001-adopt-enterprise-charter.md).
GOLANGCI_VERSION := v2.14.0
GOVULN_VERSION   := v1.8.0
GOLANGCI         := $(BIN_DIR)/golangci-lint
GOVULNCHECK      := $(BIN_DIR)/govulncheck

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

fmt:
	gofmt -l -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "unformatted files:"; gofmt -l .; exit 1; }

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
