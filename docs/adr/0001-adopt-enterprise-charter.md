# 0001 — Adopt the enterprise engineering charter and freeze the stack

- **Status:** accepted
- **Date:** 2026-09-26
- **Deciders:** repository maintainers

## Context

Kroot is a new repository. Its scope, standards and technology choices must be
decided before the first line of domain code is written, because retrofitting
engineering discipline onto a growing codebase costs more than establishing it at
the start.

The repository sits alongside `kodev`, which already operates under an explicit
enterprise charter: a frozen stack, enforced quality gates, ADR-recorded
decisions, and policies for branching, versioning, releasing, security and
observability. Diverging from that model without a reason would produce two
repositories with two different definitions of "done".

The initial module (`github.com/klosraf/kroot`, Go 1.23.12) compiles and passes its
tests, but has no governance, no CI and no declared standards.

## Decision

1. **Adopt `AGENTS.md` as the binding engineering charter** for this repository,
   mirroring the structure and rigour of `kodev`'s charter.
2. **Adopt the policy set in `docs/enterprise/`**: coding standards, testing
   strategy, branching strategy, versioning policy, release process, security
   policy, observability and API compatibility.
3. **Freeze the toolchain** at Go 1.23.12 with the standard library as the
   default dependency, TypeScript (strict) + React 18 + Vite + Tailwind for the
   frontend, and pnpm as the only JavaScript package manager.
4. **Record every subsequent architectural decision as an ADR**, append-only, in
   `docs/adr/NNNN-title.md`.
5. **Adopt Conventional Commits with a scope**, validated by commitlint, and
   trunk-based development with short-lived branches.
6. **Make the gates executable, not aspirational**: `make ci` locally and a
   GitHub Actions workflow in CI enforce formatting, vet, lint, race tests,
   coverage and vulnerability scanning.
7. **Pin developer tooling per repository** (`golangci-lint` v2.14.0 and
   `govulncheck` installed into `./bin`) so that local runs and CI runs cannot
   drift apart silently.

## Consequences

**Positive**

- One definition of "done" exists, and it is executable.
- Decisions are traceable: an ADR explains why, and the charter says what.
- New contributors get a single entry point (`AGENTS.md` → `CONTRIBUTING.md` →
  `docs/enterprise/`).
- Drift between local and CI behaviour is caught by pinned tool versions.

**Negative / costs**

- New dependencies must be justified in writing, which slows the addition of a
  library that would have been convenient.
- The gate set (format, vet, lint, race, vuln) makes a PR slower to land than a
  one-line commit would be.
- Pinned tooling must be bumped deliberately, in its own PR, and the bump must be
  reflected in CI and in `Makefile`.

**Neutral**

- The project type is still undecided. The HTTP layer and persistence choices are
  deliberately left open and will each receive their own ADR; this charter
  constrains *how* those decisions are made, not *which* is made.

## Notes

- `v0.x` releases may change the public surface in a MINOR release; the
  compatibility guarantees in `docs/enterprise/api-compatibility.md` take effect
  at `v1.0.0`.
- The module path is `github.com/klosraf/kroot`, matching the GitHub account that
  owns the repository. Renaming it later requires updating `go.mod` and every
  internal import, so it was settled before domain code existed.
- Vulnerability reports arrive through GitHub private vulnerability reporting,
  with the maintainer email as a fallback. A project mailbox and domain can
  replace the fallback once one is established.
- **Correction (2026-09-27), appended rather than edited in — this ADR is
  append-only per `AGENTS.md` §9.** The note above describes a channel that does
  not exist: GitHub does not provide private vulnerability reporting for a
  private repository without GitHub Advanced Security (the API returns 404), so a
  reporter following it would look for a button that is not there. The channel is
  the maintainer's email, as `SECURITY.md` documents, and email is the *primary*
  channel rather than a fallback. The original note stays exactly as written,
  because the record of a decision that turned out wrong is itself part of the
  record.
- The licence is MIT, consistent with the sibling `kodev` project. Changing it
  later is a single-file change, but relicensing after third-party contributions
  requires contributor agreement, so it was decided now.
