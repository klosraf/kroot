# Kroot — UI/UX Elevation Prompt (Enterprise Standard)

> **What this file is.** A reusable, self-contained *prompt* for raising the
> interface quality of a product to a mature, production-ready standard, bound to
> this repository's real constraints and to a diagnosis reproduced against the
> code on 2026-09-27.
>
> **What it is not.** Not a rewrite of the engineering charter, and not a second
> register of what was already fixed. The register of resolved work for kroot
> lives in [`ux-elevation-prompt.md`](./ux-elevation-prompt.md) §8 and is
> authoritative for anything this document lists as already done. Where the two
> describe the same code, the register wins and this file is corrected in the
> same pull request (`AGENTS.md` §9).

---

## 0. How to use this document

**Audience.** A contributor — human or automated — who must make *mature, solid,
fluid, refined, production-ready* mean something **measurable here**, rather than
something admired.

**Order of execution.**

1. Read §1 (what the product is) and §2 (what binds the work).
2. Read §3 and **reproduce every finding against the code** before touching it.
   A finding that no longer reproduces is closed, not worked around.
3. Take work items from §5 and §6 in the order given. One item is one pull
   request, classified under `api-compatibility.md` before the code is written.
4. Hold every surface — *including the text it prints* — to §4.
5. Prove it with §7. An item ships with an artefact: a test name, a transcript, a
   linter's output, a measured number.
6. Open §6 only when a UI milestone has its own ADR. It is a specification, never
   an authorisation.

**Status legend.** *Resolved* — done and verified, evidence in the register.
*Open* — nobody has done it. *Decision* — changes a contract; needs a recorded
answer before code. *Deliberate* — intended; re-opening it wastes a review.

**Evidence states** (closed set, used throughout): *Verificado* · *Fallido* ·
*No ejecutado* · *Bloqueado* · *No aplica*. A skipped validation is **No
ejecutado** — never *Verificado*. Never claim a visual or manual review that a
single agent did not perform.

---

## 1. Product context

Kroot is a **terminal-first CLI application** built on the Go standard library
([ADR-0003](../adr/0003-cli-first-terminal-application.md)). One command registry
renders three surfaces — help text, roff manual pages, and shell completion
scripts — so a command cannot be reachable yet undocumented, and the manual cannot
name a flag the binary does not accept. **That invariant is the starting point of
every change in this document.**

| Surface | What the caller sees | Stability |
|---|---|---|
| `kroot` / `kroot help [cmd]` | Command list, or one command's usage and prose | Wording and layout explicitly **not stable** |
| `kroot man [cmd]` | A roff manual page on stdout, same registry | `man kroot`, `man kroot-<cmd>` |
| `kroot completion <shell>` | A completion script on stdout, commands baked in | Regeneration after upgrade is part of the contract |
| `kroot version`, `-version` | Version, commit, build time, toolchain | Build metadata **is** a guarantee |
| `KROOT_LOG_LEVEL` | Log detail on stderr; unknown value rejected pre-dispatch | Name + accepted values **stable** |
| Exit `0` / `1` / `2` | Success / runtime failure / usage error | **Stable** — scripts branch on them |
| Streams | stdout = what was asked for; stderr = diagnostics | **Stable** |

**Not present, on purpose:** no HTTP surface, no persistence, no frontend, no
colour, no pager, no TUI. Each remains behind its own ADR. A prompt that assumed
otherwise would be specifying a product that does not exist.

**The interface concern is written twice** — once for the terminal that exists
(§5) and once for a `web/` that does not (§6) — so a second interface cannot
quietly adopt a different standard.

---
## 2. Binding constraints (distilled, not invented)

Restated from the repository's own rules so this file can be used alone. **Where a
line and its source disagree, the source wins.**

| Rule | Source |
|---|---|
| **Stack frozen:** Go 1.23.12 · TypeScript `strict` + `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes` · React 18 · Vite · Tailwind · pnpm only · Markdown under `docs/` | `AGENTS.md` §1 |
| **No new language, runtime, framework or database without an ADR.** Dependencies are sparing: each justified with what it does, why the stdlib is insufficient, which alternatives were rejected | `AGENTS.md` §1, `coding-standards.md` |
| **Names fixed:** binary `kroot` · module `github.com/klosraf/kroot` · env `KROOT_*` · data dir `~/.kroot` | `AGENTS.md` §2 |
| **Clean by tool, not by eye:** `gofmt`/`gofumpt`/`goimports`, `go vet`, `golangci-lint` — all enforced by `make ci` | `coding-standards.md` |
| **Errors are values:** wrap with `%w`, inspect with `errors.Is`/`errors.As`, never compare strings, never swallow. `panic` only in `init` or unreachable states. A discarded error needs a comment saying why it is safe | `coding-standards.md` |
| **`context.Context` first** on anything that blocks; never stored in a struct; every goroutine has an owner and a cancellation path | `AGENTS.md` §3 |
| **Interfaces declared by the consumer**, small, satisfied implicitly; exported identifiers carry doc comments starting with the identifier name | `coding-standards.md`, `AGENTS.md` §3 |
| **Lint suppressions specific and explained**; blanket or unexplained ones fail `nolintlint` | `coding-standards.md` |
| **Tests:** new behaviour ships tests; a bug fix ships a regression test that fails before and passes after; table-driven by default; `-race`; coverage must not decrease; deterministic by construction; helpers live in `internal/testutil` | `AGENTS.md` §4, `testing-strategy.md` |
| **Security:** no secrets ever; config via `KROOT_*`; validate and bound everything crossing a trust boundary; `govulncheck ./...` on every PR with no path filter | `AGENTS.md` §5 |
| **Observability:** `log/slog` only, no `fmt.Println` in library code; stderr carries logs, human-readable on a TTY and JSON when redirected | `AGENTS.md` §6, `observability.md` |
| **Gates:** `make ci` = `fmt-check` + `vet` + `lint` + `test-race` + `vuln` + `build`. Linux on every PR, macOS on push to `main` ([ADR-0002](../adr/0002-keep-macos-verification-off-pull-requests.md)) | `AGENTS.md` §7 |
| **Git:** trunk-based, `main` protected; branches `feat/ fix/ chore/ docs/ release/ hotfix/`; Conventional Commits **with a scope**, commitlint-validated; one concern per PR, < ~500 changed lines | `AGENTS.md` §8, `branching-strategy.md` |
| **Docs move with the change:** README, generated manual and `CHANGELOG.md` update in the **same PR** as any user-visible change; ADRs are append-only | `AGENTS.md` §9 |

