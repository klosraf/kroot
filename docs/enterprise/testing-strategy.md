# Testing Strategy — Kroot

## Principle

Tests exist to make change safe. A test that only restates the implementation is
a liability: it slows refactoring without catching defects. Test **behaviour**
through the public surface of a package.

## The pyramid

| Level | Scope | Tooling | Where |
|---|---|---|---|
| Unit | one package, no I/O | `go test`, table-driven | `*_test.go` beside the code |
| Integration | several packages, real filesystem or DB | `go test` with build tag `integration` | `*_test.go`, `//go:build integration` |
| End-to-end | real binary, real HTTP | Playwright / shell harness | `e2e/` |
| Frontend unit | component and hook logic | Vitest | `web/**/*.test.tsx` |

Unit tests dominate. Integration and E2E tests are reserved for behaviour that
genuinely crosses a boundary and would be misrepresented by mocks.

## Required gates

Every PR runs, and must pass:

```sh
go vet ./...
gofmt -l .
go test -race -cover ./...
govulncheck ./...
```

Integration tests run when the change touches the code they cover:

```sh
go test -tags=integration ./...
```

## Rules

1. **New behaviour requires a test.** A PR that adds a branch without covering
   it will be sent back.
2. **Bug fixes ship with a regression test** that fails on the old code and
   passes on the new code. State in the PR that you verified both states.
3. **Table-driven by default.** Cases live in a slice of structs and run through
   `t.Run`, so failures name the case.
4. **Coverage must not decrease.** There is no global percentage target; a
   meaningful target is per-package and agreed in review. Untested new code is
   the actual problem.
5. **No flaky tests.** A test that fails intermittently is a bug report against
   the test or the code. Quarantine and fix it; never retry it away.
6. **Deterministic by construction.** No `time.Sleep` to synchronise, no
   dependence on wall-clock ordering, no shared state between cases. Use
   `t.Cleanup`, `t.TempDir`, `t.Setenv`, and injectable clocks.
7. **Mocks are seams, not substitutes.** Mock the process boundary (network,
   clock, filesystem), never the function under test.
8. **Test helpers live in `internal/testutil`**, are imported only from
   `_test.go` files, and are blocked from production builds by `depguard`.

## Concurrency

Any code path that shares state across goroutines ships with a test run under
`-race`. A green non-race run proves nothing about a concurrent defect.

## Frontend

- **Vitest** for components, hooks and pure logic.
- **Playwright** for critical user journeys only; keep the suite small enough
  that it stays trustworthy.
- Snapshot tests are discouraged: they assert on output shape rather than
  behaviour, and they approve regressions as often as they catch them.

## Performance regression

A change with a performance claim includes a benchmark:

```sh
go test -bench=. -benchmem ./path/to/pkg
```

`make bench` records before/after numbers so reviewers can check the claim
rather than trust it.
