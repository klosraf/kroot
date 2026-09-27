# 0003 — kroot is a terminal-first CLI application

- **Status:** accepted
- **Date:** 2026-09-27
- **Deciders:** repository maintainers

## Context

ADR-0001 deliberately left the project type undecided: CLI, service, or
full-stack. Since then the module has grown into a CLI skeleton with a real
caller contract — exit codes, stream routing, build metadata — while the
policies describe a future service only generically (health endpoints, metrics,
correlation). Before adding runtime behaviour, the product type has to be
decided, because a CLI and a service answer logging, shutdown and configuration
differently. A wrong default here would be load-bearing within a release.

## Decision

1. **kroot is a terminal-first CLI application, standard library only.** The
   product is the `kroot` binary and its caller contract. The contracts already
   written (exit codes, streams, build metadata) become binding on the
   implementation rather than aspirational.
2. **Server mode, if it ever exists, is a subcommand under its own ADR.** The
   HTTP layer and persistence stay undecided until a consumer exists; no
   speculative surface is added for them.
3. **`web/` stays out of scope** until a UI milestone lands (unchanged from
   ADR-0001).
4. **Phase 1 core:** `KROOT_LOG_LEVEL` (the first `KROOT_*` variable; a MINOR
   addition under the versioning policy, since it ships with a safe default),
   a `context.Context` threaded from `main` to `run` — ready to be derived from
   signals the day a command can block — and TTY-aware log encoding
   (human-readable text on a terminal, JSON when stderr is redirected). No
   command blocks or spawns a goroutine yet, so the context is a contract, not
   yet a cancellation path.

## Consequences

**Positive**

- Work has a single direction: every runtime feature is judged as CLI behaviour
  first (piped, redirected, scripted, signalled).
- The existing contracts stop being documentation of intent and start being
  enforced by tests.
- Logging, shutdown and configuration arrive before the first long-running
  command needs them, instead of being retrofitted under it.

**Negative / costs**

- If a service mode lands later, only the CLI core (logging, config, exit codes)
  transfers directly; request handling is new work under its own ADR.
- TTY-dependent behaviour must be tested through injected writers, never by
  assuming the test process has a terminal.
- `signal.NotifyContext` waits for the first long-running command: wiring it
  before anything can block would change observable behaviour (exit codes,
  shutdown ordering) with no test able to observe the difference. The context
  is threaded now; the signal wiring lands with the first blocking command.

**Neutral**

- No new dependency: `os/signal`, `syscall`, `log/slog` and `runtime` are all
  standard library, so the frozen stack is untouched.

## Alternatives rejected

- **Full-stack from day one.** There is no consumer for HTTP, persistence or a
  frontend yet; building them now would be speculative surface with no tests
  that mean anything.
- **Service-first.** It contradicts the investment already made: exit codes,
  stream routing and build metadata are CLI answers, and a service would leave
  them as dead contracts.
- **A TUI framework (bubbletea, lipgloss, survey-style prompts).** Unjustified
  while all output is plain text; a dependency needs a consumer, and there is
  none yet. Revisit with the first interactive command.
- **ANSI color output with `NO_COLOR` detection.** kroot emits no color codes,
  so `NO_COLOR` is honored trivially today. Detection without a consumer would
  be dead code, and dead code is how drift starts. Revisit when styled output
  exists.

## Notes

- Revisit when the first long-running or interactive command lands; that is when
  progress output, paging and colors become real requirements rather than
  speculation.
- `KROOT_LOG_LEVEL` is a MINOR addition under the versioning policy: a new
  configuration variable with a safe default (`info`).
