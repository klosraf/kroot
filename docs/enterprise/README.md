# Kroot — Enterprise Documentation Index

This folder is the single source of truth for running Kroot as an
enterprise-grade product: how work is branched, versioned, released, secured and
observed. It is written for maintainers and operators.

The engineering charter that governs day-to-day contribution lives in
[`AGENTS.md`](../../AGENTS.md).

| Document | Purpose |
|---|---|
| [`coding-standards.md`](./coding-standards.md) | Go and TypeScript rules, naming, error handling |
| [`testing-strategy.md`](./testing-strategy.md) | Test pyramid, coverage, race detection, E2E |
| [`branching-strategy.md`](./branching-strategy.md) | Trunk-based model, branch naming, PR rules |
| [`versioning-policy.md`](./versioning-policy.md) | What bumps major / minor / patch |
| [`release-process.md`](./release-process.md) | Release checklist, tags, artifacts, rollback |
| [`security-policy.md`](./security-policy.md) | Threat model, handling, hardening checklist |
| [`observability.md`](./observability.md) | Logs, metrics, health checks, SLOs |
| [`api-compatibility.md`](./api-compatibility.md) | Compatibility guarantees for CLI, HTTP and env vars |

## Document rules

- These documents describe **intent**, and they are enforced by tooling where
  possible. If a rule is not enforced automatically, say so explicitly.
- A change to a policy is itself a PR, reviewed like code.
- Policies are versioned with the repository; the version on `main` is the
  authoritative one for the current development line.
