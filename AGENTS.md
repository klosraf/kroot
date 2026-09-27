# Kroot — Enterprise Engineering Charter

This repository is maintained to enterprise-grade standards. Every contributor —
human or automated — must comply with the rules below. Where this charter and a
local habit disagree, **the charter wins**. Deviations require an ADR in
[`docs/adr/`](./docs/adr/).

## 0. Scope

Kroot is a **full-stack product**:

- **Backend** — Go 1.23.x, this module (`github.com/klosraf/kroot`).
- **Frontend** — TypeScript + React, added under `web/` when the first UI
  milestone lands.

Everything that ships from this repository is typed, tested, observable, secure
and reproducible. "It works on my machine" is not a delivery standard.

## 1. Stack & toolchain

| Layer | Choice | Status |
|---|---|---|
| Language (backend) | Go 1.23.12 (`go.mod`) | **frozen** |
| Language (frontend) | TypeScript, `strict: true` | **frozen** |
| UI framework | React 18 + Vite + Tailwind CSS | **frozen** |
| JS package manager | pnpm (never npm/yarn) | **frozen** |
| Documentation | Markdown under `docs/` | **frozen** |
| HTTP layer | Go standard library first | to decide (ADR) |
| Persistence | none yet | to decide (ADR) |

Rules:

- No new language, runtime, framework or database without an ADR.
- Dependencies are added sparingly. Every new dependency needs a justification in
  the PR: what it does, why the standard library is insufficient, and which
  alternatives were rejected.
- Go dependencies are pinned in `go.mod` and `go.sum` is committed.

## 2. Naming

- Binary `kroot` · module `github.com/klosraf/kroot` · env `KROOT_*` · data dir
  `~/.kroot`.
- Branch names, commit types and PR titles follow
  [`docs/enterprise/branching-strategy.md`](./docs/enterprise/branching-strategy.md).

## 3. Code standards

**Go** — enforced by `make ci`:

- `gofmt` + `gofumpt` clean, `go vet` clean, `golangci-lint` clean.
- Errors are values: wrap with `%w`, inspect with `errors.Is` / `errors.As`.
  Never swallow an error; never `panic` outside `init` or impossible states.
- `context.Context` is the first parameter of anything that does I/O or blocks.
- Interfaces are declared by their consumer, kept small, satisfied implicitly.
- Exported identifiers carry doc comments starting with the identifier name.
- No `init()` side effects, no mutable package-level state, no `unsafe` without
  an ADR.
- Every goroutine has a documented owner and a cancellation path.

**TypeScript** (when `web/` lands):

- `strict: true`; no `any` without an inline justification comment.
- React 18 function components only; props typed with explicit interfaces.
- Biome + `tsc --noEmit` clean; Tailwind utilities over inline styles.

Details: [`docs/enterprise/coding-standards.md`](./docs/enterprise/coding-standards.md).

## 4. Testing

- New behaviour requires tests. A bug fix requires a regression test that fails
  before the fix and passes after.
- Table-driven tests are the default shape.
- `go test -race -cover ./...` must pass, and coverage must not decrease.
- Frontend: Vitest for units, Playwright for critical user flows.
- Test behaviour, not implementation. A mock never replaces the subject under test.

Details: [`docs/enterprise/testing-strategy.md`](./docs/enterprise/testing-strategy.md).

## 5. Security

- No secrets in the repository, ever. Configuration arrives through `KROOT_*`
  variables or mounted files; `.env` files are git-ignored.
- All input crossing a trust boundary is validated and bounded.
- SQL, once it exists, is parameterised — never string-interpolated.
- `govulncheck ./...` runs in CI on every pull request: the job has no path
  filter, so a documentation-only change runs it too.
- Vulnerabilities are reported privately per [`SECURITY.md`](./SECURITY.md).

Details: [`docs/enterprise/security-policy.md`](./docs/enterprise/security-policy.md).

## 6. Observability

- Structured logs through `log/slog`; no `fmt.Println` in library code.
- Every request carries a correlation ID from ingress to log line.
- Health endpoints distinguish liveness from readiness.

Details: [`docs/enterprise/observability.md`](./docs/enterprise/observability.md).

## 7. Quality gates

A change is not mergeable until **every** gate passes. `make ci` is the local
definition of done; the CI workflow is authoritative.

| Gate | Command |
|---|---|
| Formatting | `make fmt-check` |
| Static analysis | `make vet` |
| Lint | `make lint` |
| Tests + race + coverage | `make test-race` |
| Vulnerability scan | `make vuln` |
| Build | `make build` |

## 8. Git & delivery

- Trunk-based: `main` is protected and always releasable.
- Short-lived branches: `feat/`, `fix/`, `chore/`, `docs/`, `release/`, `hotfix/`.
- Conventional Commits, validated by commitlint.
- SemVer (`vMAJOR.MINOR.PATCH`); releases are tagged and published from CI.
- One concern per PR, squash-merged into `main`. Never merge `main` into a
  feature branch — rebase instead.
- PRs stay under ~500 changed lines where possible; a large diff needs a reason.

## 9. Documentation

- Changes to the CLI, HTTP API or `KROOT_*` variables update their docs **in the
  same PR**.
- Architectural decisions are recorded in `docs/adr/NNNN-title.md`. ADRs are
  append-only: a superseded ADR stays and points at its replacement.
- `CHANGELOG.md` follows Keep a Changelog and is updated for every
  user-visible change.