---
## 3. Diagnosis — reproduced against the code, 2026-09-27

Measured, not remembered. Each row is reproducible with the probe below.

### 3.1 Findings and their state

Reproduced 2026-09-27 against the tree at `d605a2a`. All four were open at the
time of the diagnosis and all four are now **resolved**; the evidence is in §5.

| ID | Finding | Consequence | State |
|---|---|---|---|
| **UX-15** | The `Flags:` block of `kroot help` is emitted by `flag.PrintDefaults()`, which writes a **literal TAB** and a 4-space hanging indent: `'Flags:\n  -version\n    \tprint the version and exit\n'`. The `Commands:` block directly above uses two leading spaces and a computed name column | Two blocks in one screen obey **different** typographic rules. §4.3 mandates the computed-column treatment for every name/summary list, so the product violates the standard written beside it — and a TAB is the one whitespace character that survives every reindent, tab-stop and paste | **resolved** (CLI-11) — *not stable* per §4.9 |
| **UX-16** | General help ends at the flag list. It never mentions `kroot help <command>`, the exit-code table, or the manual | The first-run state (§4.5) is half-served: a first-time caller gets orientation but **no path deeper**. The discovery affordance that makes the other three commands reachable is the one thing absent from the screen | **resolved** (CLI-12) — *not stable* |
| **UX-17** | `kroot help a b` and `kroot man a b` answer `usage error: help takes at most one command name, got 2` — **naming no next step** | A direct violation of this document's own non-negotiable §4.1 #3. UX-04 fixed the unknown-command path; the **arity** path was never audited, leaving the one failure with nothing to try | **resolved** (CLI-13) — *not stable* |
| **UX-18** | On a TTY a caller typo is logged as a log record: `time=… level=ERROR msg="kroot failed" err="usage error: unknown command: \"badcmd\"; see \"kroot help\" …"` | Three defects in one line. (a) A **caller's** mistake is reported as the **program's** failure. (b) It is emitted at `ERROR`, so `KROOT_LOG_LEVEL=warn` cannot silence it and `error` means nothing. (c) The recovery — the actual UX payload — is buried in an `err=` attribute of a log envelope, behind a timestamp the human did not ask for | **resolved** (CLI-14) — decided in [ADR-0005](../adr/0005-severity-of-a-caller-caused-failure.md) |

The probes below are kept as written. Run against a fixed tree they now report
the corrected behaviour, which is what makes them regression checks rather than
a description of one moment.

```sh
# UX-15 — the TAB, byte-exact. BSD grep as well as GNU, because a check that only
# runs on a maintainer's Linux is a check nobody runs.
kroot help | LC_ALL=C grep "$(printf '\t')" && echo 'DEFECT: tab' || echo 'ok'

# UX-16 — the first screen now names where to go next
kroot help | tail -3

# UX-17 — the arity path now names the correct form
kroot help a b 2>&1 | tail -1
kroot man  a b 2>&1 | tail -1

# UX-18 — a caller's typo is WARN and is attributed to the invocation, not the
# program; a real failure stays ERROR. Measured on a terminal, via a pty.
python3 -c "
import pty,os
pid,fd=pty.fork()
if pid==0: os.execv('kroot',['kroot','badcmd'])
d=b''
try:
  while True:
    c=os.read(fd,1024)
    if not c: break
    d+=c
except OSError: pass
os.waitpid(pid,0); print(repr(d))"
```

### 3.2 Root cause, for the class rather than the instance

UX-15, UX-16 and UX-17 share one cause: **the standard is written per surface,
and nothing compares surfaces against each other.** Each block was reviewed on its
own terms — the flag list is "whatever the stdlib prints", the help footer "not
required", arity errors "already carry a reason". The defect exists only *between*
them. Hence §4.1 #8: a screen is reviewed as one screen, and the consistency check
in §7.3 is required proof for any change touching more than one block of the same
surface.

UX-18 is a different class — a **severity-model** gap. The repository documents
*where* logs go and *how* they are encoded, but never *which* failures belong to
the program and which to the caller, so every failure became `ERROR` under one
blanket `msg`. Rewording without deciding the model yields a nicer message with
identical wrong semantics.

### 3.3 Deliberate — recorded so nobody "fixes" them

- **No colour, no pager, no progress, no TUI.** ADR-0003 rejects each until a
  consumer exists. The first long-running or interactive command is that consumer,
  and it brings `NO_COLOR` detection with it — not before.
- **`kroot version -h` prints the *program's* help.** ADR-0004: the global flag set
  owns `-h`; a command's own help is `kroot <command> --help` or
  `kroot help <command>`.
- **Signals do not force an exit code.** The context is threaded;
  `signal.NotifyContext` waits for the first command that can block (ADR-0003).
- **Completion scripts bake in the command list.** Stated in every generated
  header; regeneration after upgrade is the documented upgrade step.
