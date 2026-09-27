# Kroot — Integrated Experience Prompt

One file, one structure. Everything needed to raise this product's experience to a
mature, production-ready standard — the diagnosis, the standard, the workstreams,
the verification rules and the register of what this pass resolved — integrated
with the documentation this repository already has. Nothing here is a rewrite of
the charter; §2 distills it and §9 maps every source so no rule has two homes.

## 0. How to use this document

**Who it is for.** A contributor, human or automated, that has to know what
*mature, solid, fluid, refined and production-ready* mean **here**, in terms that
can be checked rather than admired.

**Order of execution.**

1. Read §1 for what the product is, and §2 for the rules that bind the work.
2. Read §3 and **confirm each finding against the code** before touching anything.
   A finding that no longer reproduces is closed, not worked around.
3. Take work items from §5 in the order given there. One item is one pull request.
4. Hold everything to §4 — including the text a surface prints, not only the code
   it runs.
5. Prove it with §7. An item is done when its criteria are demonstrated with an
   artefact: a test name, a transcript, a linter's output.
6. Open §6 only when a frontend milestone has its own ADR. It is a specification,
   never an authorisation.

**Status legend.** *Resolved* — done and verified, with the evidence in §8.
*Open* — nobody has done it. *Decision* — changes a contract, needs a recorded
answer before code. *Deliberate* — the current behaviour is intended; re-opening
it wastes a review.

**What this document is not.** Not an authorisation for a frontend. Not a licence
to add colour, pagers or a TUI. Not a rewrite: every open item is small, local and
testable. Not a style guide for taste — where a choice cannot be checked, the
existing convention wins, and a change of taste needs a reason a reviewer can
disagree with.

## 1. Product context

Kroot is a **terminal-first CLI application skeleton** built on the Go standard
library. One command registry renders the help text, the manual pages and the
shell completion scripts; the caller contract — exit codes, streams,
configuration — is documented and covered by tests. The product type is recorded
in [`docs/adr/0003-cli-first-terminal-application.md`](../adr/0003-cli-first-terminal-application.md),
and it is what makes the terminal the product experience rather than a stopgap.

| Surface | What the caller sees | Contract |
|---|---|---|
| `kroot` / `kroot help [command]` | Command list, or one command's usage and description | Wording and layout are explicitly *not* stable |
| `kroot man [command]` | A roff manual page on stdout, from the same registry | `man kroot`, `man kroot-<command>` |
| `kroot completion <bash\|fish\|zsh>` | A completion script on stdout, command list baked in | Regeneration after upgrade is part of the contract |
| `kroot version`, `-version` | Version, commit, build time, toolchain | Build metadata is a documented guarantee |
| `KROOT_LOG_LEVEL` | Log detail on stderr; an unknown value is rejected before any command runs | Name and accepted values are stable |
| Exit codes `0` / `1` / `2` | Success, runtime failure, usage error | Stable; scripts branch on them |
| Streams | stdout carries what was asked for; stderr carries diagnostics | Stable |

**Not present, on purpose:** no HTTP surface, no persistence, no frontend.
ADR-0003 keeps each behind its own decision, and §6 inherits that gate. A prompt
that assumed otherwise would be specifying a product that does not exist — which
is why the interface concerns in §4 are written twice, once for each world.

**One registry, three renderers.** Help, the manual and completion all read the
same command set, so a command cannot be reachable but undocumented, and the
manual cannot name a flag the binary does not accept. Any change to this product
starts from that invariant.

## 2. Binding rules (distilled, not invented)

These are the repository's own rules, restated so this document can be used alone.
Where a line and its source disagree, the source wins — §9 says where each lives.

| Rule | Source |
|---|---|
| **Stack frozen:** Go 1.23.12 for the backend; TypeScript `strict`, React 18, Vite and Tailwind for `web/`; pnpm only; Markdown under `docs/` | `AGENTS.md` §1 |
| **No new language, runtime, framework or database without an ADR** | `AGENTS.md` §1 |
| **Dependencies are sparing.** Each new one is justified: what it does, why the standard library is insufficient, which alternatives were rejected. Standard library first | `AGENTS.md` §1, `coding-standards.md` |
| **Names are fixed:** binary `kroot`, module `github.com/klosraf/kroot`, environment `KROOT_*`, data directory `~/.kroot` | `AGENTS.md` §2 |
| **Go is clean by tool, not by eye:** `gofmt`/`gofumpt`/`goimports`, `go vet`, `golangci-lint`, all enforced by `make ci` | `coding-standards.md` |
| **Errors are values:** wrap with `%w`, inspect with `errors.Is`/`errors.As`, never compare strings, never swallow. `panic` only in `init` or genuinely unreachable states. A discarded error (`_ =`) requires a comment saying why it is safe | `coding-standards.md` |
| **`context.Context` is the first parameter of anything that blocks**, never stored in a struct; every goroutine has an owner and a cancellation path | `AGENTS.md` §3, `coding-standards.md` |
| **Interfaces are declared by the consumer**, kept small, satisfied implicitly; accept interfaces, return concrete types | `coding-standards.md` |
| **Exported identifiers carry doc comments starting with the identifier name**; comments explain *why*, not what the code already says | `AGENTS.md` §3, `coding-standards.md` |
| **Lint suppressions are specific and explained** (`//nolint:gosec // G204: …`); blanket or unexplained ones fail `nolintlint` | `coding-standards.md` |
| **Tests:** new behaviour ships with tests; a bug fix ships with a regression test that fails before and passes after; table-driven by default; `-race`; coverage must not decrease; no flaky tests; deterministic by construction; test behaviour rather than implementation; helpers live in `internal/testutil` and never reach production builds | `AGENTS.md` §4, `testing-strategy.md` |
| **Security:** no secrets in the repository ever; configuration through `KROOT_*` or mounted files; validate and bound everything crossing a trust boundary; `govulncheck ./...` runs on every pull request with no path filter; vulnerabilities reported privately | `AGENTS.md` §5, `SECURITY.md` |
| **Observability:** structured logs via `log/slog`, no `fmt.Println` in library code; logs on stderr, human-readable on a terminal and JSON when redirected; no colour codes are emitted, therefore `NO_COLOR` is not read — it arrives with the first styled output, not before | `AGENTS.md` §6, `observability.md` |
| **Gates:** `make ci` = `fmt-check` + `vet` + `lint` + `test-race` + `vuln` + `build`; CI is authoritative — Linux on every pull request, macOS on a push to `main` | `AGENTS.md` §7, `ADR-0002` |
| **Git:** trunk-based, `main` protected; branches `feat/`, `fix/`, `chore/`, `docs/`, `release/`, `hotfix/`; Conventional Commits with a scope, validated by commitlint; SemVer; one concern per pull request, squash-merged, under ~500 changed lines where possible | `AGENTS.md` §8, `branching-strategy.md` |
| **Documentation travels with the change:** CLI, HTTP or `KROOT_*` changes update their docs in the same pull request; decisions become append-only records in `docs/adr/NNNN-title.md`; `CHANGELOG.md` follows Keep a Changelog for every user-visible change | `AGENTS.md` §9 |
| **Compatibility:** stable means the subcommand and flag names documented in the manual, plus the exit codes. Help wording and layout are explicitly *not* stable. Undefined input "will not be silently accepted" | `api-compatibility.md` |

### 2.1 Adopted external playbook — profile, authority and scope

