# Changelog

All notable changes to Kroot are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **A caller's mistake is no longer reported as the program's failure.** Every
  diagnostic was logged as `level=ERROR msg="kroot failed"`, so a mistyped command
  and an unreadable filesystem produced records differing only in the text of
  `err`. Three defects lived in that line: a usage error was attributed to the
  program, `KROOT_LOG_LEVEL=warn` could not silence a typo (so `error` described
  nearly every diagnostic the binary emits), and the recovery — the only part
  written for a person — sat buried in an `err=` attribute behind a timestamp
  nobody asked for. A usage error is now logged at `WARN` as `kroot: usage error`;
  a configuration or runtime failure stays at `ERROR`. The level follows the same
  split the exit code already used, so the two cannot disagree, and the record's
  keys are unchanged, so a redirected consumer still receives the same shape with
  the whole message. Decided in
  [`docs/adr/0005-severity-of-a-caller-caused-failure.md`](./docs/adr/0005-severity-of-a-caller-caused-failure.md).

### Added

- **The help screen now says where to go next.** `kroot help` ended at the flag
  list, so every command was discoverable but none was explorable: the hint that
  `help` and `man` take a command name — the affordance that makes the other
  commands reachable — was the one thing missing from the page. It closes with two
  lines naming `kroot help <command>` and `man kroot`, built from the program's own
  name and omitted rather than printed empty when those commands are not
  registered.

### Fixed

- **The `Flags:` block of `kroot help` was laid out by the standard library while
  the block above it was laid out by kroot.** `flag.PrintDefaults` separates a
  name from its usage with a literal tab and a four-space hanging indent, so one
  screen carried two typographic rules at once. The block is now rendered with
  kroot's own computed column — the same padding the command list uses — so no
  human-facing surface contains a tab, and the manual still renders each flag
  separately.
- **`kroot help a b` and `kroot man a b` named no next step.** Every other
  rejection already ended in something actionable — a near miss, the command list,
  the command's own help — but too many operands produced only a count, leaving
  the one failure on that path a caller could not act on. Both now name the correct
  form, and still exit `2` with nothing on stdout.

- **The zsh completion script did not exist where the product documents installing
  it.** The generated file carried no `#compdef` line, so `compinit` never associated
  it with the binary: installed as `_kroot` on `fpath` — exactly as the script's own
  header instructs — nothing was registered and TAB produced no candidates at all.
  The registration code at the end of the file was unreachable, because it only ran
  if something executed the file, which the documented install path never does. Two
  changes close it. The script now opens with `#compdef <binary>` and ends with a
  dispatch keyed on `funcstack`: an autoloaded file *is* the body of the function
  `compinit` calls, so it completes there, while a sourced file is not, so it
  registers there. The menu is also offered only in the first operand position,
  where a subcommand is valid — which is what bash already did, so one product no
  longer behaves differently per shell and the README's promise holds. The test that
  proves it installs the generated file on a temporary `fpath` and reads it back
  through a real `compinit`; every earlier test *sourced* the script instead, which
  is exactly why none of them could see this.
- **`kroot version foo` printed the version and exited `0`.** `help` and `man` had
  already been rejecting a surplus operand, and `api-compatibility.md` states that
  undefined input "will not be silently accepted" — `version` was simply the command
  nobody covered, so a caller who typed a third word was told it worked. It now
  exits `2` with nothing on stdout and a diagnostic naming `kroot help version`, so
  the answer to "then what does it accept?" is one invocation away.
- **The README's first screen neither described the product nor could be followed as
  written.** It opened with an unfinished placeholder comment; the zsh recipe wrote
  into `${fpath[1]}`, which is frequently not writable by the caller and said nothing
  about the file name being load-bearing, so the install failed or silently produced
  a dead completion; and the actionable examples carried a `$` prompt prefix, which
  breaks pasting them into a shell. The description now states what the product is,
  every actionable block is copy-pasteable with `$` reserved for transcripts of
  output, and the zsh recipe creates a directory the caller owns, writes `_kroot`, and
  puts that directory on `fpath` **before** `compinit` runs.
