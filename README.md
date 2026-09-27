# kroot

<!-- One-line description of what kroot does. -->

Kroot is a full-stack product built to enterprise standards: a Go backend
(implemented here, in the Go modules below) with a TypeScript + React frontend
added under `web/` when the first UI milestone lands.

The project type and the HTTP/persistence layers are deliberately still open.
The engineering standards that govern *how* those decisions are made are not —
see [`AGENTS.md`](./AGENTS.md).

## Start here

| Document | Read it for |
|---|---|
| [`AGENTS.md`](./AGENTS.md) | The binding engineering charter: stack, code standards, gates |
| [`CONTRIBUTING.md`](./CONTRIBUTING.md) | How to propose, review and land a change |
| [`docs/enterprise/`](./docs/enterprise/) | Branching, versioning, release, security, observability, API compatibility |
| [`docs/adr/`](./docs/adr/) | Why things are the way they are |
| [`SECURITY.md`](./SECURITY.md) | Reporting a vulnerability (privately) |
| [`CHANGELOG.md`](./CHANGELOG.md) | What changed, per release |

## Requirements

- **Go 1.23+** — this module is pinned to `go 1.23.12` in [`go.mod`](./go.mod).
- **pnpm** — only for repository tooling (commit linting). Not needed to build or
  run the backend.
- `curl` — used once by `make tools` to fetch a pinned `golangci-lint`.

## Getting started

```sh
make tools      # install pinned dev tools into ./bin (golangci-lint, govulncheck)
make ci         # the local definition of done
make run        # run from source
```

## Commands

| Command | What it does |
|---|---|
| `make help` | list every target |
| `make build` | compile `bin/kroot` with version, commit and build time injected |
| `make run` | run from source (`make run ARGS="version"`) |
| `make fmt` | format sources in place |
| `make fmt-check` | fail if anything is unformatted |
| `make vet` | `go vet ./...` |
| `make lint` | `golangci-lint run` (pinned v2.14.0) |
| `make test` | unit tests |
| `make test-race` | tests with the race detector and coverage |
| `make cover` | coverage summary |
| `make bench` | benchmarks |
| `make vuln` | `govulncheck ./...` |
| `make tools` | install pinned dev tools into `bin/` |
| `make ci` | fmt-check + vet + lint + test-race + vuln + build |
| `make clean` | remove `bin/` and `.tmp/` |

## Usage

```sh
$ kroot version
kroot dev (commit none, built unknown)

$ kroot help
kroot - application skeleton
...
```

Version, commit and build time are injected at link time by `make build` and are
`dev` / `none` / `unknown` for a plain `go run .`. See
[`docs/enterprise/versioning-policy.md`](./docs/enterprise/versioning-policy.md).

## Layout

```
.
├── main.go                     # entry point: flag parsing + command dispatch
├── main_test.go                # table-driven tests for run()
├── go.mod / go.sum
├── Makefile                    # single entry point for every task
├── package.json                # repository tooling only (commit linting)
├── commitlint.config.mjs
├── AGENTS.md                   # engineering charter (binding)
├── CONTRIBUTING.md
├── SECURITY.md
├── CHANGELOG.md
├── .editorconfig
├── .golangci.yaml              # pinned lint configuration (v2 schema)
├── .github/
│   ├── workflows/ci.yaml       # build, vet, lint, race tests, vuln scan, commits
│   ├── pull_request_template.md
│   └── ISSUE_TEMPLATE/
├── docs/
│   ├── adr/                    # architecture decision records
│   └── enterprise/             # engineering policies
└── bin/                        # build artifacts and pinned tools (git-ignored)
```

| Path | Purpose |
|---|---|
| `internal/` | private code; not importable from outside the module |
| `cmd/<binary>/` | one directory per binary, when there is more than one |
| `testdata/` | fixtures; ignored by the Go toolchain |
| `web/` | frontend workspace, added with the first UI milestone |

## Conventions

- Errors are wrapped with `%w` and inspected with `errors.Is` / `errors.As`.
- `context.Context` is the first parameter of anything that does I/O.
- New behaviour ships with table-driven tests in `*_test.go`.
- Conventional Commits with a scope: `feat(server): add readiness probe`.
- Trunk-based development; `main` is protected and always releasable.

The full rules are in [`AGENTS.md`](./AGENTS.md) and
[`docs/enterprise/`](./docs/enterprise/).

## Licence

[MIT](./LICENSE) © 2026 Carlos Tamayo Ponce.