- **A `dev` build reports `dev`/`none`/`unknown`.** A stale default shown honestly
  beats a plausible lie (`main.go`, `manualDate`).
- **Help on stdout exits `0`; diagnostics on stderr.** The caller asked for text
  and received it.
## 4. The experience standard

Every rule here is satisfied by an **observation**, never an intention. A rule that
cannot be checked does not belong in this section.

### 4.1 Non-negotiables

1. **One source of truth per fact.** Command lists come from the registry; exit
   codes from the constants the process returns; accepted configuration values from
   the slice the parser validates against; documentation from the code that defines
   it. New duplication is a defect in the making.
2. **Nothing is silently accepted or silently dropped.** A surplus operand, an
   unknown shell, an unaccepted log level and a truncated write all end in a
   diagnostic and a non-zero exit.
3. **Every failure names the next step** — the fix itself (`did you mean "bash"?`)
   or where to find it (`see "kroot help"`). An arity error is a failure like any
   other (UX-17).
4. **Determinism is a feature.** Same input, same bytes: sorted lists, alignment
   computed from the data. A generated file that reorders itself teaches reviewers
   to ignore its diffs.
5. **Requested output on stdout, everything else on stderr** — on the success *and*
   the failure path.
6. **Text does not over-claim.** No wording promising more than the code keeps; no
   heading over nothing.
7. **Accessibility is a constraint, not a phase.** Nothing may depend on colour,
   cursor position, animation, a wide terminal, or a TTY that may not be there.
8. **A screen is reviewed as one screen.** Every block on a surface obeys the same
   typographic and structural rules, and no block inherits its formatting from a
   third-party default (§3.2).
9. **Every surface has an entry point and an exit.** The first thing a reader sees
   names where to go next; the last thing confirms where the output went.

### 4.2 Voice

- **Sentence case.** One-line summaries carry no trailing period — a summary is not
  a sentence.
- **Imperative for actions** ("print the version and exit"), **declarative for
  facts**.
- **Errors say three things in one message**: what was wrong, what would have been
  accepted, what to do next.
  - `usage error: unknown command: "versioo"; did you mean "version"?`
  - `usage error: completion: unknown shell "tcsh": want one of bash|fish|zsh`
- **No blame, no vagueness.** "Invalid input" without the accepted vocabulary is an
  unfinished message.
- **One word per concept, everywhere**: command, flag, operand, shell, page, caller,
  diagnostic. A synonym is a second concept to a reader.
- **Honesty about scope.** No "server", "sync" or "dashboard" in text or metadata
  until the thing exists.

### 4.3 Typography and rhythm

The terminal's type scale: what is long, what is short, what lines up, what stays
flat on purpose.

| Element | Rule | Why |
|---|---|---|
| Name/summary list (`Commands:`, `Flags:`) | Two leading spaces; the name column padded to the longest name plus two, **computed from the data** | A hard-coded width misaligns the moment an entry is added — and a third-party renderer that emits tabs breaks it today (UX-15) |
| Section headings | `Usage:`, `Commands:`, `Flags:` — a word and a colon, then a blank line | A consistent shape is what makes help skimmable |
| Body text | One idea per line, wrapped for 80 columns | A line the author never chose is a line nobody can review |
| Man page source | Caps for section names, `.TP` for terms, ≤ 80 source columns | `mandoc -T lint` warns past that, and it is the linter of the ecosystem the output belongs to |
| Log records | Structured keys, `lower_snake_case`, values as data | Keys are the stable interface for operators; messages are not |
| Whitespace | **Spaces only.** A literal TAB in human-facing output is a defect: it re-expands per tab-stop, survives reindentation, and breaks byte-exact assertions | The one character that makes two blocks of one screen disagree (UX-15) |
| Emphasis | None: no bold, no underline, no colour codes | Nothing may depend on a capability the destination might lack |

### 4.4 Layout and width

- **80 columns is the design width** — what a man page, a README diff and a split
  terminal all tolerate. Enforced in tests, not by eye.
- **Never reflow authored prose.** roff fills paragraphs at render time; re-wrapping
  fights the author.
- **One blank line between blocks, never two.**
- **The terminal is not the only destination.** Every surface is proven redirected,
  piped, and read by a machine — see §4.7.

### 4.5 States

Every surface declares its states. In a CLI the names change; the obligation does
not.

| Interface state | Terminal equivalent | Required behaviour |
|---|---|---|
| First run | No arguments | Print help: a bare command is a request for orientation, not an error — **including a pointer to the next thing to try** (UX-16) |
| Empty | Nothing to list | Omit the section rather than printing a heading over nothing |
| Loading | A slow or blocking operation | Not applicable yet; when it arrives, progress goes to stderr and only when stderr is a terminal |
| Error | A rejected invocation | Diagnostic on stderr, non-zero exit, **next step named** |
| Partial | A truncated write | Report failure; never claim success for half an artefact |
| Interrupted | A cancelled context | Fail fast; signal wiring arrives with the first blocking command |
| Degraded | A validation tool is absent | Skip visibly on a developer machine, fail loudly in CI; never pass silently |
| Success | The requested output | On stdout, complete, and nothing else |

### 4.6 Feedback and microinteractions

The small moments where a product feels attentive or careless. Here they are named,
bounded and pinned by tests.