**Provenance.** `/Users/klosraf/Prototipos/MASTER_APPLICATION_DEVELOPMENT_PLAYBOOK.md`
— "CODEX UNIVERSALIS", 35 sections, Spanish with an English operating companion,
edition 1.0 dated 2026-09-27, consolidating
`/Users/klosraf/.config/ai-unified/DEVELOPMENT_CONSTITUTION.md` (SHA-256 recorded
in its §35). Read on 2026-09-27.

**Authority.** Its §02 states that it does not override the rules in force in the
repository, and that it is not implicit permission to act. So `AGENTS.md` and the
`docs/enterprise/` policies remain binding, and this subsection records which parts
are **adopted**, which are **NO APLICA**, and why. Nothing was copied wholesale:
its own §03 composes requirements as *core + state + product type + language +
platform + data + criticality*, and a four-command Go CLI with no HTTP surface and
no persistence is exactly that selection. Copying 79 KB of universal policy into a
repository of this size would create a second, divergent constitution — the failure
§4.1 exists to prevent.

**Discovery card (playbook §03), filled for this project.** Unknowns are marked
pending, as §03 requires; nothing here is invented.

| Dimension | Value |
|---|---|
| Purpose | A caller contract for a CLI: command registry, help, manual pages, completion scripts, exit codes, streams, configuration |
| Scope | In: the `internal/cli` framework, four commands, `KROOT_LOG_LEVEL`, generated artefacts. Out: HTTP, persistence, frontend, GUI |
| State | Existing, `v0.1.0` released, pre-`v1.0.0` guarantees |
| Repository | `/Users/klosraf/Prototipos/kroot`, branch `main`; this pass sits uncommitted in the working tree |
| Language | Go 1.23.12, no generated code committed |
| Stack | Standard library only; `golangci-lint` v2.14.0, `govulncheck` v1.1.4, pnpm 9.15.4 for commit linting |
| Platform | Terminal — the product type, ADR-0003 |
| Users | Callers scripting from a shell; the accessibility need is machine-readable output, not visual design |
| Data | None: no persistence, no telemetry, no user data |
| Connectivity | None required; the binary never opens a socket |
| Load | Interactive and one-shot; a script pays startup on every call |
| Concurrency | None yet: no command blocks and no goroutine outlives a call |
| Identity | None: no accounts, no permissions, no network |
| Distribution | Built binary or `go install`; CI on Linux for every pull request, macOS on `main` |
| Operation | Developer machines and CI runners; no service, no on-call |
| Risk | **R1** — public information and reversible effects: a wrong exit code or a dead completion script misleads scripts but destroys nothing durable |
| Design | Terminal-first and deliberately unstyled; the system in §6 is a specification, not a decision |
| Languages | English, in code, comments and documentation |
| Validation | `make ci`, `mandoc -T lint`, real bash and zsh drivers, `make bench` |

**Adopted, and what it changes here.**

| From the playbook | Adopted as | Where it lands |
|---|---|---|
| §01 requirement levels — **DEBE / DEBERÍA / PUEDE / CONDICIONAL / NO APLICA / ASPIRACIONAL** | The vocabulary every rule in this document is written in. DEBE is mandatory within its stated scope; a NO APLICA must carry a reason | §2, §4 |
| §19 evidence states — **Verificado / Fallido / No ejecutado / Bloqueado / No aplica** | A closed set replacing the prose "an artefact". A skipped validation is *No ejecutado*, never *Verificado* | §7.2, §8 |
| §19 "never claim visual or manual validation that did not happen", and §01 "do not claim independent review when a single agent performed it" | Everything in §8 was executed and judged by one agent. No independent human review has occurred, and none is claimed | §7.2, §8 |
| §28 phase gates (0–10) | The general ladder. §5.3 is its CLI narrowing: this repository sits in phase 4 — a vertical flow working end to end — with 6 and 7, hardening and refinement, partially entered | §5.3, §10 |
| §29 Definition of Ready and Done | §10 gains a *Ready* pre-gate; the Done list is kept and tightened | §10 |
| §18 CLI performance profile — **time, memory, output size, cancellation** | The budget in §4.8 measures exactly those four. That section now states that LCP, INP and CLS are **NO APLICA** to a CLI — the playbook forbids applying them — and they survive only inside the gated web specification | §4.8, §6.8 |
| §19 adversarial scenarios, CLI subset | Empty, oversized, duplicated, invalid, Unicode and hostile input; lifecycle; distribution (clean install, upgrade); interface (screen reader, long text) — added to the table of required proofs | §7.3 |
| §11 full state inventory and motion budgets | For the web workstream only: the complete inventory, adding *selected*, *success*, *no permission* and *disconnected*, plus the duration bands 80–140, 120–200, 160–240 and 180–300 ms | §6.4, §6.5 |
| §12 "WCAG 2.2 AA within the agreed scope… for native, terminal, voice and XR, add platform-specific guidance" | The web section keeps WCAG 2.2 AA; the terminal section keeps §4.7, which is the platform-specific half | §4.7, §6.6 |

**NO APLICA, with the reason — never with silence.**

| Playbook area | Why it does not apply here |
|---|---|
| §6–§8 environments, toolchain, frameworks, add-ons | The stack is frozen by `AGENTS.md` §1 and the tools are pinned in the `Makefile`; no framework selection is open |
| §13 frontend state and cache, §14 backend APIs and identity, §15 data and transactions | No HTTP surface and no persistence; ADR-0003 keeps both undecided. Adopting them now would be the speculative surface the charter rejects |
| §16–§17 security, privacy and licensing | Security already has its own policy; privacy, licensing and regulatory duties are DEBE only when user data exists, and none does |
| §22 AI, RAG and tool-using agents | No model surface and no inference path |
| §23 desktop, mobile, kiosk, TV, XR, voice | Terminal only. The clause that does apply — "terminal: stable output" — is already §4.7 |
| §24 extensibility and plugins, §26 unit economics | A single binary, one distribution channel, no plugin host |
| §30–§32 production preparation, templates, composition examples | Nothing to prepare beyond the gates this repository already runs |
| LCP / INP / CLS, bundle budgets | Web-only, and the playbook itself says not to apply them to a CLI |

**Where the adoption is recorded — decided 2026-09-27.** The playbook is
**referenced by its absolute path**, and the applicable subset is recorded in this
subsection. It is **not** vendored into this repository, and **not** written into
`AGENTS.md`.

The reason is the charter's own: `AGENTS.md` is the single normative source, and a
charter that names a policy it does not own becomes a second constitution — the
failure §4.1 exists to prevent. A reference that breaks when the file moves is the
cheaper defect, because it is repaired by editing two lines in one place. If
`/Users/klosraf/Prototipos/MASTER_APPLICATION_DEVELOPMENT_PLAYBOOK.md` moves, the
provenance line above and this paragraph are exactly what has to change.

## 3. Diagnosis

Measured against the code, not remembered. The reproduction of each item is in §8;
what remains is shaped so a reviewer can decide it.

### Resolved in this pass