- The validation of kroot's own generated output did not run in CI.
  `internal/cli` proves the completion scripts and manual pages by handing them to
  the tools that will consume them — `bash -n`, `zsh -n`, and `mandoc -T lint`
  plus a parse-tree read-back — but each of those tests **skipped** when its tool
  was absent, and `mandoc` is not installed on `ubuntu-latest` by default. A skip
  is indistinguishable from a pass in a required status check, so the suite
  reported success while the roff and shell validation covered nothing on every
  pull request. Two changes close it. The `test` job installs `mandoc` on Linux,
  pinned to the version Ubuntu ships so a formatter change cannot silently flip a
  result. And `requireTool` now reads `KROOT_REQUIRE_SHELL_TOOLS`, which the
  workflow sets: a developer machine still skips, because `mandoc` is installed
  nowhere by default and failing would make `go test ./...` fail for a reason
  unrelated to the code, while CI fails loudly if a tool goes missing. Because the
  workflow and the test are a literal in a YAML file and a literal in a Go file
  with nothing connecting them, `TestRequireShellToolsEnvNameIsWired` reads the
  workflow and compares the names — this mismatch happened while writing the
  change, and without the test it would have restored the skip silently.

- `SECURITY.md` described a repository that is not this one, in three places. It
  stated the reporting channel was constrained because the repository is private
  — it is **public**, and GitHub's private vulnerability reporting is available
  here (the API answers `{"enabled": false}` rather than the 404 a private
  repository returns), though it is not enabled yet, so email remains the working
  channel. It told operators to "restrict network exposure ... unless a reverse
  proxy with TLS and authentication sits in front" and to "monitor logs for
  unexpected authentication" — kroot opens no sockets, speaks no protocol and has
  no authentication, so that was advice for a server the project does not have. And
  it claimed `govulncheck ./...` "gates every PR that touches Go code", which is
  the exact drift already recorded as fixed in `AGENTS.md` and
  `docs/enterprise/security-policy.md`; the `vulnerabilities` job has no path
  filter and runs on every pull request, and this file was simply missed when the
  other two were corrected. The document now states what the program is — no
  network surface, no authentication, no persistence yet, environment-only
  configuration — and gives operator guidance that applies to a local binary,
  including the behaviours a supervisor can actually depend on: an unrecognised
  `KROOT_LOG_LEVEL` is rejected rather than coerced, and the exit code
  distinguishes a usage error from a runtime failure.

### Added

- **Completion now offers what each position actually accepts.** The scripts
  completed command names and nothing else, so `kroot completion <TAB>` offered
  commands rather than shells, and `kroot help <TAB>` and `kroot man <TAB>` — where
  a command name *is* the valid answer — offered nothing at all. Every command now
  declares its first operand, and the three generators offer that vocabulary in the
  position it belongs to: shells after `completion`, command names after `help` and
  `man`, and nothing after a command that takes no operand or past the first operand.
  The declaration lives on the command rather than inside a template, so a fourth
  shell added to `Shells` reaches help, the manual and all three scripts from one
  place and a test fails the moment a template starts carrying its own copy of the
  list. Two details the shells forced: the zsh case assigns its array rather than
  listing words, because a bare word in a `case` branch is a command to zsh and
  fails with "command not found"; and the fish condition counts the tokens before
  the cursor, because `__fish_seen_subcommand_from` stays true at every later
  position and the vocabulary would otherwise keep being offered after the operand
  that already accepted one. Proved by driving a real bash and a real zsh through
  the documented install path; fish is not installed where this suite runs, so its
  condition is asserted structurally.
- **`kroot <command> -h` and `--help` print that command's own help.** Three
  spellings already meant "tell me about this command" and disagreed: `kroot help
  version` gave the command's help, `kroot version -h` gave the *program's* help
  because the global flag set owns `-h`, and `kroot version --help` printed the
  version until the surplus-operand fix turned it into an error. The alias now
  routes through the help command rather than a second renderer, so `kroot version
  --help` and `kroot help version` print the same bytes, tolerate a failed write the
  same way and exit alike. Only the first operand counts — `kroot version extra
  --help` stays a usage error, because after the first operand `-h` is data — and
  the program's own `-h` still comes from `flag.ErrHelp`, unchanged. Recorded as
  ADR-0004 rather than inherited from a flag package's default, since `api-compatibility.md`
  makes flags documented in the manual a stable surface from `v1.0.0`.
