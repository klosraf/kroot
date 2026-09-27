# 0005 — A caller's mistake is not the program's failure

- **Status:** accepted
- **Date:** 2026-09-27
- **Deciders:** repository maintainers

## Context

Every failure the binary can report was logged the same way: one `slog.Error`
call with the fixed message `kroot failed`, carrying the error as an `err`
attribute. A caller who mistyped a command and a caller whose filesystem was
unreadable produced records that differed only in the text of `err`.

Measured on a terminal, before this record:

```
time=2026-09-27T10:40:49.713-07:00 level=ERROR msg="kroot failed" err="usage error: unknown command: \"badcmd\"; see \"kroot help\" for the command list"
```

Three defects live in that line.

1. **The attribution is wrong.** `badcmd` is something the caller typed, not
   something the program did. The record says the program failed.
2. **The level carries no information.** A usage error is logged at `ERROR`, so
   `KROOT_LOG_LEVEL=warn` cannot silence it and `error` describes almost every
   diagnostic the binary ever emits. A level a caller cannot act on is not a
   level.
3. **The recovery is buried.** The one part written for a human — *see "kroot
   help"* — is trapped inside a log attribute, behind a timestamp nobody asked
   for, under a heading that misattributes it.

`docs/enterprise/observability.md` settles *where* logs go and *how* they are
encoded. It never says **which failures belong to the program and which to the
caller**, and that gap is the whole defect: with no rule, every failure took the
only available shape.

This needs deciding rather than tidying, because the log level of a failure is
filterable. A caller who scripts against `KROOT_LOG_LEVEL` would come to depend
on a usage error being `ERROR` — or on its not being.

## Decision

1. **The level follows the cause, not the exit code.** A failure the caller can
   fix by re-invoking is the caller's and is logged at `WARN`; everything else is
   the program's and stays at `ERROR`. The existing `exitCodeFor` already draws
   exactly this line for exit codes, so the log level now agrees with the exit
   code it accompanies rather than being chosen separately.

   | Failure | Exit | Level | Example |
   |---|---|---|---|
   | Configuration rejected | 1 | `ERROR` | `KROOT_LOG_LEVEL=verbose` |
   | Runtime failure | 1 | `ERROR` | write failed, dependency missing |
   | Usage error | 2 | `WARN` | unknown command, surplus operand |
   | Cancelled context | 1 | `ERROR` | shutdown already underway |

2. **A usage error says so in the record's own words.** The message names the
   class of failure rather than asserting the program broke: `kroot: usage error`
   instead of `kroot failed`. The `err` attribute keeps the full sentence, so a
   structured consumer still gets the whole message and the record's key set is
   unchanged.

## Consequences

**Positive**

- `KROOT_LOG_LEVEL=warn` now means something: a caller can see a usage error and
  silence it, or silence everything below `ERROR` and still see real failures.
- The record a caller reads names the cause accurately, and the recovery text is
  no longer the only part that reads like it was written for a person.
- The level and the exit code cannot disagree, because both come from one
  classification.

**Negative / costs**

- A change in observable behaviour: a usage error's level moves from `ERROR` to
  `WARN`. Documented as a `MINOR` under `versioning-policy.md` — a diagnostic's
  level is not a stable key, and the keys are what the contract fixes.
- Any consumer that has come to rely on usage errors arriving at `ERROR` — an
  alert rule, say — must be updated. That is the cost of the level being
  meaningful at all, and it is cheaper now, before `v1.0.0`, than after.

**Neutral**

- No new dependency: the levels are `log/slog`'s own.
- Exit codes, stream routing and the error text itself are unchanged.

## Alternatives rejected

- **Keep one level and fix only the wording.** Rewording without deciding the
  model produces a better sentence with identical wrong semantics, and leaves
  `KROOT_LOG_LEVEL=warn` still unable to silence a typo.
- **Log usage errors below `WARN`, at `INFO`.** A caller who mistyped a command
  has not been given information they asked for; `INFO` would put it behind the
  default level, where the default run still hides it.
- **Report usage errors on stdout.** The caller already has the exit code, and
  `api-compatibility.md` reserves stdout for what was asked for. A diagnostic
  there breaks every caller that redirects.
- **Map the level from the exit code numerically** (`2` → `WARN` as an offset).
  It would couple the log level to a numbering accident. The classification is a
  fact about the cause, so it is decided as one.
- **Drop the timestamp on a terminal.** Tempting, and it does put the message
  first. But the timestamp is `slog`'s, not a decision made here, and suppressing
  it is a presentation change to the handler that deserves its own record.

## Notes

- Pinned by `TestUsageErrorIsNotLoggedAsAProgramFailure`, which asserts the level
  and the message for both classes, and by `TestDiagnosticSurvivesRedirection`,
  which asserts the JSON record keeps its keys when stderr is not a terminal.
- Revisit if a failure class appears that is neither "the caller can fix it by
  re-invoking" nor "the program could not do its job" — the two-way split is what
  makes the mapping short, and a third kind is a signal the split is wrong.
- Related: [ADR-0003](0003-cli-first-terminal-application.md) deferred
  `signal.NotifyContext` until a command can block. The cancelled-context row in
  the table above is the level that path will report.

3. **The record's keys do not change.** `time`, `level`, `msg`, `err` are what a
   consumer parses, and this decision moves a value rather than reshaping the
   record. `api-compatibility.md` already fixes the keys; it fixes them here.

4. **The human-readable encoding is untouched.** A terminal still receives a
   `slog` text record, and a redirect still receives JSON with the same keys.
   What changes is that the record no longer misdescribes the failure. The
   question of whether a diagnostic should be a log record at all on a terminal
   is a separate decision about a different surface, and is not answered here.