| ID | Finding | Consequence it had | Status |
|---|---|---|---|
| UX-01 | Installing the zsh script exactly as its own header documented — a file named `_kroot` on `fpath` — registered **nothing**: `compinit` never associated the file with the command, so TAB did nothing and the script's own registration was unreachable | The documented install path *is* the whole product for that feature. A completion that silently does not exist teaches the caller the tool is broken | resolved |
| UX-02 | The zsh script offered the command list in every position, including positions where a subcommand is not a valid operand, while bash already declined those positions | The assistant built for writing commands suggested commands the caller could not use, and one product behaved differently per shell | resolved |
| UX-03 | `kroot version foo` printed the version and exited `0`, while `help` and `man` already rejected a surplus operand | Undefined input was silently accepted, contradicting `api-compatibility.md`; a caller who typed a third word was told it worked | resolved |
| UX-04 | A command name resembling nothing produced a rejection with no next step; the recovery pointer appeared only when a near miss was found | The one failure a reader cannot act on was the one with nothing to try next | resolved |
| UX-05 | The generated manual documented no configuration variable, so a caller who reads `man kroot` could not learn that `KROOT_LOG_LEVEL` exists or what it accepts | The manual is the surface `api-compatibility.md` points at for stable names; an incomplete manual reads as authoritative while being wrong | resolved |
| UX-06 | `README.md` opened with an unfinished placeholder comment; the zsh recipe wrote to `${fpath[1]}` (frequently not writable, and silent about the file name being load-bearing); actionable examples carried a `$` prefix that breaks pasting | The first screen of a repository is part of the experience; an instruction that fails when followed is a defect, not a typo | resolved |
| UX-10 | `-h` and `--help` had no single rule: `kroot version -h` gave the **program's** help, `kroot version --help` printed the version until the surplus-operand fix, and `kroot help version` was the only spelling that answered | Three spellings meant "tell me about this command" and disagreed, so a reader reaching for the universal convention got another tool's help or an error | resolved (ADR-0004) |
| UX-14 | No stated startup-time budget and no committed benchmark | "Fast enough" was an opinion nobody could disagree with, while the repository's own rule requires a benchmark behind any performance claim | resolved |
| UX-11 | Completion offered command names and nothing else, so `kroot completion <TAB>` offered commands instead of shells, and `kroot help <TAB>` / `kroot man <TAB>` offered nothing where a command name *is* the valid answer | The assistant for writing commands was silent exactly where it had an answer, and loud where it did not | resolved |
| UX-12 | bash fell back to filename completion where fish and zsh declined, so the same product behaved three ways depending on the shell — visible only to whoever installed two | kroot takes no file operand, so a path offered after a rejected word is a suggestion the binary cannot accept; all three now decline | resolved |

### Root cause, for the class rather than the instance

Every zsh defect survived a suite that already validated generated scripts against
real shells, because the suite **sourced** the artefact while the product
**installs** it. An installed file is autoloaded as the function `compinit` calls; a
sourced file is not. The two run different code, and only the second was ever
exercised. The rule that follows is in §7: a generated artefact is proven through
the path the documentation tells the caller to use, not the path that is easiest to
script.

### Open, and needing an answer

| ID | Item | Shape of the change |
|---|---|---|
| UX-13 | No colour, no progress output, no pager, no TUI | **Deliberate.** ADR-0003 and `observability.md` both reject styled output until a consumer exists. The first interactive or long-running command is that consumer, and it brings `NO_COLOR` with it |

Every finding in this section is now resolved or deliberate. The list stays because
a new command or a new shell reopens it: a command that accepts a file operand is
the one change that would justify reconsidering a declined position.

### Deliberate, and therefore not defects

Recording these stops the next contributor from "fixing" them:

- **`kroot version -h` prints the program's help.** The flag set owns `-h`; a
  command's own help is reached through `kroot help <command>` until UX-10 decides
  otherwise.
- **Signals do not force an exit code.** The context is threaded from `main`, and
  `signal.NotifyContext` waits for the first command that can block, as ADR-0003
  states. A test that cancels a context is not a signal test.
- **A help write failure is tolerated on the `help` path.** The text already
  targets stdout and there is nowhere left to report the failure; the reasoning is
  in `runHelp`, and every other write failure is propagated.
- **The completion scripts bake in the command list.** Documented in every
  generated header, and the reason regeneration is part of the upgrade story.
- **Logs go to stderr, encoded by destination.** Only the encoding is unstabilised;
  the structured keys are the interface.

## 4. The experience standard

Every rule here is satisfied by an observation, not an intention. When a rule
cannot be checked, it does not belong here.

### 4.1 Non-negotiables

1. **One source of truth per fact.** Command lists come from the registry; exit
   codes from the constants the process returns; accepted configuration values from
   the slice the parser validates against; documentation from the code that defines
   it. New duplication is a defect in the making.
2. **Nothing is silently accepted or silently dropped.** A surplus operand, an
   unknown shell, an unaccepted log level and a truncated write all end in a
   diagnostic and a non-zero exit.
3. **Every failure names the next step** — either the fix itself
   (`did you mean "bash"?`) or where to find it (`see "kroot help"`).
4. **Determinism is a feature.** Same input, same bytes: sorted lists, alignment
   computed from the data. A generated file that reorders itself teaches reviewers
   to ignore its diffs.
5. **Requested output on stdout, everything else on stderr**, including on the
   failure path.
6. **Text does not over-claim.** No wording that promises more than the code keeps,
   and no heading over nothing.
7. **Accessibility is a constraint, not a phase.** Nothing may depend on colour,
   cursor position, animation, a wide terminal, or a TTY that may not be there.

### 4.2 Voice

- **Sentence case**; one-line summaries carry no trailing period, because a summary
  is not a sentence.
- **Imperative for actions** ("print the version and exit"), **declarative for
  facts**.
- **Errors say three things in one message**: what was wrong, what would have been
  accepted, what to do next.
  - `usage error: unknown command: "versioo"; did you mean "version"?`
  - `usage error: completion: unknown shell "tcsh": want one of bash|fish|zsh`
- **No blame, no vagueness.** "Invalid input" without the accepted vocabulary is an
  unfinished message.
- **One word per concept, everywhere**: command, flag, operand, shell, page,
  caller, diagnostic. A synonym is a second concept to a reader.
- **Honesty about scope.** No "server", "sync" or "dashboard" in text or metadata
  until the thing exists.

### 4.3 Typography and rhythm

The terminal equivalent of a type scale: what is long, what is short, what lines
up, and what stays flat on purpose.

| Element | Rule | Why |
|---|---|---|
| Command list | Two leading spaces; the name column padded to the longest name plus two, computed from the data | A hard-coded width misaligns the moment a command is added |
| Section headings | `Usage:`, `Commands:`, `Flags:` — a word and a colon, then a blank line | Consistent shape is what makes help skimmable |
| Body text | One idea per line, wrapped for 80 columns | A line the author never chose is a line nobody can review |
| Man page source | Caps for section names, `.TP` for terms, at most 80 source columns | `mandoc -T lint` warns past that, and it is the linter of the ecosystem the output belongs to |
| Log records | Structured keys, `lower_snake_case`, values as data | Keys are the stable interface for operators; messages are not |
| Emphasis | None: no bold, no underline, no colour codes | Nothing may depend on a capability the destination might lack |

### 4.4 Layout and width

- **80 columns is the design width** — what a man page, a README diff and a split
  terminal all tolerate.
- **Never reflow authored prose.** roff fills paragraphs at render time; re-wrapping
  here fights the author.
- **One blank line between blocks, never two.**
- **The terminal is not the only destination.** Every emitted document is read as a
  diff, a redirected file, or a formatter's input: treat the bytes as the artefact,
  not the screen.

### 4.5 States

Every surface declares its states. In a CLI the names change; the obligation does
not.