- **The generated manual documents the environment.** `kroot man` now carries an
  `ENVIRONMENT` section naming each `KROOT_*` variable, the values it accepts in the
  parser's order, and its default — with the vocabulary passed in from the very slice
  `parseLogLevel` validates against, so the page cannot advertise a level the binary
  rejects, and adding a level changes the page without touching the generator. The
  section belongs to the program page alone: a variable is read by the program, not
  by one command, and repeating it under every command is how a manual starts
  disagreeing with itself, so a command page carries none. A page with nothing to say
  omits the heading rather than printing an empty one, and the definition is asserted
  through `mandoc -T lint` and its parse tree, because it carries a quoted default —
  punctuation a `.TP` body rarely sees.

### Changed

- **The three shells no longer disagree about a declined position.** bash was
  registered with `-o default`, so a position the script declined fell back to
  filename completion, while fish (`-f`) and zsh (no match) offered nothing. Nobody
  who installed one shell could observe the difference, and nobody who installed
  two could remember which was which. kroot takes no file operand anywhere, so a
  path offered after a rejected word is a suggestion the binary cannot accept; all
  three now decline, bash without `-o default`. The reasoning is in the generated
  script's own comment and in `README.md`, and a test holds the registration lines
  of all three shells to it.
- **A command name that resembles nothing now names where the list is.**
  `kroot kubernetes` answered with the facts and no next step, while `kroot versioo`
  was already answered with the command it meant. The suggestion *is* the recovery, so
  it suppresses the pointer: a name with no near miss now ends with
  `see "kroot help" for the command list`, and the near-miss message is unchanged. The
  error chain still wraps both sentinels, so `errors.Is` callers are unaffected, and
  the manual's own rejection routes through the same code, so `kroot man <typo>` is
  answered exactly as `kroot <typo>` is.

## [v0.1.0] - 2026-09-27

The first release. Everything below shipped together, because the terminal core
was built as one piece: a command registry, the help and completion and manual
pages rendered from it, a documented exit-code contract, and the first
configuration variable. There is no HTTP API, no persistence and no frontend; the
project type is a terminal-first CLI application, decided in
`docs/adr/0003-cli-first-terminal-application.md`.

Nothing here is a compatibility guarantee yet. The guarantees in
[`docs/enterprise/api-compatibility.md`](./docs/enterprise/api-compatibility.md)
take effect at `v1.0.0`, and while `MAJOR` is `0` the CLI surface may still change
in a MINOR release.

### Added

- `kroot man [command]` writes a manual page in roff source to standard output —
  the program page with no argument, or one command's page under the name
  `kroot-<command>`, so `kroot man completion | man -l -` works the way a reader
  expects. This closes a contract that was already asserted but never satisfied:
  `api-compatibility.md` ties the v1.0.0 stability of "subcommand names and flags
  documented in the manual" to a manual that did not exist, and `Command.Long` was
  documented as "the manual print" with no consumer. The page is generated from
  the same registry that backs `help` and `completion`, so a command cannot be
  reachable but undocumented. Each page records the version and build date it came
  from, and the exit-code table is supplied by `main` from the constants the
  process actually returns rather than restated in the generator, so the page
  cannot disagree with the binary about its exit contract. Authored text is
  escaped for roff — backslash, hyphen, a leading `.` or `'`, and tabs, which roff
  does not honour — and macro arguments are quoted, without escaping hyphens,
  because an escaped hyphen in a date is a date no formatter can parse. Tests hand
  every page to `mandoc -T lint` and read the parse tree back, which is what caught
  two defects in the first draft: footers truncated at the space in `kroot 1.2.3`,
  and an `.SH` injected by a command summary becoming a real heading. Additive, so
  a MINOR under the versioning policy.
- `internal/cli`: `Program.Flags` (a rendering function) is replaced by
  `Program.FlagSet`, holding the flag set itself. Help still renders through
  `PrintFlags`, and the manual can now walk the set to typeset each flag's name and
  description separately, which a pre-rendered help block cannot provide. The
  indirection it replaced was already redundant — `PrintFlags` renders only when
  called — so nothing about the two renderers changed. No behaviour change.
