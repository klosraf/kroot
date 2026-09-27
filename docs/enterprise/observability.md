# Observability — Kroot

## Logs

- Structured logging via the standard library `log/slog`. No `fmt.Println` in
  library code; `realMain` configures the handler once, before anything can
  fail, so every later failure is reported by the logger it just built.
- Levels: `DEBUG` for local development, `INFO` as the production default, `WARN`
  for recoverable anomalies, `ERROR` for failures a human should look at.
- Level is controlled by `KROOT_LOG_LEVEL` (`debug|info|warn|error`), defaulting
  to `info` when unset. An unknown value is **rejected, not coerced**: the
  process reports `configuration rejected` with the accepted vocabulary on
  stderr and exits `1` (see `api-compatibility.md` § "Exit codes"). The failure
  is reported at `ERROR` — no configured level may suppress the record that
  explains why it was rejected.
- The level follows the **cause**, and agrees with the exit code beside it. A
  failure the caller can fix by re-invoking — a usage error, exit `2` — is the
  caller's and is reported at `WARN` as `kroot: usage error`. Everything else is the
  program's and is reported at `ERROR`. The split is the same one
  `api-compatibility.md` § "Exit codes" draws, so the two cannot disagree; see
  [`docs/adr/0005-severity-of-a-caller-caused-failure.md`](../adr/0005-severity-of-a-caller-caused-failure.md).
- Encoding follows the destination, not a flag: stderr attached to a terminal
  gets human-readable text, stderr redirected to a pipe or a file gets JSON. The
  structured keys are identical either way, and only the encoding is unstabilised
  by `api-compatibility.md`.
- No color codes are emitted, so `NO_COLOR` has nothing to disable and is not
  read. Detection is introduced with the first styled output, not before
  (`docs/adr/0003-cli-first-terminal-application.md`).

Required keys on every record where they apply:

| Key | Meaning |
|---|---|
| `component` | emitting subsystem, stable identifier |
| `err` | the error value, wrapped, never a stringified stack |
| `request_id` | correlation ID for the originating request |
| `duration_ms` | elapsed time for a timed operation |

```go
slog.InfoContext(ctx, "config loaded",
    "component", "config",
    "request_id", reqID,
    "duration_ms", ms,
)
```

Never logged, at any level: credentials, tokens, authorization headers, full
request bodies, or personal data.

## Correlation

A single `X-Request-ID` is accepted from the ingress proxy or generated at the
boundary if absent. It is attached to the `context.Context`, propagated to every
downstream call, and included on every log record for that request.

## Health endpoints

Liveness and readiness answer different questions and must not be conflated:

| Endpoint | Question | Semantics |
|---|---|---|
| `GET /healthz` | is the process alive? | constant `200`, no dependency checks, safe to poll frequently |
| `GET /readyz` | can it serve traffic now? | `200` when every dependency is reachable, `503` otherwise with a reason per dependency |

`/readyz` must not be used as a liveness probe: a brief dependency outage would
otherwise cause the orchestrator to restart a healthy process.

## Metrics

Exposed at `GET /metrics` in Prometheus text format once the HTTP layer exists.
The first metrics worth adding, in order of usefulness:

| Metric | Type | Why |
|---|---|---|
| `kroot_http_requests_total{method,route,status}` | counter | error rate and traffic shape |
| `kroot_http_request_duration_seconds{route}` | histogram | latency percentiles |
| `kroot_build_info{version,commit}` | gauge (always `1`) | correlate behaviour with a release |

Alert on symptoms, not causes: a rising 5xx rate and a saturated p99 latency are
actionable; a single elevated CPU reading is not.

## SLOs

Targets, to be calibrated against real traffic rather than guessed:

| Objective | Target | Window |
|---|---|---|
| Availability | 99.5% of successful responses | 30 days rolling |
| Latency | p99 request latency under an agreed budget | 30 days rolling |
| Error rate | below 1% of requests | 30 days rolling |

An SLO without an error budget is decoration. When the budget is exhausted,
feature work pauses in favour of reliability work.

## Tracing

Distributed tracing is introduced when there is more than one service to
correlate. Until then, `X-Request-ID` plus structured logs provide the same
narrative at a fraction of the operational cost.