| Interface state | Terminal equivalent | Required behaviour |
|---|---|---|
| First run | No arguments | Print help: a bare command is a request for orientation, not an error |
| Empty | Nothing to list | Omit the section rather than printing a heading over nothing |
| Loading | A slow or blocking operation | Not applicable yet; when it arrives, progress goes to stderr and only when stderr is a terminal |
| Error | A rejected invocation | Diagnostic on stderr, non-zero exit, next step named |
| Partial | A truncated write | Report failure; never claim success for half an artefact |
| Interrupted | A cancelled context | Fail fast; signal wiring arrives with the first blocking command |
| Degraded | A validation tool is absent | Skip visibly on a developer machine, fail loudly in CI (`KROOT_REQUIRE_SHELL_TOOLS`); never pass silently |
| Success | The requested output | On stdout, complete, and nothing else |

### 4.6 Feedback and microinteractions

The small moments where a product feels attentive or careless. Here they are named,
bounded and pinned by tests.

| Moment | Mechanism | Rule |
|---|---|---|
| Completing a typed word | TAB in bash, zsh, fish | Must work through the *documented install path*, not a plausible variant of it. Names are offered only where a name is valid |
| Making a typo | Bounded edit distance | At most three suggestions, ordered by distance then name; a short vocabulary gets a tighter bound, because at two edits every shell name resembles every other |
| Mistyping with no near match | Pointer to the command list | One hint, never two; a suggestion suppresses the pointer |
| Forgetting an argument | The accepted vocabulary | `want one of bash\|fish\|zsh`, never "invalid argument" alone |
| Upgrading the tool | A regeneration notice in every generated file | The caller learns the artefact is stale from the artefact |
| Redirecting or piping | Silence | No progress, no colour, no pager, no cursor movement |
| Waiting on a slow operation | stderr, gated on a terminal | Not applicable yet; specified now so the first blocking command cannot invent its own answer |

Bounded, deterministic, phrased once, tested: that is the whole standard here.

### 4.7 Accessibility

- **No dependence on colour.** None is emitted, so `NO_COLOR` has nothing to
  disable (`observability.md` records why). Styled output arrives with the consumer
  that needs it, and brings `NO_COLOR` with it.
- **Screen-reader and pipe safety.** No ANSI escapes, no cursor control, no
  spinners. A test asserts the emitted bytes of every surface contain no escape
  character.
- **A TTY is never assumed.** Capability is read from the destination
  (`isTerminal`, tolerant of an unstattable writer) and exercised through injected
  writers, because the test process may have no terminal at all.
- **Width and locale independence.** 80 columns; ASCII output; measurements that
  matter are counted in runes, so a non-ASCII typo is measured the way a reader
  sees it.
- **Keyboard only, by construction.** There is no pointer to require.
- **Scripts are first-class users.** Exit codes and stream discipline are the
  accessible interface for automation, and they are the stable surface.

### 4.8 Performance

LCP, INP and CLS are **NO APLICA** here: the playbook that introduces them (§18)
forbids applying web metrics to a CLI, and they survive only inside the gated web
specification in §6.8. What a CLI is measured on is its own profile — **time,
memory, output size and cancellation** — and that is what the record below
measures.

- **Budgets are stated before they are claimed.** A startup-time figure recorded by
  a committed benchmark and reproduced by `make bench` is the minimum
  (`testing-strategy.md` requires a benchmark behind any claim).
- **No work at startup that a command does not need:** flag handling and
  configuration parsing, nothing else. No network, no persistence.
- **Output is streamed, not assembled,** so a failed write is reported at the stage
  that failed; buffers are preallocated where the size is known (`prealloc` keeps
  the habit honest).
- **Small enough to diff.** A surface too large to review is a surface nobody
  reviews.

**The record (CLI-09).** Measured 2026-09-27 on an Intel Xeon W-3223 @ 3.50 GHz,
macOS 26.7.

| Measurement | Value | Budget |
|---|---|---|
| `BenchmarkRunVersion` (in-process: flags, config, dispatch, write) | **1524–1559 ns/op**, 2450 B/op, 28 allocs/op | **≤ 10 µs/op** and ≤ 8 KiB/op |
| Process-level `kroot version`, fork/exec included | **≈ 14.3 ms** per invocation (50 runs, wall clock) | reference figure, not CI-enforced |

The budget is the durable half; the measured value is a reading, and it moves with
the machine and the load — which is why it is recorded with the machine it came
from instead of being written as if it were a constant. The in-process budget
leaves roughly six times the measured headroom on purpose: it exists to catch a
*category* of regression — an accidental file read, a reflection call, an
allocation storm at startup — not to police microseconds, which would produce a
flaky gate and a number nobody trusts. The process-level figure is recorded but
not gated, because `make bench` runs Go benchmarks and a spawn harness would be a
different, noisier tool; it is the honest answer to "what does a script pay", and
it is dominated by the operating system rather than by this code.


### 4.9 Compatibility

Every experience change is classified before it is made, against
`api-compatibility.md` §2.

| Change | Classification |
|---|---|
| New command or optional flag | MINOR; manual, README and completion regeneration travel with it |
| Help or man wording, layout, section order | Not stable — patch-level, still reviewed for voice |
| Exit code or stream semantics | Stable — a decision with a migration path and an ADR |
| `KROOT_*` name or accepted values | Name is stable; a new variable with a safe default is MINOR |
| Generated artefact contents | Not themselves stable, but the install path and file name they document are part of the instructions |
| Log record keys | Stable; message text is not |
| Styled output, progress, paging | A new capability, therefore a new decision with its own ADR |

**Pre-1.0 is not a licence.** While `MAJOR` is `0` the surface may still change, but
a documented behaviour that changes silently is the failure this document exists to
prevent — and the exit-code table already predates `v1.0.0` while being depended
upon in practice.

### 4.10 Mapping: interface concern → terminal equivalent

The same standard, stated twice, so a frontend cannot quietly adopt a different one.

| Concern | In `web/` | Here, today |
|---|---|---|
| Typography | Type scale, weights, line height | §4.3; 80-column source |
| Spacing | Spacing scale, rhythm | §4.4: blank-line discipline, computed padding |
| Layout | Grid, containers, breakpoints | Width budget, block order, stream separation |
| Navigation | Menus, breadcrumbs, deep links | Registry, `help`, `man`, completion |
| States | Loading, empty, error, optimistic | §4.5, from exit codes to omitted sections |
| Feedback | Toasts, inline errors, undo | stderr diagnostics, suggestions, exit codes |
| Microinteractions | Hover, focus, transitions | Completion, `did you mean`, regeneration notices |
| Accessibility | WCAG 2.2 AA | §4.7: no colour dependence, no TTY assumption, scripts as users |
| Performance | LCP, INP, CLS, bundle budget | §4.8: startup time, streaming output, benchmark |
| Responsive | Breakpoints, touch targets | Narrow terminals, pipes, redirects |
| Theming | Design tokens | None, deliberately |

## 5. CLI workstream

One item is one pull request, and an item is finished when its acceptance criteria
are demonstrated with an artefact. Each is marked by the state in which §3 left it.

### 5.1 Resolved items and what they now guarantee