| Moment | Mechanism | Rule |
|---|---|---|
| Completing a typed word | TAB in bash, zsh, fish | Must work through the *documented install path*, not a plausible variant. Names are offered only where a name is valid |
| Making a typo | Bounded edit distance | At most three suggestions, ordered by distance then name; a short vocabulary gets a tighter bound, because at two edits every shell name resembles every other |
| Mistyping with no near match | Pointer to the command list | One hint, never two; a suggestion suppresses the pointer |
| Mistyping arity | The accepted count | Names the correct form: `see "kroot help <command>"` (UX-17) |
| Forgetting an argument | The accepted vocabulary | `want one of bash\|fish\|zsh`, never "invalid argument" alone |
| Upgrading the tool | A regeneration notice in every generated file | The caller learns the artefact is stale from the artefact |
| Redirecting or piping | Silence | No progress, no colour, no pager, no cursor movement |
| Reading a failure | Severity and framing | A caller's mistake is not the program's failure; see §4.7 and UX-18 |
| Waiting on a slow operation | stderr, gated on a terminal | Not applicable yet; specified now so the first blocking command cannot invent its own answer |

### 4.7 Accessibility

The user is not only a person at a terminal. It is a pipe, a file, a script, a CI
job, a screen reader, a 40-column window and a colour-blind reader.

- **No dependence on presentation.** No meaning carried by colour, alignment,
  cursor position or weight alone.
- **Every surface works without a TTY.** Injected writers, never
  `os.Stdout`-sniffing inside a command; TTY detection lives in one place and is
  tested through those writers, never by assuming the test process has a terminal
  (ADR-0003).
- **Machine-readability is a first-class output**, not a fallback: valid JSON when
  redirected, stable keys, an exit code a script can branch on.
- **The contract is documented where the caller looks**: `man kroot` for the
  environment and exit codes, `--help` for the surface in hand.
- **No output depends on a wide terminal.** Long values wrap or truncate by a
  documented rule; they never reflow a screen into unreadability.
- **A diagnostic is not a log record.** When a human is present, the message is the
  message; a timestamp, a level and an `err=` attribute are for machines. Severity
  is part of the contract, because a caller filters on it (UX-18, §4.9).

### 4.8 Performance

- **A stated budget beats a vague claim.** The repository requires a benchmark
  behind any performance claim, and the budget exists to catch a *category* of
  regression (an accidental file read, an allocation storm at startup), not to
  police microseconds and produce a flaky gate.
- **Numbers are re-measured in the same change that quotes them.** A figure written
  once and never revisited is a claim nobody can check.

**The recorded baseline** (`BenchmarkRunVersion`, in-process: flags, config,
dispatch, write). Re-measure before quoting; see the register for the machine it
came from.

| Measurement | Budget | Measured reading |
|---|---|---|
| In-process dispatch, ns/op | ≤ 10 µs/op | ~1.55 µs, 2.4 KB, 28 allocs |
| Process-level `kroot version` (fork/exec included) | recorded, not gated | ≈ 14.3 ms — dominated by the OS, not this code |

### 4.9 Compatibility

Every experience change is classified **before** it is made, against
`api-compatibility.md` §2.

| Change | Classification |
|---|---|
| New command or optional flag | MINOR; manual, README and completion regeneration travel with it |
| Help or man **wording, layout, section order** | Not stable — patch-level, still reviewed for voice |
| Exit code or stream semantics | **Stable** — a decision with a migration path and an ADR |
| `KROOT_*` name or accepted values | Name is stable; a new variable with a safe default is MINOR |
| Generated artefact contents | Not themselves stable, but the install path and file name they document are part of the instructions |
| Log record keys | Stable; message text is not |
| **Log level chosen for a failure class** | **Stable** — a caller can filter on it; therefore a **Decision** with an ADR (UX-18) |
| Styled output, progress, paging | A new capability, therefore a new decision with its own ADR |

**Pre-1.0 is not a licence.** While `MAJOR` is `0` the surface may still change, but
a documented behaviour that changes silently is the failure this document exists to
prevent — and the exit-code table already predates `v1.0.0` while being depended on
in practice.

### 4.10 Mapping: interface concern → terminal equivalent

The same standard stated twice, so a frontend cannot quietly adopt a different one.

| Concern | In `web/` | Here, today |
|---|---|---|
| Typography | Type scale, weights, line height | §4.3; 80-column source, spaces only |
| Spacing | Spacing scale, rhythm | §4.4: blank-line discipline, computed padding |
| Layout | Grid, containers, breakpoints | Width budget, block order, stream separation |
| Navigation | Menus, breadcrumbs, deep links | Registry, `help`, `man`, completion |
| States | Loading, empty, error, optimistic | §4.5, from exit codes to omitted sections |
| Feedback | Toasts, inline errors, undo | §4.6: stderr diagnostics, suggestions, exit codes |
| Microinteractions | Hover, focus, transitions | Completion, `did you mean`, regeneration notices |
| Accessibility | WCAG 2.2 AA | §4.7: no colour dependence, no TTY assumption, scripts as users |
| Performance | LCP, INP, CLS, bundle budget | §4.8: startup time, streaming output, benchmark |
| Responsive | Breakpoints, touch targets | Narrow terminals, pipes, redirects |
| Theming | Design tokens | None, deliberately |

---

## 5. Workstream A — the terminal surface

One item is one pull request, classified before the code is written, and finished
when its acceptance criteria are demonstrated with an artefact. **All four items
landed on 2026-09-27**; the table records what each one now guarantees and the
artefact that proves it.

