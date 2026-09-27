# kroot

<!-- One-line description of what kroot does. -->

Kroot is a terminal-first CLI application — decided in
[`docs/adr/0003-cli-first-terminal-application.md`](./docs/adr/0003-cli-first-terminal-application.md),
built on the Go standard library, with the `kroot` binary and its caller contract
as the deliverable. A server mode and a TypeScript + React frontend under `web/`
both remain possible; neither exists, and each would arrive under its own ADR.

The HTTP and persistence layers are deliberately still open. The engineering
standards that govern *how* those decisions are made are not — see
[`AGENTS.md`](./AGENTS.md).

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
kroot dev (commit none, built unknown, go1.23.12)

$ kroot help
kroot - application skeleton
...
```

Version, commit and build time are injected at link time by `make build` and are
`dev` / `none` / `unknown` for a plain `go run .`. The Go toolchain is not
injected: it comes from the runtime, so the binary always names the compiler that
produced it. See
[`docs/enterprise/versioning-policy.md`](./docs/enterprise/versioning-policy.md).

## Manual

`kroot man` writes a manual page in roff source to standard output. With no
argument it documents the program; with a command name it documents that command
alone, under the page name `kroot-<command>`.

```sh
$ kroot man | man -l -                     # read the program page
$ kroot man completion | man -l -          # read one command's page
$ kroot man > share/man/man1/kroot.1       # install it
```

The page is generated from the same command registry that backs `kroot help` and
`kroot completion`, so a command cannot be reachable but undocumented — there is
no second list to keep in step. Each page records the version and build date it
was generated from, and the exit-code table is passed in from the code that
returns those codes rather than restated, so the page cannot disagree with the
binary about them.

Because the page comes from the binary, it is only as current as the binary that
wrote it — regenerate on upgrade. Run `kroot man` through `mandoc -T lint` (or
`groff -man`) to check a generated page; kroot's own tests do exactly that.

## Shell completion

`kroot completion <bash|fish|zsh>` writes a completion script to standard output.
It completes subcommand names, and only in the first operand position — the one
place a subcommand is a valid answer.

```sh
# bash, system-wide (needs root)
$ kroot completion bash > /etc/bash_completion.d/kroot

# zsh, per user
$ kroot completion zsh > "${fpath[1]}/_kroot"

# fish
$ kroot completion fish > ~/.config/fish/completions/kroot.fish
```

The command list is written into the script, so **regenerate it after upgrading
kroot** or a newly added command will not appear. Every generated file says so in
its own header. An unsupported shell is rejected with the accepted list and, for a
near miss, a suggestion — `kroot completion bas` answers `did you mean "bash"?`
and exits `2`.

## Configuration

Configuration arrives through `KROOT_*` environment variables. The first one:

| Variable | Default | Accepted values |
|---|---|---|
| `KROOT_LOG_LEVEL` | `info` (when unset) | `debug`, `info`, `warn`, `error` |

```sh
$ KROOT_LOG_LEVEL=debug kroot version    # detail on stderr, output unchanged on stdout
$ KROOT_LOG_LEVEL=verbose kroot version  # rejected: exit 1, reason on stderr
```

An unknown value stops the process before any command runs: stderr says
`configuration rejected`, and the exit code is `1`. Logs always go to stderr —
stdout stays reserved for what the caller asked for, in human-readable form on a
terminal and JSON when redirected. See
[`docs/enterprise/observability.md`](./docs/enterprise/observability.md) and
[`docs/enterprise/api-compatibility.md`](./docs/enterprise/api-compatibility.md).

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