| Item | Guarantee that now holds | Pinned by |
|---|---|---|
| CLI-01 — zsh completion exists where it is installed | The generated file begins with `#compdef <binary>`; installed as `_<binary>` on `fpath`, `compinit` resolves the binary to it; the menu lists every registered command with its summary at the first operand and offers nothing at the second; sourcing still registers; `zsh -n` and `bash -n` are clean; `README.md` names a writable directory and says the file name is load-bearing | `TestZshScriptDeclaresItsCompdef`, `TestZshInstalledCompletionWorksViaCompinit`, `TestZshCompletionRoundTripsAHostileSummary`, `TestGeneratedScriptsParse` |
| CLI-02 — no surplus operand anywhere | `kroot version extra` exits `2` with **0 bytes on stdout** and a diagnostic naming `kroot help version`; `kroot help a b` and `kroot man a b` keep their existing messages | `TestRun` |
| CLI-03 — every rejection names a next step | A name with nothing close carries `see "kroot help" for the command list`; a near miss carries the suggestion and **no second hint**; both still match `ErrUsage` and `ErrUnknownCommand`; `kroot man <typo>` answers identically | `TestUnknownCommandNamesTheCommandListWhenNothingIsClose`, `TestErrorKeepsTheChainAndAddsTheGuess`, `TestWriteManualRejectsAnUnknownCommand` |
| CLI-04 — the manual documents the environment | The program page carries an `ENVIRONMENT` section naming each variable, its accepted values in the parser's order and its default — from the parser's own slice; command pages carry none; a page with nothing to say omits the heading; `mandoc -T lint` accepts it; source stays within 80 columns | `TestProgramPageDocumentsTheEnvironment`, `TestEnvironmentDefinitionOmitsClausesItHasNothingToSay`, `TestEnvironmentSectionIsUnderstoodByARealRoffFormatter`, `TestManualOmitsSectionsItHasNothingFor` |
| CLI-05 — the first screen works | A real description instead of a placeholder; every actionable block copy-pasteable, with `$` reserved for transcripts of output; the zsh recipe creates a directory the caller owns and puts it on `fpath` **before** `compinit` | The transcript in §8 |
| CLI-06 — command answers `-h` and `--help` | Routed through `runHelp`; prints command usage and exits `0`; `-h` remains global on program level; extra operands ignored if help is first operand, but data if after operand | `TestRun` (7 dedicated subtests), ADR-0004 |
| CLI-09 — performance budget measured and stated | `BenchmarkRunVersion` pins startup/dispatch in-process (≤ 10 µs budget; measures ~1.55 µs, 2.4 KB, 28 allocs); process-level wall-clock documented (~14.3 ms) | `BenchmarkRunVersion` in `main_test.go`, §4.8 |
| CLI-10 — escape sequences & width enforced | Automated tests sweep every surface (13 targets): zero ANSI escape bytes (`0x1b`); human-facing prose (help, man roff) strictly ≤ 80 chars per line | `TestNoSurfaceEmitsAnEscapeSequence`, `TestHumanFacingSurfacesStayWithinTheDesignWidth` |
| CLI-07 — completion offers what each position accepts | `kroot <TAB>` offers commands; `completion <TAB>` the shells; `help`/`man` `<TAB>` the command names; a command with no operand and every later position offer nothing. The vocabulary is declared on the command, so a shell added to `Shells` reaches all three scripts | `TestGeneratedBashScriptActuallyCompletes`, `TestZshInstalledCompletionWorksViaCompinit`, `TestFishScriptCarriesTheOperandVocabulary`, `TestAnAddedShellReachesEveryScriptWithoutTouchingATemplate`, `TestNewRejectsContradictoryOperands` |
| CLI-08 — the shells agree on a declined position | bash is registered without `-o default`, fish keeps `-f`, zsh returns no match, so no shell offers a path after a rejected word — kroot has no file operand to accept one. The reasoning travels in the generated script and `README.md` | `TestNoShellFallsBackToFilenamesAfterADeclinedPosition` |

### 5.2 Open items

None. Every item in §3 is resolved or deliberate, and every item in §5.1 is verified
in §8. The workstream stays open for the changes that would reopen it: a command
that accepts a file operand, a second operand position, or a shell whose vocabulary
cannot be expressed as a list of words.

### 5.3 Sequencing

| Order | Items | Reason |
|---|---|---|
| 1 | CLI-01 … CLI-10 | All resolved and verified in §8 |

## 6. Web workstream — **specification only, gated**

**This section authorises nothing.** `web/` does not exist and arrives only with its
own ADR and milestone (`AGENTS.md` §0–§1; ADR-0003 §3). What follows is the
specification that ADR inherits, written now so the day it is approved the work
starts from a standard rather than from taste — and so a second interface cannot
adopt principles that contradict §4. The stack is already frozen: TypeScript
`strict` (with `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes`), React
18, Vite, Tailwind, pnpm, Biome + `tsc --noEmit` (`coding-standards.md`).

### 6.0 What must exist before the first component

| Prerequisite | Why |
|---|---|
| An ADR deciding a frontend exists and what it is for | Without a consumer this is speculative surface, which the charter rejects |
| A named journey the CLI cannot serve | The interface earns its place by doing what the terminal cannot: visual density, history, comparison |
| The HTTP surface it needs, decided under its own ADR | A UI without a contract is a mock |
| An authentication decision, if the journey needs identity | Retrofitting auth is a rewrite, not a feature |
| A build story: what deploys, where, how it rolls back | "Production-ready" is a deployment property |

### 6.1 Design tokens: the only place values exist

A hex code, a pixel or a duration inside a component is a defect: it cannot be
themed, audited or changed centrally.

| Token group | Contents | Rule |
|---|---|---|
| Colour | Semantic roles only: `surface`, `surface-raised`, `border`, `text`, `text-muted`, `accent`, `success`, `warning`, `danger`, `focus` | No `blue-500` in a component — roles survive a theme change, names do not |
| Typography | One scale of six to eight steps with line heights; one interface family, one monospace for identifiers and numbers | Numbers are tabular where they are compared |
| Spacing | A single scale, 4 px base, geometric steps | Every margin and padding is a step, never arbitrary |
| Radii | At most three | Consistency is visible; variety is not |
| Elevation | Three levels, shadow plus surface role | Dark themes carry elevation through surface, not shadow alone |
| Motion | Two durations, one easing | Nothing longer than `base`, ever |
| Focus | One ring definition, used everywhere | Replaced never, removed never |

Light and dark both exist from the first commit or neither does: a theme
retrofitted later is a rewrite of every hard-coded colour. The default follows the
operating system and is remembered per user.

### 6.2 Layout, spacing, density

- **An application shell, not a page.** A persistent frame — navigation,
  workspace, contextual panel — whose primary work surface never moves between
  routes; movement is what makes an application feel assembled from parts.
- **A grid with named regions** that stay named as the viewport narrows: the same
  shell collapses rather than becoming a different application.
- **Spacing is rhythm.** Consistent steps read as care; irregular gaps read as a
  prototype.
- **Density is a setting.** Comfortable and compact, both expressed through the
  spacing scale — a terminal-first audience often prefers dense views.
- **Empty, loading and error are states of the layout**, each with a defined region
  and height, so nothing jumps.

### 6.3 Navigation and wayfinding

- **Three questions answered at all times:** where am I, what can I do here, how do I
  get back. A screen that answers two is unfinished.
- **Deep links for everything shareable**, including filters, sort order and
  selection where they matter; the URL restores the view.
- **Keyboard first.** A discoverable key map covering navigation, the primary action
  of each view, search and dismissal; focus order follows reading order, is never
  trapped outside a modal, and returns to the trigger when one closes.
- **Browser affordances respected:** back and forward work, refresh is safe, and a
  form warns before discarding unsaved work.
- **Search is a first-class route**, not a filter buried in a table.

### 6.4 Components and their states

Every component ships with **all** of its states before it is used twice. A
component with only a default state gets patched differently on each screen it
appears in — which is exactly how a product stops looking like one product.