| Item | What it now guarantees | Evidence |
|---|---|---|
| **CLI-11** — the `Flags:` block obeys the screen's own rules | `kroot help` emits no byte `0x09`; the flag name sits in the same computed column as the command names, with the gap measured on the **rendered** `-name` so it is two spaces and not one; a multi-line flag usage is folded so it cannot break the column; the manual still renders each flag's name and usage separately; everything stays within 80 columns | `TestNoSurfaceEmitsATab` (new, sweeps every human surface), `TestPrintFlagsUsesTheCommandColumnRule`, `TestPrintFlagsNormalisesMultilineUsage`, `TestHumanFacingSurfacesStayWithinTheDesignWidth` |
| **CLI-12** — the first screen says where to go next | `kroot help` closes with two lines naming `kroot help <command>` and `man kroot`; both are built from `Program.Name`, and either is omitted when that command is not registered rather than printed empty | `TestGeneralHelpNamesTheNextStep`, `TestGeneralHelpOmitsNextStepsItCannotName` (both new) |
| **CLI-13** — the arity path names a next step | `kroot help a b` and `kroot man a b` each carry the correct form; they still wrap `ErrUsage`, keep the count of what arrived, exit `2`, and write nothing to stdout | `TestArityFailuresNameTheCorrectForm` (new) |
| **CLI-14** — a caller's mistake is not the program's failure | A usage error is logged at `WARN` as `kroot: usage error`; configuration and runtime failures stay at `ERROR`; `KROOT_LOG_LEVEL=error` silences a typo but not a real failure; the record's keys are unchanged, so a redirect still receives parseable JSON carrying the whole message | [ADR-0005](../adr/0005-severity-of-a-caller-caused-failure.md), `TestUsageErrorIsNotLoggedAsAProgramFailure`, `TestWarnSilencesAUsageErrorButNotAProgramFailure`, `TestDiagnosticSurvivesRedirection` (all new) |

**Sequencing, as it was.** CLI-14 is a **Decision** and blocks on an ADR, so it
started first and landed last. CLI-11 landed before CLI-12 so the new footer's own
block obeys the corrected column rule rather than inheriting the defect one commit
later. CLI-13 was independent throughout.

**What this workstream did not do.** No colour, no pager, no progress output, no
TUI, no new flag, no exit-code change. Each is a separate decision (§4.9), and
adding one would have made the change the speculative surface ADR-0003 exists to
prevent.

**Two defects found while implementing, both fixed here rather than deferred.**

- The column width was first measured on the bare flag name while the row rendered
  `-name`, silently shortening the gap to one space on every row. It is invisible
  while one flag exists and appears the moment a longer one is added, so the
  width and the label now come from one `flagLabel` helper and the test asserts
  two spaces rather than the mere absence of a tab.
- The `mandoc` recipe conflated *No ejecutado* with *Fallido*: a missing tool and a
  failing lint took the same branch, so a red build could hide behind a green skip.
  It now checks `command -v` first and reports the lint's exit code. Related: a
  `go build` with no `-ldflags` carries the page date `unknown`, which `mandoc`
  warns on — the dev-build shape, not a defect, since a build with metadata
  injected lints silent. Both verified 2026-09-27.

---

### 5.1 Gate run for this workstream

| Gate | Result |
|---|---|
| `make fmt-check` | clean |
| `make vet` | clean |
| `bin/golangci-lint run` | **0 issues** |
| `go test -race -covermode=atomic ./...` | **ok** — 97.7% in `github.com/klosraf/kroot` (up from 97.6%), **100.0%** in `.../internal/cli` |
| `bin/govulncheck ./...` | No vulnerabilities found |
| `make build` | ok |
| **`make ci`** | **exit 0** |

Each new test was confirmed to **fail against the pre-fix code** and pass after —
the check §7.2 requires, and the reason a green suite here means something. For
CLI-14 the severity level, the level-filtering behaviour and the JSON record were
each reverted independently to confirm all three tests fail rather than only the
obvious one.

Coverage did not decrease: it rose, because the tests added exercised a
previously untested branch of the flag renderer.

---

## 6. Workstream B — `web/` (specification only, gated)

**This section authorises nothing.** `web/` does not exist and arrives only with
its own ADR and milestone (`AGENTS.md` §0–§1; ADR-0003 §3). What follows is the
specification that ADR inherits, written now so the day it is approved the work
starts from a standard rather than from taste. The stack is frozen: TypeScript
`strict` (with `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes`), React
18, Vite, Tailwind, pnpm, Biome + `tsc --noEmit`.

### 6.1 Design tokens — the only place a value exists

No raw hex, px or ms in a component. A value that appears twice is a token.

- **Colour** — a semantic ramp, never a decorative one: `surface`, `surface-raised`,
  `border`, `text`, `text-muted`, `accent`, plus `success` / `warning` / `danger` /
  `info`. Every text pair meets **WCAG 2.2 AA (4.5:1)**; non-text state meets 3:1.
  Light and dark are two token sets, not two sets of overrides.
- **Typography** — one family for UI, one for code, a modular scale (1.200 minor
  third) with a fixed line height per step; **tabular numerals** for anything
  numeric, so columns of figures align.
- **Space** — a 4 px base scale: `1 2 4 8 12 16 24 32 48 64`. Spacing is chosen
  from this scale, never invented per component.
- **Radius, shadow, motion** — two radii, two elevations, and durations from `fast`
  (state change), `base` (enter/exit), `slow` (view change). The bands are an
  initial proposal, adjustable by testing, never a per-component constant.
- **Motion is a token, not a vibe.** Every duration and easing comes from the scale;
  `prefers-reduced-motion: reduce` collapses transforms and transitions to opacity
  or nothing, and nothing depends on an animation completing.

### 6.2 Layout, spacing, density

- **An 8-point grid**, with the 4 px scale permitted for intra-component gaps.
- **A measure of 45–75 characters** for prose; a full-bleed exception for data
  tables, never for paragraphs.
- **Optical alignment, not just mathematical.** Icons and controls align to the cap
  height of adjacent text, not to the baseline of the box.
- **Density is a mode, not a default.** Comfortable and compact share the same tokens
  at different scale steps; a table is never re-laid-out to change density.
- **One max content width** for the reading surface; data surfaces may go wider
  within a bounded container.

### 6.3 Navigation and wayfinding

- **The current location is always visible**, and the route is a real URL: back,
  forward and refresh work, and a form warns before discarding unsaved work.
