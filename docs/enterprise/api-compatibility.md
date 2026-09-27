# API Compatibility — Kroot

This document defines what callers and operators may depend on, and what they
may not. It complements [`versioning-policy.md`](./versioning-policy.md), which
states *when* a change is breaking; this document states *what* a breaking change
is.

> **Current status:** pre-`v1.0.0`. Nothing in this table is guaranteed yet. The
> guarantees below take effect at `v1.0.0` and are recorded here so the intent is
> explicit before the first release.

## Guarantees from `v1.0.0`

### CLI

| Stable | Not stable |
|---|---|
| Subcommand names and flags documented in the manual | help text wording and layout |
| Exit codes: `0` success, `1` runtime failure, `2` usage error | progress output format |

Adding a subcommand or an optional flag is a MINOR change. Removing or renaming
one is a MAJOR change, and is preceded by a deprecation MINOR that prints a
warning naming the replacement.

### HTTP API

| Stable | Not stable |
|---|---|
| Path, method, and documented request/response fields | undocumented fields |
| Status code semantics | error message text |
| `Content-Type` and documented media types | header ordering, whitespace |

Rules:

- Additive within a MAJOR: new optional request fields, new response fields, new
  endpoints.
- A newly added response field is additive and safe; a *removed* or *retyped*
  field is breaking.
- Deprecations are announced one MINOR ahead with `Deprecation` and `Sunset`
  response headers, plus a documentation note.
- Unknown request fields are rejected once a schema exists, so typos fail loudly
  rather than silently doing nothing.

### Environment variables

| Stable | Not stable |
|---|---|
| Names of documented `KROOT_*` variables | internal variable names |
| Accepted value semantics | the exact text of validation error messages |

- A variable is never renamed or repurposed within a MAJOR.
- Every new variable ships with a documented default, so existing deployments
  keep working without configuration changes.
- A variable that becomes meaningless is deprecated for one MINOR, then removed
  at the next MAJOR.

### Exit codes

```
0  success
1  runtime failure (I/O, network, dependency, configuration rejected)
2  usage error (unknown flag, unknown command, bad argument)
```

Scripts may branch on these. Any new code is additive and documented.

### On-disk state (`~/.kroot`)

- Migrations are automatic and forward-compatible on upgrade.
- A migration that cannot be reversed requires a MAJOR bump, a documented
  rollback procedure, and a backup step in the release notes.
- Downgrades are supported for one MINOR back; older downgrades are not
  guaranteed.

### Log records

The message text is not an interface. The **structured keys** used by operators
for alerting and dashboards are, and change only with a MINOR bump and a
documented note.

## Explicitly not guaranteed

- Any endpoint or flag marked experimental in its documentation.
- Internal Go packages under `internal/` — importing them is unsupported and
  will break without notice.
- Performance characteristics, unless a benchmark is committed alongside the
  claim.
- Behaviour of undefined inputs: a malformed request may be rejected in any way,
  but it will not be silently accepted.

## Changing this document

A change to a guarantee is itself a breaking change and requires a MAJOR bump
plus an ADR explaining the migration path.
