# Changelog

All notable changes to Kroot are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

### Fixed

- `commitlint.config.mjs` accepted only a subset of the Conventional Commits
  types, so a valid `style(...)` commit passed the local `commit-msg` hook but
  would have been rejected by the CI `commits` job. The vocabulary now matches
  the specification.
- The commitlint `subject-case` rule rejected legitimate mid-subject acronyms
  such as `CLI`, `HTTP` and `API`, contradicting the naming rules in
  `docs/enterprise/coding-standards.md`.
- The `commits` CI job no longer runs for Dependabot and Renovate pull requests,
  whose generated sentence-case subjects would fail the `subject-case` rule by
  construction.
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
- Vulnerability reports are handled through GitHub private vulnerability
  reporting, with the maintainer's email as a fallback. A project mailbox and
  domain can replace the fallback once one exists.