- **A hierarchy, not a maze**: global → section → item, with the trail available
  where depth exceeds two levels.
- **Search is a first-class route**, not a filter buried in a table — with the result
  count, the active filter and a clear-all that is always reachable.
- **The primary action of each view is unambiguous** — one filled button per view,
  and it is the action the view exists to perform.
- **Keyboard parity**: a complete map covering navigation, the primary action, search
  and dismissal. Focus order follows reading order, is never trapped outside a
  modal, and returns to the trigger when one closes.

### 6.4 Components and their states

Every component ships with **all** of its states before it is used twice. A
component with only a default state gets patched differently on each screen — which
is exactly how a product stops looking like one product.

| State | Requirement |
|---|---|
| Default, hover, active | Distinguishable at a glance, never by colour alone |
| Focus-visible | The single focus ring, never suppressed |
| Disabled | Says *why* on hover or focus; a better pattern is preferred where one exists |
| Loading | Preserves layout: a skeleton with the final dimensions, never a spinner that shifts content |
| Empty | Names what would be here and offers the action that creates it — never "No data" |
| Error | Says what failed, what to do, and whether retry is safe |
| Partial / stale | Says when the data is old and how to refresh it |
| Read-only | Looks interactive only when it is |
| Selected | Distinguished from focus and from hover, and announced, not only coloured |
| Success | Confirms what happened and where the result went |
| No permission | States the missing capability and who could grant it; never a dead control |
| Disconnected | Says the session is stale and offers reconnection; actions that cannot succeed are refused with a reason, not queued silently |

Rules that outlive any component:

- **No dead ends:** every empty and error state offers the next action.
- **Forms:** visible label, help text where the format is non-obvious, validation on
  blur and on submit (never while typing a first character), the error beside its
  field *and* in a summary for screen readers, and the primary action disabled only
  when the reason is obvious.
- **Tables:** sort, filter and pagination have defined, URL-backed state;
  virtualization arrives at a measured threshold, not by habit.
- **Destructive actions** confirm with the object and its consequence named, and
  offer undo where the operation can be reversed.
- **Notifications** are informative, dismissible, never modal, and never the only
  carrier of an error.
- **Numbers** are tabular, consistently precise, explicitly unit'd and locale-aware.

### 6.5 Feedback and microinteractions

The difference between "works" and "feels solid" is almost entirely here.

- **Every action acknowledges within 100 ms.** Anything slower says what is
  happening rather than going silent.
- **Motion has a purpose:** orientation (where did this come from), continuity (same
  object), or feedback (this worked). Without one of the three, remove it.
- **Never chain animations the user must wait for**, and never delay an action to
  finish an effect.
- **Optimistic updates only where rollback is honest** — a toggle, a rename, a local
  setting. Never for anything financial, destructive or server-authoritative.
- **Errors persist until resolved or dismissed**; they do not auto-vanish.

### 6.6 Accessibility — WCAG 2.2 AA, enforced in CI

- Full keyboard operability; visible focus on every interactive element.
- Landmarks and a correct heading hierarchy; a skip link; `lang` set.
- Colour contrast 4.5:1 text / 3:1 non-text; **never colour alone** as a signal.
- Every input has a programmatic label; errors are associated via `aria-describedby`
  and announced in a live region.
- Touch targets ≥ 44×44 px with adequate spacing.
- `axe-core` runs in CI on every component and route and **fails the build**; a
  suppressed rule carries a comment naming the reason.
- Manual passes: keyboard only, and a screen reader over the primary journey — both
  recorded as evidence, never assumed.

### 6.7 Responsive design

- **Mobile-first**, with breakpoints derived from the layout's needs rather than
  device names: `sm 640 · md 768 · lg 1024 · xl 1280 · 2xl 1536`.
- **Content-driven breakpoints.** A table becomes cards because the content needs
  it, not because a width was crossed.
- **Tested at 320, 768, 1024 and 1440** — plus 200% zoom and a 320 px viewport with
  no horizontal scroll, both accessibility requirements rather than preferences.
- **Every layout works in one column.** If it only works in three, the mobile
  layout is a separate design, not a reflow.

### 6.8 Performance budgets

Enforced in CI, measured on a mid-tier device with a throttled CPU.

| Metric | Budget |
|---|---|
| LCP | < 2.0 s (p75) |
| INP | < 200 ms (p75) |
| CLS | < 0.05 |
| Initial JS (gzip, route level) | < 200 KB |
| Fonts | `font-display: swap`, ≤ 2 weights, subset, preloaded only above the fold |
| Images | Modern formats, `width`/`height` set, lazy below the fold, priority only for the LCP image |
| Third-party script | Each justified, with a budget line per addition |

Defaults rather than options: route-level code splitting, virtualised long lists
above a measured row count, debounced search, memoised expensive derived state, and
no layout-shifting work after first paint.

### 6.9 Compatibility and locale

- Baseline: the two most recent versions of Chrome, Firefox, Safari and Edge; iOS
  Safari and Android Chrome. Anything older needs a reason.
- **Progressive enhancement:** the core journey works with JavaScript disabled or
  failed; forms submit without it.
- No feature is assumed from a browser version alone — check the capability.
- **Locale:** dates, times, numbers and currency via `Intl`, never string-built.
  Layouts accommodate 30% longer strings (German) and right-to-left mirroring.
- Graceful degradation per feature, with the browser matrix in the README.

### 6.10 Definition of done for this workstream

