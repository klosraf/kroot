# Changelog

All notable changes to Kroot are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial module `github.com/kroot/kroot` (Go 1.23.12).
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

### Fixed

- `commitlint.config.mjs` accepted only a subset of the Conventional Commits
  types, so a valid `style(...)` commit passed the local `commit-msg` hook but
  would have been rejected by the CI `commits` job. The vocabulary now matches
  the specification.
- The commitlint `subject-case` rule rejected legitimate mid-subject acronyms
  such as `CLI`, `HTTP` and `API`, contradicting the naming rules in
  `docs/enterprise/coding-standards.md`.

### Notes

- The project type (CLI / service / full-stack application) is not finalised;
  the module favours the Go standard library until an ADR says otherwise.
- No public API, CLI flag or environment variable is guaranteed stable before
  the first `v0.1.0` tag. The compatibility guarantees documented in
  `docs/enterprise/api-compatibility.md` take effect at `v1.0.0`.
- `SECURITY.md` names `security@kroot.dev` as the reporting address; confirm it
  before the first public release.
- No `LICENSE` file yet — choose one before publishing.

