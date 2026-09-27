# Coding Standards — Kroot

Normative rules for code in this repository. `AGENTS.md` states the principles;
this document states the specifics.

## Go

### Formatting and analysis

Enforced, in order, by `make ci`:

| Gate | Tool | Command |
|---|---|---|
| Format | `gofmt`, `gofumpt`, `goimports` | `make fmt-check` |
| Static analysis | `go vet` | `make vet` |
| Lint | `golangci-lint` (pinned v2) | `make lint` |

A lint suppression must be specific and explained:

```go
//nolint:gosec // G404: this PRNG seeds a load-balancing shuffle, not a secret.
```

Blanket suppressions, or suppressions without a reason, fail the `nolintlint`
check.

### Errors

- Return errors; do not panic. `panic` is reserved for programmer errors in
  `init` or genuinely unreachable states.
- Wrap with context using `%w`: `fmt.Errorf("loading config: %w", err)`.
- Inspect with `errors.Is` / `errors.As`. Never compare error strings.
- Sentinel errors are package-level `var ErrFoo = errors.New("...")` values and
  are wrapped, never replaced.
- An error that a caller cannot act on must still be logged or returned — never
  silently dropped. Use `_ =` only with a comment stating why it is safe.

### Interfaces and types

- Define an interface in the package that *consumes* it, not the one that
  implements it.
- Prefer one-method interfaces (`io.Reader`, `http.Handler`) over wide ones.
- Accept interfaces, return concrete types.
- Use `any` (not `interface{}`) for genuinely untyped values and narrow
  immediately.

### Concurrency

- Every goroutine has an owner responsible for its termination and a documented
  cancellation path (`context.Context`, a `done` channel, or a `WaitGroup`).
- Never start a goroutine in a library function without a way for the caller to
  stop it.
- `context.Context` is the first parameter; it is never stored in a struct.
- Protect shared state with a mutex or a channel, never with assumptions about
  scheduling. Any shared-state change ships with a `-race` test.

### Naming

| Kind | Convention | Example |
|---|---|---|
| Package | short, lower-case, no underscores | `envconfig` |
| Exported identifier | PascalCase, doc comment required | `Server`, `NewServer` |
| Unexported identifier | camelCase | `parseFlags` |
| Acronym | all caps inside the name | `HTTPServer`, `UserID` |
| Test file | `<file>_test.go` | `main_test.go` |
| Test helper package | `internal/testutil` (test-only, depguard-enforced) | |

### Package layout

| Path | Purpose |
|---|---|
| `internal/` | private code; not importable from outside the module |
| `cmd/<binary>/` | one directory per binary when there is more than one |
| `pkg/` | code deliberately offered for reuse (optional, rare) |
| `testdata/` | fixtures; ignored by the Go toolchain |
| `web/` | frontend workspace, added when the UI milestone lands |

`main` packages contain wiring only: parse flags, read configuration, construct
dependencies, hand off, handle shutdown. Logic lives in importable packages so it
can be tested without spawning a process.

### Dependencies

- The standard library first. A dependency must be justified in the PR.
- Pin the version in `go.mod`; commit `go.sum`.
- A dependency that is unmaintained, unpublished, or unlicensed is rejected.

## TypeScript (`web/`, when it lands)

- `strict: true`, plus `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes`.
- No `any`. Use `unknown` and narrow with type guards.
- Function components only; props typed with an explicit `interface`.
- No logic in JSX expressions beyond simple ternaries; extract to a hook or a
  named function at module scope.
- State: `useState` / `useReducer` locally, context for cross-cutting concerns.
  No global mutable singletons.
- Styling with Tailwind utilities. No inline `style` objects.
- Biome + `tsc --noEmit` must be clean.

## Documentation

- Every exported identifier carries a doc comment starting with its name.
- Comments explain **why**, not what the code already says.
- Public behaviour changes update their documentation in the same PR.
