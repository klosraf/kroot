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

# The date of HEAD in UTC, or empty in a tree with no commits. `package` builds
# with this instead of the wall clock, so two runs over one commit produce the
# same bytes — what docs/enterprise/release-process.md asks of a release build.
# The default `build` keeps the wall clock on purpose: a developer's build should
# say when it happened, not when the commit did.
COMMIT_DATE := $(shell TZ=UTC git show -s --format=%cd --date=format:%Y-%m-%dT%H:%M:%SZ HEAD 2>/dev/null)

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

.PHONY: help tools build install run package fmt fmt-check vet lint test test-race cover bench vuln tidy clean ci

help:
	@echo "kroot - available targets"
	@echo ""
	@echo "  Development"
	@echo "    build        compile $(BIN_DIR)/$(BINARY) with version metadata"
	@echo "    install      go install into the current GOBIN"
	@echo "    run          run from source (pass ARGS=... for arguments)"
	@echo "    package      build the release artefacts into dist/ (PACKAGE_VERSION=...)"
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

# --- Release packaging -------------------------------------------------------
#
# Builds the artefacts named in docs/enterprise/release-process.md §Artifacts: the
# binary as kroot_<version>_<os>_<arch>, SHA256SUMS, the build metadata, the
# changelog excerpt, and an installable tarball carrying the man pages and the
# completion scripts this same binary generates.
#
# PACKAGE_VERSION defaults to VERSION and is passed into the build, so the
# artefact name and what the binary reports can never disagree — the drift the
# LDFLAGS comment above warns about. That default is currently an abbreviated
# commit rather than a release number, because the v0.1.0 tag cannot be reached
# from any branch; see docs/enterprise/release-process.md. For a candidate:
#
#     make package PACKAGE_VERSION=0.2.0-rc.1
#
# sbom.spdx.json is listed in the same section and is deliberately not produced:
# no SBOM generator is available, and a hand-written file with that name would
# claim a standard nothing has checked it against. build-info.txt is what the Go
# toolchain can state truthfully.
PACKAGE_VERSION ?= $(VERSION)
PKG_OS    := $(shell $(GO) env GOOS)
PKG_ARCH  := $(shell $(GO) env GOARCH)
PKG_NAME  := $(BINARY)_$(PACKAGE_VERSION)_$(PKG_OS)_$(PKG_ARCH)
PKG_DIR   := dist
PKG_ROOT  := $(CURDIR)/.tmp/package
PKG_STAGE := $(PKG_ROOT)/$(BINARY)-$(PACKAGE_VERSION)

package:
	VERSION=$(PACKAGE_VERSION) BUILT_AT=$(COMMIT_DATE) $(MAKE) build
	@rm -rf "$(PKG_STAGE)"
	@mkdir -p "$(PKG_STAGE)/bin" "$(PKG_STAGE)/share/man/man1" "$(PKG_DIR)"
	@mkdir -p "$(PKG_STAGE)/share/completion/bash" "$(PKG_STAGE)/share/completion/fish" "$(PKG_STAGE)/share/completion/zsh"
	@# Both inventories are read from the binary's own generated output rather than
	@# restated here, so adding a command or a shell needs no edit to this file.
	@# An inventory that cannot be read is a failure rather than a package that
	@# quietly ships fewer pages than the program documents.
	@commands=$$(./$(BIN_DIR)/$(BINARY) help | awk '/^Commands:$$/{p=1;next} p&&/^$$/{exit} p{print $$1}'); \
	if [ -z "$$commands" ]; then \
		echo "package: no commands found in the generated help" >&2; exit 1; \
	fi; \
	shells=$$(./$(BIN_DIR)/$(BINARY) help completion | sed -n 's/^  kroot completion <\(.*\)>$$/\1/p' | tr '|' ' '); \
	if [ -z "$$shells" ]; then \
		echo "package: could not read the supported shells from the usage line" >&2; exit 1; \
	fi; \
	for c in $$commands; do \
		./$(BIN_DIR)/$(BINARY) man $$c > "$(PKG_STAGE)/share/man/man1/$(BINARY)-$$c.1"; \
	done; \
	./$(BIN_DIR)/$(BINARY) man > "$(PKG_STAGE)/share/man/man1/$(BINARY).1"; \
	for s in $$shells; do \
		case $$s in \
			bash) out="$(PKG_STAGE)/share/completion/bash/$(BINARY)" ;; \
			fish) out="$(PKG_STAGE)/share/completion/fish/$(BINARY).fish" ;; \
			zsh)  out="$(PKG_STAGE)/share/completion/zsh/_$(BINARY)" ;; \
			*) echo "package: no install path is known for shell $$s" >&2; exit 1 ;; \
		esac; \
		./$(BIN_DIR)/$(BINARY) completion $$s > "$$out"; \
	done; \
	cp $(BIN_DIR)/$(BINARY) "$(PKG_STAGE)/bin/$(BINARY)"; \
	cp LICENSE README.md CHANGELOG.md "$(PKG_STAGE)/"
	@cp $(BIN_DIR)/$(BINARY) "$(PKG_DIR)/$(PKG_NAME)"
	@$(GO) version -m $(BIN_DIR)/$(BINARY) > "$(PKG_DIR)/build-info.txt"
	@awk '/^## \[Unreleased\]/{p=1;next} /^## \[/{p=0} p' CHANGELOG.md > "$(PKG_DIR)/CHANGELOG-excerpt.md"
	@# COPYFILE_DISABLE keeps macOS from adding ._AppleDouble entries, which would
	@# otherwise make the same commit package differently per platform.
	@COPYFILE_DISABLE=1 tar --format=ustar --owner=0 --group=0 --numeric-owner \
		-czf "$(PKG_DIR)/$(PKG_NAME).tar.gz" -C "$(PKG_ROOT)" "$(BINARY)-$(PACKAGE_VERSION)"
	@cd "$(PKG_DIR)" && if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum $(PKG_NAME) $(PKG_NAME).tar.gz build-info.txt CHANGELOG-excerpt.md > SHA256SUMS; \
	else \
		shasum -a 256 $(PKG_NAME) $(PKG_NAME).tar.gz build-info.txt CHANGELOG-excerpt.md > SHA256SUMS; \
	fi
	@echo "packaged $(PKG_NAME) in $(PKG_DIR)/"

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