- `kroot completion <bash|fish|zsh>` writes a shell completion script to stdout.
  The script completes subcommand names in the first operand position and nothing
  else, because that is the only position where a subcommand is a valid answer;
  bash falls back to filename completion for the words it declines. The command
  list is generated from the registry, so the script cannot offer a command that
  does not exist or omit one that does, and it is sorted so regenerating it does
  not produce a noisy diff. The supported shells live in one exported list that
  also supplies the usage line and the rejection message, so those three cannot
  drift apart. Adding a shell is one entry plus one generator. An unsupported
  shell is a usage error naming the accepted list and proposing a near miss
  (`kroot completion bas` → `did you mean "bash"?`). The matching reuses the
  bounded edit distance an unknown command gets, but tightened to one edit:
  shell names are three or four letters, so at two edits `tcsh` sits within reach
  of `zsh`, `fish` and `bash` at once and a request for an unsupported shell would
  be answered with every supported one. Command summaries are escaped per shell
  before they reach a script, so free text cannot terminate the quoted word it
  sits in. This is additive — no existing behaviour changes — so it is a MINOR
  under the versioning policy.
- Command framework under `internal/cli`: `Registry`, typed `Command` and `Env`,
  subcommand routing, single-command help via `kroot help <cmd>`, aligned
  command listings, and closest-match suggestions (`did you mean ...?`) using
  bounded Levenshtein distance on runes.
- `KROOT_LOG_LEVEL` — the first `KROOT_*` configuration variable, with `info` as
  its default: `debug`, `info`, `warn`, `error`. An unknown value is rejected
  rather than coerced, reported as `configuration rejected` with the accepted
  vocabulary on stderr, and exits `1` (the wording and the code that
  `api-compatibility.md` ties to a rejected configuration). The log destination
  picks the encoding — human-readable text on a terminal, JSON when stderr is
  redirected or piped — with identical structured keys either way, and no color
  codes are emitted, so `NO_COLOR` has nothing to disable. Documented in
  `docs/enterprise/observability.md` and the README's Configuration section.
- Project type decided as a terminal-first CLI application in
  `docs/adr/0003-cli-first-terminal-application.md`, with `context.Context`
  threaded from `main` to `run` so the first command that can block inherits a
  cancellation path instead of inventing one. `realMain` extracted from `main`
  so the wiring — configuration rejection included — is tested without
  spawning a process.
- `kroot version` reports the Go toolchain that built the binary, so the compiler
  is visible beside the revision it produced:
  `kroot dev (commit none, built unknown, go1.23.12)`.
  `docs/enterprise/versioning-policy.md` has always documented that field and the
  binary never printed it, which made that document's example output something
  the program cannot produce. The value comes from the runtime rather than from
  `-ldflags`, so it cannot drift from what actually compiled the binary — the same
  class of drift that let the pinned vulnerability scanner be built by an
  undeclared Go version.
- Initial module `github.com/klosraf/kroot` (Go 1.23.12).
- CLI entry point with `help` and `version` commands, plus `--version`.
- Build metadata (`version`, `commit`, `buildTime`) injected at link time and
  reported by `kroot version`.
- Table-driven unit tests covering command dispatch, the version flag, unknown
  commands and write-error handling.
- `Makefile` with `build`, `install`, `run`, `test`, `test-race`, `cover`,
  `bench`, `vet`, `fmt`, `fmt-check`, `lint`, `vuln`, `tidy`, `clean`, `tools`
  and `ci` targets.
- Enterprise governance baseline: `AGENTS.md` charter, eight policies under
  `docs/enterprise/`, `docs/adr/0001-adopt-enterprise-charter.md`,
  `.editorconfig`, `.golangci.yaml`, commitlint configuration, `CONTRIBUTING.md`,
  `SECURITY.md`, `CHANGELOG.md`, PR template and issue forms.
- CI workflow (`.github/workflows/ci.yaml`) running formatting, vet, lint, race
  tests with coverage, vulnerability scanning, build and commit-message linting.
- Dev tooling pinned per repository — `golangci-lint` v2.14.0 and `govulncheck`
  v1.8.0 installed into `./bin` by `make tools` — so local and CI runs cannot
  drift apart.
- Repository hygiene: `LICENSE` (MIT), `.gitattributes` for line-ending
  normalisation, `.github/CODEOWNERS`, and `.github/dependabot.yml` for weekly
  dependency updates across Go modules, npm tooling and GitHub Actions.

### Changed