1. Tokens exist and are used; no raw values in components.
2. Every state in §6.4 is implemented, reachable and screenshot-documented.
3. Keyboard journey, screen-reader pass and `axe` clean — all three recorded.
4. Budgets in §6.8 measured in CI and passing.
5. Breakpoints in §6.7 verified, plus 200% zoom with no horizontal scroll.
6. Every new string exists in **both** `en` and `es`; parity passes.
7. Accessibility findings are CI-enforced, not documented.
8. `biome`, `tsc --noEmit` and Vitest clean; the critical journey has a Playwright
   end-to-end test.

---

## 7. Verification and gates

### 7.1 The floor

`make ci` must exit 0 — formatting, `go vet`, `golangci-lint`, `-race` tests with
coverage that did not decrease, `govulncheck`, and the build. A change that does
not clear the floor does not reach review, regardless of how it looks.

### 7.2 What counts as evidence

| Claim | Evidence that closes it |
|---|---|
| Behaviour changed | A **test name** — the test fails before the change and passes after |
| Output changed | A **byte-exact transcript** from the built binary (not `go run`, which substitutes its own exit code) |
| A generated artefact is right | The **tool that consumes it**: `bash -n`, `zsh -n`, a driver that actually completes, `mandoc -T lint` and a parse-tree read-back |
| A layout is within budget | A **test** asserting the bound, sweeping every surface |
| A number is quoted | The **benchmark output** and the machine it came from |
| Documentation is right | A **run through the documented path** — for zsh, installed on a temporary `fpath` and read by `compinit` |
| A fix that **adds** an interface | A **before/after comparison**: the test cannot run against the old tree, so build the parent commit, drive both builds through the same probe, and paste both |
| A tool was unavailable | *No ejecutado*, with the tool named — never *Verificado* |
| Someone reviewed it | Named. A single agent's self-review is not a review, and saying so is part of the evidence |

### 7.3 Which proof a change owes

| Change | Required proof |
|---|---|
| A new command | Registry entry with `Usage` and `Long`; a `TestRun` case; and the command reaching help, the manual and every completion script |
| A new flag | Rendering in help and in the manual, from the flag set rather than a second description |
| **A block on a surface with other blocks** (CLI-11, CLI-12) | The consistency test in the same PR: no TAB, one column rule, one blank-line rule — asserted across every block of that surface, not only the one edited |
| Changed error wording | A test asserting the **presence of the recovery**, not the whole sentence, so the wording stays editable |
| A change to exit codes, streams or **log severity** | A recorded decision and an ADR; then tests asserting both stdout emptiness and the code |
| A generated artefact | The consuming tool, through the documented path (§7.2) |
| A configuration variable | Parsing, rejection of an unknown value, and the manual's vocabulary — all from one definition |
| New output text | The width and escape-byte guards, extended to a **TAB guard** |
| A frontend component | Vitest for logic and hooks, `axe` in CI, one end-to-end journey, a manual keyboard pass |
| Documentation only | The link check, a transcript if it contains instructions, and a `CHANGELOG.md` entry when the wording is user-visible |
| A figure quoted in this document | Re-measured in the same change — a number written once and never revisited is a claim nobody can check |

### 7.4 Recipes

Copy-pasteable, because a check nobody can run is a check nobody runs.

```sh
# UX-15 — no TAB in any human-facing surface. Written for BSD grep as well as GNU,
# because a check that only runs on a maintainer's Linux is a check nobody runs.
kroot help | LC_ALL=C grep "$(printf '\t')" && echo 'DEFECT: tab' || echo 'ok'

# UX-16 — the first screen names where to go next
kroot help | tail -4

# UX-17 — the arity path names a next step
kroot help a b 2>&1 | tail -1
kroot man  a b 2>&1 | tail -1

# UX-18 — a caller typo is not the program's failure, and survives redirection
kroot badcmd 2>&1 >/dev/null | tail -1            # human-readable on a TTY
kroot badcmd 2>&1 >/dev/null | python3 -m json.tool >/dev/null && echo 'ok: JSON on a pipe'

# zsh: install the way the generated header documents, then ask compinit what it
# registered for the binary. An empty answer is the defect (UX-01).
d=$(mktemp -d)
kroot completion zsh > "$d/_kroot"
zsh -f -c 'fpath=($1 $fpath)
           autoload -Uz compinit && compinit -u -d "$1/dump" >/dev/null
           print -r -- "registered=${_comps[kroot]}"' zsh-probe "$d"

# bash: syntax first, then the behaviour a caller actually gets
kroot completion bash > "$d/kroot.bash" && bash -n "$d/kroot.bash"
bash --noprofile --norc -c '
  source "$1"
  COMP_WORDS=(kroot ""); COMP_CWORD=1
  _kroot_completions
  printf "%s\n" "${COMPREPLY[@]}"' bash-probe "$d/kroot.bash"

# fish: not installed everywhere — skip visibly, or fail in CI
kroot completion fish > "$d/kroot.fish" && fish -n "$d/kroot.fish"

# manual: lint, then read the parse tree rather than the rendered overstrike.
# `command -v` first, so a missing tool is *No executed* and a failing lint is
# *Fallido* — never conflated, which is how a red build hides behind a green skip.
#
# A `go build` with no `-ldflags` carries the page date "unknown", and mandoc
# warns on it (exit 2). That is the dev-build shape, not a defect: lint a build
# with metadata injected and it is silent. Verified 2026-09-27 on both.
command -v mandoc >/dev/null \
  && { kroot man > "$d/kroot.1" && mandoc -T lint "$d/kroot.1"; echo "lint exit=$?"; } \
  || echo 'mandoc absent -> No executed'
kroot man | man -l -

# streams and exit codes, on the success path and the failure path
kroot version > "$d/out" 2> "$d/err"; echo "code=$? stdout=$(wc -c < "$d/out")"
kroot version extra > "$d/out" 2> "$d/err"; echo "code=$? stdout=$(wc -c < "$d/out")"
```

