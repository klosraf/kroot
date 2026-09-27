# Changelog

All notable changes to Kroot are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

- Module path corrected from `github.com/kroot/kroot` to
  `github.com/klosraf/kroot`, matching the GitHub account that owns the
  repository.
- `docs/enterprise/branching-strategy.md` now documents the branch protection
  actually applied to `main`, including why the required approval count is 0
  while the repository has a single maintainer.

### Fixed

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

- The project type (CLI / service / full-stack application) is not finalised;
  the module favours the Go standard library until an ADR says otherwise.
- No public API, CLI flag or environment variable is guaranteed stable before
  the first `v0.1.0` tag. The compatibility guarantees documented in
  `docs/enterprise/api-compatibility.md` take effect at `v1.0.0`.
- Vulnerability reports go to the maintainer's email, as `SECURITY.md` documents.
  GitHub's private vulnerability reporting is not available on a private
  repository without GitHub Advanced Security, so it is not a channel here; this
  note used to say otherwise. A project mailbox and domain can replace the
  fallback once one exists.