- CI runs `test (macos-latest)` only on a push to `main`, not on every pull
  request. macOS runners bill at a 10x rate multiplier, and the account was
  consuming roughly 12,400 billable-equivalent minutes per month against a 3,000
  per month allowance — at which point GitHub refused to start any job in the
  repository, reporting only "The job was not started because recent account
  payments have failed or your spending limit needs to be increased", which is
  indistinguishable from a code failure through the API. Cross-platform
  verification still happens, because merging *is* a push.
  `test (macos-latest)` is consequently no longer a required status check on
  `main`: a required check that a pull request never reports blocks that pull
  request permanently. See
  `docs/adr/0002-keep-macos-verification-off-pull-requests.md`.
- Module path corrected from `github.com/kroot/kroot` to
  `github.com/klosraf/kroot`, matching the GitHub account that owns the
  repository.
- `docs/enterprise/branching-strategy.md` now documents the branch protection
  actually applied to `main`, including why the required approval count is 0
  while the repository has a single maintainer.

### Fixed

- Global flags were not listed under the `Flags:` section in `kroot help`.
  Because `flag.FlagSet` was directed to `io.Discard` to suppress standard error
  chatter, `fs.PrintDefaults()` wrote to discard instead of the help output
  stream. `cli.PrintFlags` now redirects `fs` output to the target writer
  temporarily and restores the previous output via `defer`.
- The repository sent vulnerability reporters to a channel that does not exist.
  `SECURITY.md` had already been corrected to name the maintainer's email,
  because GitHub does not offer private vulnerability reporting for a private
  repository without GitHub Advanced Security (the API returns 404), but
  `docs/adr/0001-adopt-enterprise-charter.md` still said reports "arrive through
  GitHub private vulnerability reporting", and this file's `### Notes` section
  said the same — so this document contradicted itself, carrying the correction
  in `### Fixed` above and the disproven claim at the bottom. Someone following
  either would have looked for a button GitHub does not offer. The `### Notes`
  section now names email, and the ADR keeps its original wording with an
  appended correction, because ADRs are append-only (`AGENTS.md` §9).
- Five documents claimed enforcement that does not exist, contradicting the rule
  in `docs/enterprise/README.md` that a rule which is not enforced automatically
  must say so explicitly. `.github/dependabot.yml` stated that Dependabot pull
  requests are exempt from the commit-message gate, while `ci.yaml` states the
  opposite and the exemption had in fact been dropped;
  `docs/enterprise/release-process.md` stated that the tag, artifact and publish
  steps are "automated by the release workflow", but no release workflow exists
  and no tag has ever been cut; `AGENTS.md` and
  `docs/enterprise/security-policy.md` both described `govulncheck` as running
  "on every PR touching Go code" when the job has no path filter; and
  `docs/enterprise/observability.md` stated that `KROOT_LOG_LEVEL` controls the
  log level when no Go file reads any `KROOT_*` variable. All five now describe
  what the repository actually does.
- The `vulnerabilities` gate passed for the wrong reason. `GOVULN_VERSION` was
  pinned to `v1.8.0`, which requires Go >= 1.26.0, while the module is pinned to
  Go 1.23.12. Under the default `GOTOOLCHAIN=auto`, `go install` silently
  downloaded Go 1.26 and built the scanner with it, so the compiler that actually
  performed the scan — locally and in CI — appeared nowhere in the repository.
  The scanner is now pinned to `v1.1.4`, the newest `x/vuln` release that builds
  with Go 1.23.12, and `GOTOOLCHAIN=local` is declared explicitly in both the
  `Makefile` and the CI workflow so an incompatible dependency fails loudly
  instead of quietly changing the compiler. The bump to `actions/setup-go@v7`,
  whose v6.0.0 release sets `GOTOOLCHAIN=local`, is what surfaced the drift.
- `make fmt` and `make fmt-check` did not enforce the format gate that
  `docs/enterprise/coding-standards.md` documents. Both called plain `gofmt`,
  while the enforced gate is golangci-lint's formatter set — gofmt, gofumpt and
  goimports. Measured on a function whose body opens with a blank line:
  `gofmt -l .` reports nothing, while `golangci-lint run` rejects the file with
  `File is not properly formatted (gofumpt)` and `unnecessary leading newline
  (whitespace)`. A developer following the documented fix — run `make fmt` —
  could therefore still fail the lint gate, and the `test` job's own `gofmt` step
  contradicted the `lint` job about what "formatted" means. `make fmt` and
  `make fmt-check` now run `golangci-lint fmt` and `golangci-lint fmt --diff`, and
  the duplicate `gofmt` step is gone so one tool defines it.