The last pair catches the most: a failure that writes to stdout, or a success that
writes to stderr, breaks every caller that redirects.

### 7.5 Review checklist

A reviewer answers in order. A "no" is a request for a change, not a preference.

1. Is the change classified against `api-compatibility.md`, and does the class match
   what actually changed?
2. Does it introduce a second copy of a fact that already has a home?
3. Does **every** new failure name the accepted vocabulary and a next step — arity
   failures included?
4. Is the output deterministic — sorted, computed, stable across runs?
5. Does stdout carry only the requested output, on the success *and* the failure
   path?
6. Is there a test that fails before the change and passes after?
7. Are generated artefacts validated by the tool that consumes them, through the
   path the documentation tells the caller to use?
8. Are the width, escape-byte and **tab** budgets respected?
9. Does every block of the edited surface obey the same typographic rule — no block
   inheriting a third party's formatting?
10. If the surface is a first screen, does it name where to go next?
11. Do `README.md`, the generated manual and `CHANGELOG.md` move in the same PR?
12. Is it one concern, under ~500 changed lines, with a Conventional Commit and a
    scope?
13. If a dependency was added: what it does, why the standard library is not enough,
    which alternatives were rejected?
14. Are the deliberate omissions still omissions — no heading over nothing, no
    promise the code does not keep?
15. Does the wording follow §4.2, including the fixed vocabulary?
16. Would a caller who only ever reads the terminal discover this without reading
    the diff?
17. Does anything depend on colour, cursor position, animation, a wide terminal, or
    a TTY that may not be there?

### 7.6 Anti-patterns, rejected on sight

- **Editing the wording until the test passes.** The test asserts the recovery a
  caller needs; the wording serves the test, not the other way round.
- **Inheriting a third-party renderer's formatting.** `flag.PrintDefaults()` is a
  fine implementation of *its* contract and the wrong one for this screen (UX-15).
- **Sourcing an artefact and calling it installed.** The documented installation is
  the contract; a variant of it proves nothing about what a caller gets.
- **Testing the helper rather than the surface.** A unit test of a private formatter
  cannot show that help, the manual or a script is right.
- **Unbounded suggestions.** A list of guesses is worse than none: the reader then
  has to choose between them.
- **A heading added for symmetry.** A section with nothing under it promises content
  and delivers a heading.
- **Behaviour changed as "polish."** Anything a caller can observe is classified;
  stable items are decided, not tidied.
- **A hard-coded width, name or exit code** that already has a source of truth
  elsewhere — including a second description of a flag.
- **Colour added for emphasis.** Styled output is an ADR with a `NO_COLOR` story,
  not a one-line change.
- **Snapshots standing in for behaviour.** They approve regressions as often as they
  catch them.
- **A silent skip.** If a required tool is missing in CI, the build fails.

## 8. Definition of done

**Ready — the gate before writing any code.** An item that fails this is not
started; it is clarified.

- [ ] The finding is reproduced against the code, and the current behaviour is
      understood rather than assumed.
- [ ] Scope and exclusions are written down, including what the change will *not* do.
- [ ] The compatibility class is decided before the code: stable, not stable, or a
      new capability that needs its own ADR.
- [ ] Every acceptance criterion is observable and names how it will be checked.
- [ ] Dependencies and permissions are available, or the blocker is registered.
- [ ] The test strategy is provided, and a defect fix names the test that fails
      before it.

**Done — a change made under this prompt is finished when every line is true**, and
the pull request says so:

1. Its finding, if any, was confirmed against the code before it was worked on.
2. It is classified against `api-compatibility.md`, and the class matches the
   change.
3. No fact gained a second home; anything that could drift now has one definition.
4. It has a test that fails before the change and passes after, and every generated
   artefact is proven through the documented path.
5. `make ci` exits 0 — formatting, vet, lint, race, coverage, vulnerability scan and
   build — and the coverage did not decrease.
6. The evidence named in §7.2 is attached: test names, transcripts, lint output.
7. `README.md`, the generated manual and `CHANGELOG.md` move in the same pull
   request where the change is user-visible.
8. The §7.5 review checklist is answered, with each "no" resolved or rejected
   explicitly.
9. Any decision the item required exists as a record **before** the code does.
10. Nothing in the change depends on colour, cursor position, animation, a wide
    terminal, or a TTY that may not be there.

---

## 9. Where each fact lives

| Fact | Home |
|---|---|
| Stack, gates, delivery, security, documentation duty | [`AGENTS.md`](../../AGENTS.md) |
| Engineering policies in depth | [`docs/enterprise/`](../enterprise/) |
| Why the product is what it is | [`docs/adr/`](../adr/) |
| Exit codes, streams, and what is stable | [`api-compatibility.md`](../enterprise/api-compatibility.md) |
| Log routing, encoding, correlation | [`observability.md`](../enterprise/observability.md) |
| **The experience standard** (diagnosis, workstreams, verification) | **this file** |
| **What was already resolved, with its evidence** | [`ux-elevation-prompt.md`](./ux-elevation-prompt.md) §8 |

**Rules that keep this honest.**

1. This document **cites, never restates.** When a line and a source disagree, the
   source wins and this file is corrected in the same change.
2. A rule that changes in a source is mirrored here in the same pull request.
3. Experience work has **one standard and one register**: the standard here, the
   register in `ux-elevation-prompt.md` §8. A finding or criterion must not appear in
   both, because two copies of a fact are the defect §4.1 #1 exists to prevent.
4. A claim of completeness is scoped to what the document actually checked.
   "Every finding is resolved" was true of the first pass and wrong of the
   repository, because the second pass found four more by reading whole screens
   instead of blocks. A document that says "nothing is open" must say *open
   where*, or a later reader inherits a guarantee nobody re-measured.