| State | Requirement |
|---|---|
| Default, hover, active | Distinguishable at a glance, never by colour alone |
| Focus-visible | The single focus ring, never suppressed |
| Disabled | Says *why* on hover or focus; a better pattern is preferred where one exists |
| Loading | Preserves layout: a skeleton with the final dimensions, never a spinner that shifts content |
| Empty | Names what would be here and offers the action that creates it — never "No data" |
| Error | Says what failed, what to do, whether retry is safe |
| Partial / stale | Says when the data is old and how to refresh |
| Read-only | Looks interactive only when it is |
| Selected | Distinguished from focus and from hover, and announced, not only coloured |
| Success | Confirms what happened, and where the result went |
| No permission | States the missing capability and who could grant it; never a dead control with no reason |
| Disconnected | Says the session is stale and offers reconnection; actions that cannot succeed are refused with a reason, not queued silently |

Rules that outlive any component:

- **No dead ends:** every empty and error state offers the next action.
- **Forms:** visible label, help text where the format is non-obvious, validation on
  blur and on submit (never while typing a first character), the error beside its
  field *and* in a summary for screen readers, and a primary action disabled only
  when the reason is obvious.
- **Tables:** sort, filter and pagination have defined, URL-backed state;
  virtualization arrives with a measured threshold, not by habit.
- **Destructive actions** confirm with the object and its consequence named, and
  offer undo where the operation can be reversed.
- **Notifications** are informative, dismissible, never modal, and never the only
  carrier of an error.
- **Numbers** are tabular, consistently precise, explicitly unit'd, locale-aware.

### 6.5 Feedback and microinteractions

The difference between "works" and "feels solid" is almost entirely here.

- **Every action acknowledges within 100 ms.** Anything slower says what is
  happening rather than going silent.
- **Motion has a purpose:** orientation (where did this come from), continuity (same
  object), feedback (this worked). Without one of the three, remove it.
- **Durations come from tokens** — `fast` for state changes, `base` for enters and
  exits — and are never chained into a sequence the user must wait for. The bands
  the playbook proposes as a starting point, adjustable by testing: control
  80–140 ms, local change 120–200 ms, panel 160–240 ms, view 180–300 ms. They are
  an initial proposal, not a standard, and an action is never delayed to finish an
  effect.
- **`prefers-reduced-motion` is honoured everywhere**; nothing depends on an
  animation completing.
- **Optimistic updates only where rollback is honest** (a toggle, a label, a local
  reorder). Anything that can fail for a reason the user must see waits for the
  server.
- **Errors are recoverable, not just reported:** retry where retry is safe, a
  copyable identifier where a human has to investigate.
- **The interface never lies about state.** A pressed button reported success; a
  stopped spinner means finished, failed or cancelled — and says which.

### 6.6 Accessibility — WCAG 2.2 AA, enforced

A release gate, not a backlog item.

| Requirement | Criterion |
|---|---|
| Contrast | 4.5:1 body text, 3:1 large text and interface boundaries |
| Keyboard | Every control reachable and operable, no trap, focus visible at all times |
| Target size | ≥ 24 × 24 px, ≥ 8 px apart on touch |
| Semantics | Native elements first; landmarks, headings in order, lists as lists |
| Names | Every control has an accessible name; an icon is never the only label |
| Live regions | Results announced politely, errors assertively, never twice |
| Not colour alone | State carried by text, shape or position as well |
| Zoom and reflow | Usable at 200% zoom and 320 px, without horizontal scrolling |
| Motion | `prefers-reduced-motion` respected; no flashing content |
| Forms | Errors programmatically associated with their fields and summarised |
| Dialogs | Focus moves in on open, is trapped inside, returns to the trigger on close, `Escape` dismisses |
| Time | No time-limited interaction without an extension mechanism |

Automated checks (axe or equivalent) run in CI and block on violations — they
catch roughly a third of real issues, so every interactive component also carries a
manual keyboard pass recorded in its pull request.

### 6.7 Responsive design

Mobile-first, with breakpoints chosen for the interface rather than for devices.

| Width | Expectation |
|---|---|
| 320–639 px | Single column, shell collapses to a drawer, primary action thumb-reachable, no horizontal scroll |
| 640–1023 px | Two regions: navigation and workspace; the contextual panel becomes a sheet |
| 1024–1439 px | Full shell: navigation, workspace, contextual panel |
| 1440 px and above | The workspace stops growing; content keeps a readable measure and the surplus width becomes margin, not longer lines |

Touch and pointer are different inputs: hover-only affordances have a tap
equivalent, drag has a keyboard equivalent, and nothing depends on a cursor. Safe
areas are respected on devices with insets.

### 6.8 Performance budgets

Budgets are enforced in CI, not hoped for. A change that exceeds one either
justifies the new number in its pull request or is reverted.

| Metric | Budget |
|---|---|
| LCP (p75, mobile profile) | < 2.5 s |
| INP (p75) | < 200 ms |
| CLS (p75) | < 0.1 |
| Initial JS, compressed | < 200 KB, route-split beyond it |
| Route change | No full-page reload; data fetching never blocks first paint |
| Images | Modern formats, explicit dimensions, lazy below the fold |
| Fonts | Self-hosted, subset, `font-display: swap`, no invisible text |

Memoize only what profiling shows is expensive — `React.memo` is a measured
decision, not a habit — and never ship a dependency the platform already provides.

### 6.9 Compatibility and locale

- **Browsers:** a stated `browserslist`; the earliest supported browser gets a
  graceful degradation, never a blank page.
- **Desktop shell:** if a Tauri build ships, its webview is a supported target and
  is tested exactly as a browser is; capability checks replace user-agent checks for
  clipboard, notifications, filesystem and window APIs.
- **Locale:** if the product is bilingual, both locales ship in the same change,
  keys are equal in both, no visible string is concatenated from fragments, dates
  and numbers go through `Intl`, and the layout tolerates the longer language
  without truncation. That is a product decision; this document only fixes the rule
  once it is taken.
- **Offline and failure:** every network call has a timeout, a defined error state
  and a retry policy; a dropped connection never leaves the interface in a state the
  user cannot leave.

### 6.10 Definition of done for this workstream

1. Every state in §6.4 exists, including empty, loading, error and partial.
2. Keyboard and screen-reader passes recorded by hand; axe reports no violations.
3. Tokens used exclusively — no raw colour, spacing, radius or duration.
4. Both themes and all four breakpoints seen, at 200% zoom.
5. §6.8 budgets hold on the production build.
6. Tests: Vitest for logic and hooks, one Playwright journey, and no snapshot
   asserting behaviour.
7. Documentation and `CHANGELOG.md` updated in the same pull request.
8. The journeys named in the ADR that authorised `web/` all work.

## 7. Verification and gates

### 7.1 The floor

Nothing here replaces `make ci`, and CI is authoritative.

| Gate | Command | What it protects |
|---|---|---|
| Formatting | `make fmt-check` | The formatter set the lint gate enforces |
| Static analysis | `make vet` | Type-level mistakes the compiler accepts |
| Lint | `make lint` | Unsafe patterns, unused code, unexplained suppressions |
| Tests, race, coverage | `make test-race` | Behaviour, concurrency, and coverage that does not decrease |
| Vulnerability scan | `make vuln` | Called code with known CVEs |
| Build | `make build` | That the thing still compiles and links |

### 7.2 What counts as evidence

An experience change is justified by an observation, recorded in the pull request.
The states are closed, so a claim cannot be softer than what happened:

| State | Meaning here |
|---|---|
| **Verificado** | A command was run, its output seen, and it supports the claim |
| **Verificado por comparación** | The behaviour was observed before and after the change — two builds, one probe — because the interface a test would need did not exist in the tree the test would have to run against. The transcript is the evidence |
| **Fallido** | It was run and it did not support the claim |
| **No ejecutado** | It was available but not run — including a test that skipped. Never a pass |
| **Bloqueado** | It could not run: a tool, a permission or an environment was missing |
| **No aplica** | It does not apply to this change, with the reason written down |

A visual or manual check that nobody performed is **No ejecutado**, not *Verificado*.

| Claim | Evidence |
|---|---|
| "This is fixed" | The failing case before, the passing case after, and the test that pins it |
| "It behaves the same in every shell" | A driver run in each shell that can run, plus a structural assertion for the one that cannot |
| "This message helps" | The exact text, for the invocation that produces it |
| "This is safe when redirected" | stdout to a file, stderr shown, exit code echoed |
| "This is faster" | Before and after numbers from `make bench` |
| "Docs are updated" | The diff of the document, in the same pull request |

**A skip is not a pass.** The suite validates generated artefacts by handing them to
`bash`, `zsh` and `mandoc`; those tests skip when a tool is absent, which is right on
a developer machine and wrong in CI. `KROOT_REQUIRE_SHELL_TOOLS` turns the skip into
a failure, and a test compares the name in `shelltools_test.go` with the one in
`ci.yaml` so a rename cannot quietly restore the skip.

### 7.3 Which tests a change owes

| Change | Required proof |
|---|---|
| A new command | Registry entry with `Usage` and `Long`; a `TestRun` case; and the command reaching help, the manual and every completion script (the existing tables assert this) |
| A new flag | Rendering in help and in the manual, from the flag set rather than a second description |
| Changed error wording | A test asserting the *presence of the recovery*, not the whole sentence, so the wording stays editable |
| A change to exit codes or streams | A recorded decision and an ADR; then tests asserting both stdout emptiness and the code |
| A generated artefact | The tool that consumes it: `bash -n`, `zsh -n`, a driver that actually completes, `mandoc -T lint` and a parse-tree read-back |
| An installation instruction | A run through the documented path, not a variant of it — for zsh, installed on a temporary `fpath` and read by `compinit` |
| A configuration variable | Parsing, rejection of an unknown value, and the manual's vocabulary, all from one definition |
| New output text | The width and escape-byte guards (CLI-10), so a surface cannot grow past its budget unnoticed |
| A frontend component | Vitest for logic and hooks, axe in CI, one end-to-end journey, a manual keyboard pass |
| Documentation only | The link check, a transcript if it contains instructions, and a `CHANGELOG.md` entry when the wording is user-visible |
| A figure quoted in this document | Re-measured in the same change. A number written once and never revisited is a claim nobody can check — and coverage, benchmark and transcript figures all drift the moment the code they describe moves |
| A defect whose fix **adds** an interface | The test cannot run against the old tree — the API it exercises did not exist — so the proof is a before/after comparison: build the parent commit, drive both builds through the same probe, and paste both. `testing-strategy.md` asks for a test that fails before the fix; where that is impossible, the comparison is the honest substitute, and saying which one you did is part of the evidence |
| Adversarial input (playbook §19, CLI subset) | Empty, oversized, duplicated, invalid, Unicode and hostile input on every accepted position; the registry already sorts and bounds names, and the roff and shell writers escape what free text can carry |
| Lifecycle and distribution | A clean install of a generated artefact, and an upgrade that leaves it stale: the completion scripts' documented install is exactly the case CLI-01 covers |
| Interface | Long text held to 80 columns, zero escape bytes, and a reader that is neither a terminal nor a person — a pipe, a file, a screen reader |

### 7.4 Recipes

Copy-pasteable, because a check nobody can run is a check nobody runs.

```sh
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

# manual: lint, then read the parse tree rather than the rendered overstrike
kroot man > "$d/kroot.1" && mandoc -T lint "$d/kroot.1"
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
3. Does every new failure name the accepted vocabulary and a next step?
4. Is the output deterministic — sorted, computed, stable across runs?
5. Does stdout carry only the requested output, on the success *and* the failure
   path?
6. Is there a test that fails before the change and passes after?
7. Are generated artefacts validated by the tool that consumes them, through the
   path the documentation tells the caller to use?
8. Are the width and escape-byte budgets respected, or is the exception named and
   justified?
9. Do `README.md`, the generated manual and `CHANGELOG.md` move in the same pull
   request?
10. Is it one concern, under ~500 changed lines, with a Conventional Commit and a
    scope?
11. If a dependency was added: what it does, why the standard library is not enough,
    which alternatives were rejected?
12. Are the deliberate omissions still omissions — no heading over nothing, no
    promise the code does not keep?
13. Does the wording follow §4.2, including the fixed vocabulary for commands,
    flags, operands and shells?
14. Would a caller who only ever reads the terminal discover this without reading
    the diff?
15. Does anything depend on colour, cursor position, animation, a wide terminal, or
    a TTY that may not be there?

### 7.6 Anti-patterns, rejected on sight

- **Editing the wording until the test passes.** The test asserts the recovery a
  caller needs; the wording serves the test, not the other way round.
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
- **A silent skip.** If a required tool is missing in CI, the build fails; that is
  the point.

## 8. Resolved register — this pass

### 8.1 Gates, as they were actually run

| Gate | Command | Result |
|---|---|---|
| Formatting | `make fmt-check` | clean, no diff |
| Static analysis | `go vet ./...` | clean |
| Lint | `bin/golangci-lint run` | **0 issues** |
| Tests + race + coverage | `go test -race -covermode=atomic ./...` | **ok** — 97.6% of statements in `github.com/klosraf/kroot`, **100.0%** in `.../internal/cli`, the latter up from the 98.4% this pass started at |

The 100% is measured on the whole of this pass merged: the workstream below, the
width budget, and the write-failure walk. On the workstream branch alone it read
98.6%, and the remainder was real — the `ENVIRONMENT` section the manual gained
here renders only when the page documents a variable, so the write-failure walk,
whose fixture did not, never reached its two error returns. A stage walk is
bounded by its own fixture; that is now stated in the test rather than left to
be rediscovered.

What remains uncovered in the whole repository is three guards in package `main`:
`os.Exit` in `main()`, and the error returns after `newProgram` and `cli.New`.
None can be triggered without editing the program itself — the net under a change
that has not been written yet.
| Vulnerability scan | `bin/govulncheck ./...` | No vulnerabilities found |
| Build | `make build` | ok |
| **The whole definition of done** | **`make ci`** | **exit 0** |

Tools present for this run: `zsh`, `bash`, `mandoc`. `fish` is absent, so its
structural checks skipped visibly — the designed behaviour on a developer machine.

**Beyond this machine.** The same gates ran in CI for pull request
[#25](https://github.com/klosraf/kroot/pull/25), run `36323624010`: `commit
messages`, `lint`, `test (ubuntu-latest)` and `vulnerabilities` all passed. macOS is
absent from that list by design rather than by omission — ADR-0002 moved it to
pushes to `main`, so the platform is verified on the merge that produces the
releasable ref. A reader can therefore check this register instead of taking it on
trust. The one claim here CI cannot confirm is the driving of real shells, which
happened on the machine that produced the binaries; §7.2 names that as *Verificado
by comparison* rather than by a test.

### 8.2 Behaviour, observed

Exit codes come from the built binary, not from `go run`, which substitutes its own.
stderr appears as JSON because the destination was a file; on a terminal the same
record is human-readable text, and the message is identical either way.

| Invocation | Observation |
|---|---|
| `kroot version extra` | `exit=2`, **stdout 0 bytes**, `usage error: version takes no arguments, got 1: see "kroot help version"` |
| `kroot kubernetes` | `exit=2`, stdout 0 bytes, `usage error: unknown command: "kubernetes"; see "kroot help" for the command list` |
| `kroot versioo` | `exit=2`, `usage error: unknown command: "versioo"; did you mean "version"?` — suggestion only, no second hint |
| `kroot help a b` | `exit=2`, `usage error: help takes at most one command name, got 2` — unchanged |
| `kroot man a b` | `exit=2`, `usage error: man takes at most one command name, got 2` — unchanged |
| `kroot version` | `exit=0`, version line unchanged |
| `kroot completion zsh` into a temporary `fpath`, then `compinit` | first line `#compdef kroot`; `registered=_kroot`; `zsh -n` clean. Before the fix: registration empty, TAB dead |
| `kroot completion bash` driven in a real bash | `kroot <TAB>` → `completion help man version` · `completion <TAB>` → `bash fish zsh` · `help <TAB>` → the command names · `man comp<TAB>` → `completion` · `version <TAB>` → nothing · `completion bash <TAB>` → nothing |
| the same script, installed on a temporary `fpath` and read by `compinit` | `registered=_kroot`, and the same six answers, with descriptions on the command list; `help` and `man` share one branch, because they share one vocabulary |
| a position all three shells decline, `kroot version <TAB>` | nothing anywhere. bash is registered without `-o default`, fish keeps `-f`, zsh returns no match: no shell offers a path, because kroot has no file operand to accept one |
| `kroot man > kroot.1 && mandoc -T lint kroot.1` | clean; the program page carries `.SH "ENVIRONMENT"` → `KROOT_LOG_LEVEL` → `One of: debug, info, warn, error. Unset means "info".` |
| `kroot man version` | `mandoc -T lint` clean; **0** occurrences of `ENVIRONMENT` — a variable belongs to the program, not to a command |