- The `Makefile` was never under version control. A `Makefile` rule in a
  developer's global ignore file (`core.excludesFile`, the qmake/Qt rule) matched
  it, so `git add .` skipped it silently and no commit ever contained it. A fresh
  clone therefore had no `make` entry point at all: `make ci` — the "local
  definition of done" in `AGENTS.md` §7 — did not exist, the install steps for
  the pinned `golangci-lint` and `govulncheck` did not exist, and every command in
  the README table was unavailable. CI stayed green only because the workflow
  reimplements each command inline instead of calling `make`, which is also why
  the two definitions of "done" could diverge unnoticed. The Makefile is now
  tracked, `.gitignore` re-includes it so a local ignore file cannot hide it
  again, and CI fails if it stops being tracked.
- The exit codes documented in `docs/enterprise/api-compatibility.md` were not
  implemented: every failure exited `1`, so the usage code `2` was unreachable
  and `kroot -h` reported a runtime failure while `kroot help` reported success —
  one request, two spellings, two answers. Scripts told to branch on the status
  could not tell "you invoked me wrong" from "it broke while running". `run` now
  wraps invocation failures in the `ErrUsage` sentinel, `flag.ErrHelp` returns
  success, and `exitCodeFor` maps `nil` to `0`, `ErrUsage` to `2` and everything
  else to `1`. `TestExitCodesMatchTheDocumentedContract` pins all three, and
  `TestRunHelpFlagIsNotAnError` fails against the previous `run`. Failure output
  goes to the matching stream: requested help and the `version` line go to
  stdout, while usage text from a failed invocation goes to stderr and a failed
  invocation writes nothing to stdout (`TestUsageFailuresGoToStderr`). The new
  `Streams` section of the same document records the contract.
- `commitlint.config.mjs` accepted only a subset of the Conventional Commits
  types, so a valid `style(...)` commit passed the local `commit-msg` hook but
  would have been rejected by the CI `commits` job. The vocabulary now matches
  the specification.
- The commitlint `subject-case` rule rejected legitimate mid-subject acronyms
  such as `CLI`, `HTTP` and `API`, contradicting the naming rules in
  `docs/enterprise/coding-standards.md`.
- `pnpm/action-setup` failed in CI with "Multiple versions of pnpm specified":
  the workflow declared `version: 9` while `package.json` declared
  `packageManager` (`pnpm@9.15.4`). The workflow no longer passes a version,
  leaving `package.json` as the single source of truth.
- A planned exemption from commit linting for Dependabot and Renovate was
  dropped before merge. It rested on the assumption that their generated
  subjects are sentence-case; measured against the real messages, both
  `ci(deps): bump actions/checkout from 4 to 7` and
  `build(deps-dev): bump @commitlint/cli from 19.8.1 to 21.2.3` pass the rules.
  An exemption granted for a non-existent problem would only have let genuinely
  malformed bot commits through unchallenged.
- `SECURITY.md` directed reporters to GitHub private vulnerability reporting,
  which GitHub does not provide on a private repository (the API returns 404
  without GitHub Advanced Security). Email is now the documented channel, with
  the GitHub flow described as the channel to enable if the repository becomes
  public.
- The `commits` job used a job-level `if`, so on a push it reported as skipped.
  A skipped job leaves its required status check expected forever and blocks
  every pull request once branch protection is enabled. The job now always runs
  and reports success on the paths where linting does not apply.

### Notes

- The project type is now decided: a terminal-first CLI application
  (`docs/adr/0003-cli-first-terminal-application.md`). The HTTP and persistence
  layers remain open and each still needs its own ADR; the module favours the Go
  standard library until one says otherwise.
- No public API, CLI flag or environment variable is guaranteed stable by this
  release or any `v0.x` release. The compatibility guarantees documented in
  `docs/enterprise/api-compatibility.md` take effect at `v1.0.0`, and while
  `MAJOR` is `0` the public surface may still change in a MINOR release. This
  note previously read "before the first `v0.1.0` tag", which described a tag
  that did not exist; the tag existing is not what makes the guarantees start.
- Vulnerability reports go to the maintainer's email, as `SECURITY.md` documents.
  GitHub's private vulnerability reporting is not available on a private
  repository without GitHub Advanced Security, so it is not a channel here; this
  note used to say otherwise. A project mailbox and domain can replace the
  fallback once one exists.