New and updated tests, all passing in the run above:
`TestZshScriptDeclaresItsCompdef`, `TestZshInstalledCompletionWorksViaCompinit`,
`TestZshCompletionRoundTripsAHostileSummary` (now states `CURRENT`, as zsh does),
`TestUnknownCommandNamesTheCommandListWhenNothingIsClose`,
`TestProgramPageDocumentsTheEnvironment`,
`TestEnvironmentDefinitionOmitsClausesItHasNothingToSay`,
`TestEnvironmentSectionIsUnderstoodByARealRoffFormatter`, and `ENVIRONMENT` added
to `TestManualOmitsSectionsItHasNothingFor`.

### 8.3 What changed in this pass

| File | Change | Finding |
|---|---|---|
| `internal/cli/completion.go` | zsh template: `#compdef` on the first line, a dual-mode dispatch keyed on `funcstack`, and a position guard before the menu | UX-01, UX-02 |
| `internal/cli/help.go` | `Program.UnknownCommand`, which appends the command-list pointer only when the registry had no near miss | UX-04 |
| `internal/cli/manual.go` | `EnvVar`, the `ENVIRONMENT` section on the program page only, `envDefinition`, and the unknown-command path routed through `Program.UnknownCommand` | UX-05, UX-04 |
| `main.go` | `rejectOperands` applied to `version`; both unknown-command call sites; `manualInfo` feeding the vocabulary from `allowedLogLevels` | UX-03, UX-04, UX-05 |
| `internal/cli/*_test.go` | Eight new or updated tests, including a real-zsh installation test that fails on the previous generator | the class in §3 |
| `internal/cli/cli.go` | `Command.Operands` with its registry validation, and the operand grouping the generators read | UX-11 |
| `internal/cli/completion.go` | The positional cases in the bash, zsh and fish templates, driven by the declared vocabulary rather than a list inside the template; and the bash registration drops `-o default`, so a declined position stays declined as it already did in the other two | UX-11, UX-12 |
| `main.go` | `completion`, `help` and `man` declare their first operand — `completion` from the `Shells` slice itself — and the entry carries it to the generator | UX-11 |
| `README.md` | A real description, copy-pasteable action blocks, a zsh recipe that names a writable directory and the load-bearing file name, and the manual's environment table | UX-06 |
| `CHANGELOG.md` | Entries for every user-visible change above | `AGENTS.md` §9 |
| `docs/prompts/` | This document: the whole experience structure, integrated | — |

## 9. Documentation map — one source of truth per fact

This file is the single structure for *experience* work. It is not a second home
for the repository's normative documents: it distills them into §2 and cites them
here, so that a rule always has exactly one owner.

| Document | Owns |
|---|---|
| [`AGENTS.md`](../../AGENTS.md) | The binding charter: scope, stack, naming, code standards, testing, security, observability, quality gates, delivery, documentation |
| [`README.md`](../../README.md) | The entry point: what the product is, getting started, commands, usage, manual, completion, configuration, layout, conventions |
| [`CONTRIBUTING.md`](../../CONTRIBUTING.md) | How to propose, review and land a change |
| [`SECURITY.md`](../../SECURITY.md) | Private vulnerability reporting |
| [`CHANGELOG.md`](../../CHANGELOG.md) | What changed, per release, in Keep a Changelog form |
| [`coding-standards.md`](../enterprise/coding-standards.md) | The specifics of Go and — when `web/` lands — TypeScript style |
| [`testing-strategy.md`](../enterprise/testing-strategy.md) | The test pyramid, the required gates, and the rules tests must obey |
| [`api-compatibility.md`](../enterprise/api-compatibility.md) | What callers may depend on, and what they may not |
| [`versioning-policy.md`](../enterprise/versioning-policy.md) | SemVer, and the build metadata a released binary reports |
| [`branching-strategy.md`](../enterprise/branching-strategy.md) | Branch names, commit types, pull-request titles |
| [`release-process.md`](../enterprise/release-process.md) | How releases are tagged and published |
| [`observability.md`](../enterprise/observability.md) | Logs, correlation, health endpoints, metrics, SLOs |
| [`security-policy.md`](../enterprise/security-policy.md) | The security program behind `SECURITY.md` |
| [`docs/adr/`](../adr/) | Decisions, append-only: `NNNN-title.md`, superseded only by pointing at a replacement |
| **This document** | The experience: diagnosis, standard, both workstreams, verification, and the register of what was resolved |
| *External, not in this repository:* `MASTER_APPLICATION_DEVELOPMENT_PLAYBOOK.md` | The universal product-development playbook. Its applicable subset is adopted and reasoned in §2.1, with the remainder declared NO APLICA. It does not override `AGENTS.md`, by its own §02 |

**Rules that keep this honest.**

1. This document **cites, never restates.** When a rule here and a source disagree,
   the source wins and this file's §2 line is corrected in the same change.
2. A rule that changes in a source must be mirrored here in the same pull request —
   `AGENTS.md` §9 makes that non-optional for anything touching the CLI.
3. Experience work has **one** entry point: this file. A finding, criterion or
   register row must not be duplicated into `docs/enterprise/`, because two copies of
   a fact are the defect §4.1 exists to prevent.

## 10. Definition of done

**Ready — the gate before writing any code**, following the playbook's §29. An item
that fails this is not started; it is clarified.

- [ ] The finding is reproduced against the code, and the current behaviour is
      understood rather than assumed.
- [ ] Scope and exclusions are written down, including what the change will *not*
      do.
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
9. Any decision the item required exists as a record before the code does.
10. Nothing in the change depends on colour, cursor position, animation, a wide
    terminal, or a TTY that may not be there.













